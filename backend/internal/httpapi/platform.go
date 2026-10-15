package httpapi

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"

	"camopanel/backend/internal/alert"
	"camopanel/backend/internal/assistant"
	"camopanel/backend/internal/audit"
	"camopanel/backend/internal/auth"
	"camopanel/backend/internal/backup"
	"camopanel/backend/internal/certs"
	"camopanel/backend/internal/config"
	"camopanel/backend/internal/cron"
	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/files"
	"camopanel/backend/internal/firewall"
	"camopanel/backend/internal/host"
	"camopanel/backend/internal/nginx"
	"camopanel/backend/internal/site"
	"camopanel/backend/internal/store"
	"github.com/gin-gonic/gin"
)

type platform struct {
	cfg       config.Config
	docker    docker.Service
	host      host.Sampler
	auth      *auth.Store
	audit     *audit.Log
	certs     *certs.Store
	files     *files.Manager
	downloads *fileDownloads
	backups   *backup.Store
	cron      *cron.Scheduler
	firewall  *firewall.Manager
	alerts    *alert.Manager
	sites     *site.Service
	nginx     *nginx.Supervisor
	store     *store.Service
	terms     string
}

func openPlatform(cfg config.Config, service docker.Service) *platform {
	p := &platform{cfg: cfg, docker: service, host: host.New(), alerts: alert.New(), terms: "", downloads: newFileDownloads()}
	if cfg.DataDir == "" {
		return p
	}
	p.terms = cfg.DataDir + "/terminals.jsonl"
	p.auth, _ = auth.OpenWithCredentials(cfg.DataDir, cfg.AdminUsername, cfg.AdminPassword)
	p.audit, _ = audit.Open(cfg.DataDir)
	p.certs, _ = certs.Open(cfg.DataDir)
	p.files, _ = files.NewWithServerRoot(cfg.DataDir, cfg.FilesRoot)
	p.backups, _ = backup.Open(cfg.DataDir, 5)
	p.cron, _ = cron.Open(cfg.DataDir)
	p.firewall = firewall.NewSystem(cfg.PanelPort())
	return p
}

func (p *platform) gate(c *gin.Context) {
	if c.Request.Method == http.MethodGet && c.Request.URL.Path == "/api/v1/files/download" && c.Query("ticket") != "" {
		claim, ok := p.downloads.take(c.Query("ticket"))
		if !ok {
			c.AbortWithStatusJSON(410, gin.H{"error": gin.H{"message": "下载链接已过期，请重新下载"}})
			return
		}
		c.Set("file_download_root", claim.root)
		c.Set("file_download_path", claim.path)
		c.Set("actor", claim.actor)
		c.Set("role", claim.role)
		return
	}
	if isPublic(c) {
		c.Set("actor", "anonymous")
		c.Set("role", "admin")
		return
	}
	if p.cfg.Token != "" && c.GetHeader("Authorization") != "" {
		if subtle.ConstantTimeCompare([]byte(c.GetHeader("Authorization")), []byte("Bearer "+p.cfg.Token)) == 1 {
			c.Set("actor", "token")
			c.Set("role", "admin")
			p.rejectViewer(c)
			return
		}
		c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"message": "请输入正确的 API 访问令牌"}})
		return
	}
	if p.auth != nil {
		if cookie, err := c.Cookie(auth.SessionCookie); err == nil {
			if sess, ok := p.auth.Lookup(cookie); ok {
				c.Set("actor", sess.Username)
				c.Set("role", sess.Role)
				p.rejectViewer(c)
				return
			}
		}
	}
	if p.cfg.Token == "" && p.cfg.AdminPassword == "" && (p.auth == nil || !p.auth.HasAdmin()) {
		c.Set("actor", "local")
		c.Set("role", "admin")
		return
	}
	c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"message": "请先登录"}})
}

func (p *platform) rejectViewer(c *gin.Context) {
	if c.GetString("role") != "viewer" {
		return
	}
	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return
	}
	path := c.Request.URL.Path
	if c.Request.Method == http.MethodPost && (path == "/api/v1/auth/logout" || path == "/api/v1/auth/login" || path == "/api/v1/ai/chat") {
		return
	}
	c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"message": "只读账号不能执行此操作"}})
}

func isPublic(c *gin.Context) bool {
	path := c.Request.URL.Path
	if c.Request.Method == http.MethodGet && path == "/api/v1/auth/status" {
		return true
	}
	if c.Request.Method == http.MethodPost && (path == "/api/v1/auth/login" || path == "/api/v1/auth/bootstrap") {
		return true
	}
	return c.Request.Method == http.MethodGet && path == "/api/v1/settings/exposure"
}

func (p *platform) auditMutating(c *gin.Context) {
	if p.audit == nil {
		return
	}
	switch c.Request.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return
	}
	actor := c.GetString("actor")
	if actor == "" {
		actor = "anonymous"
	}
	status := c.Writer.Status()
	if status == 0 {
		status = 200
	}
	_ = p.audit.Append(audit.Record{Actor: actor, Method: c.Request.Method, Path: c.Request.URL.Path, Status: status, Detail: c.GetString("audit_detail")})
}

func (p *platform) secureCookie() bool {
	return p.cfg.TLSCert != "" && p.cfg.TLSKey != ""
}

func (p *platform) mountAuth(api *gin.RouterGroup) {
	api.GET("/auth/status", func(c *gin.Context) {
		source := "unconfigured"
		configured := p.auth != nil && p.auth.HasAdmin()
		if p.cfg.AdminPassword != "" {
			source = "environment"
		} else if configured {
			source = "legacy"
		} else if p.cfg.Token != "" {
			source = "token"
		}
		totp := p.auth != nil && p.auth.TOTPEnabled()
		c.JSON(200, gin.H{"configured": configured, "source": source, "totp_enabled": totp})
	})
	api.POST("/auth/bootstrap", func(c *gin.Context) {
		if p.cfg.AdminPassword != "" {
			c.JSON(409, gin.H{"error": gin.H{"message": auth.ErrEnvironment.Error()}})
			return
		}
		if p.auth == nil {
			failure(c, errString("认证存储不可用"))
			return
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Password) < 8 {
			badRequest(c, "请设置至少 8 位的管理员密码")
			return
		}
		if err := p.auth.Bootstrap(body.Password); err != nil {
			c.JSON(409, gin.H{"error": gin.H{"message": "管理员已初始化"}})
			return
		}
		c.JSON(201, gin.H{"username": "admin"})
	})
	api.POST("/auth/login", func(c *gin.Context) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Code     string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || p.auth == nil {
			c.JSON(401, gin.H{"error": gin.H{"message": "用户名或密码错误"}})
			return
		}
		sess, err := p.auth.Login(body.Username, body.Password, body.Code, timeNow())
		if err != nil {
			c.JSON(401, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.Set("actor", sess.Username)
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(auth.SessionCookie, sess.ID, 86400, "/", "", p.secureCookie(), true)
		c.JSON(200, gin.H{"username": sess.Username, "role": sess.Role, "expires_at": sess.ExpiresAt})
	})
	api.POST("/auth/logout", func(c *gin.Context) {
		if p.auth != nil {
			if cookie, err := c.Cookie(auth.SessionCookie); err == nil {
				p.auth.Logout(cookie)
			}
		}
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(auth.SessionCookie, "", -1, "/", "", p.secureCookie(), true)
		c.JSON(200, gin.H{"message": "已退出"})
	})
	api.POST("/auth/password", func(c *gin.Context) {
		if p.cfg.AdminPassword != "" {
			c.JSON(409, gin.H{"error": gin.H{"message": auth.ErrEnvironment.Error()}})
			return
		}
		var body struct {
			Current string `json:"current"`
			Next    string `json:"next"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Next) < 8 || p.auth == nil {
			badRequest(c, "请填写当前密码和至少 8 位的新密码")
			return
		}
		if err := p.auth.ChangePassword(c.GetString("actor"), body.Current, body.Next); err != nil {
			c.JSON(401, gin.H{"error": gin.H{"message": "用户名或密码错误"}})
			return
		}
		c.JSON(200, gin.H{"message": "密码已更新"})
	})
	api.POST("/auth/viewer", func(c *gin.Context) {
		if p.cfg.AdminPassword != "" {
			c.JSON(409, gin.H{"error": gin.H{"message": auth.ErrEnvironment.Error()}})
			return
		}
		var body struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Password) < 8 || p.auth == nil {
			badRequest(c, "请设置至少 8 位的只读密码")
			return
		}
		if err := p.auth.SetViewer(body.Password); err != nil {
			failure(c, err)
			return
		}
		c.JSON(201, gin.H{"username": "viewer", "role": "viewer"})
	})
	api.POST("/auth/totp/begin", func(c *gin.Context) {
		if p.auth == nil {
			failure(c, errString("认证存储不可用"))
			return
		}
		secret, err := p.auth.BeginTOTP()
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"secret": secret})
	})
	api.POST("/auth/totp/enable", func(c *gin.Context) {
		var body struct {
			Code string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || p.auth == nil {
			badRequest(c, "请填写验证码")
			return
		}
		if err := p.auth.EnableTOTP(body.Code, timeNow()); err != nil {
			c.JSON(401, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(200, gin.H{"enabled": true})
	})
	api.GET("/auth/session", func(c *gin.Context) {
		c.JSON(200, gin.H{"actor": c.GetString("actor"), "role": c.GetString("role")})
	})
	api.GET("/settings/exposure", func(c *gin.Context) {
		httpsOn := p.secureCookie()
		message := "公网暴露面板需要 HTTPS"
		if httpsOn {
			message = "面板正在使用 HTTPS"
		}
		c.JSON(200, gin.H{"https": httpsOn, "message": message})
	})
}

func (p *platform) finish(api *gin.RouterGroup, sites *site.Service, nginxSup *nginx.Supervisor, storeSvc *store.Service) {
	p.sites = sites
	p.nginx = nginxSup
	p.store = storeSvc
	if p.cron != nil {
		p.cron.Register("backup_site", func(_ context.Context, job cron.Job) (string, error) {
			if p.backups == nil {
				return "", errString("备份不可用")
			}
			item, err := p.backups.CreateSite(job.Payload["site_id"])
			if err != nil {
				return "", err
			}
			return item.ID, nil
		})
		p.cron.Register("backup_app", func(_ context.Context, job cron.Job) (string, error) {
			item, err := p.backups.CreateDir("apps/" + job.Payload["app"])
			if err != nil {
				return "", err
			}
			return item.ID, nil
		})
		p.cron.Register("backup_database", func(_ context.Context, _ cron.Job) (string, error) {
			return "", errString("数据库控制台尚未接入真实数据库，无法执行备份")
		})
		p.cron.Register("backup_directory", func(_ context.Context, job cron.Job) (string, error) {
			item, err := p.backups.CreateDir(job.Payload["path"])
			if err != nil {
				return "", err
			}
			return item.ID, nil
		})
		if p.certs != nil {
			p.certs.SetHTTP01(sites.RenewHTTP01)
		}
		p.cron.Register("renew_certs", func(ctx context.Context, _ cron.Job) (string, error) {
			attempts, err := p.certs.RenewDue(ctx, timeNow())
			if err != nil {
				return "", err
			}
			if len(attempts) == 0 {
				return "没有到期证书", nil
			}
			return attempts[0].Error, nil
		})
		p.cron.Register("rotate_logs", func(_ context.Context, _ cron.Job) (string, error) {
			return rotateLogs(p.cfg.DataDir)
		})
		p.cron.Register("fetch_url", func(ctx context.Context, job cron.Job) (string, error) {
			return fetchURL(ctx, job.Payload["url"])
		})
		p.cron.Start()
	}
	p.mountAuth(api)
	p.mountCerts(api)
	p.mountFiles(api)
	p.mountDB(api)
	p.mountBackups(api)
	p.mountCron(api)
	p.mountDockerExtra(api)
	p.mountSecurity(api)
	p.mountAI(api)
	api.GET("/sites/:id/logs", func(c *gin.Context) {
		access, errLog, err := p.sites.ReadLogs(c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"access_log": access, "error_log": errLog})
	})
	api.POST("/sites/:id/certificate", func(c *gin.Context) {
		var body struct {
			CertID string `json:"cert_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.CertID == "" {
			badRequest(c, "请指定证书")
			return
		}
		cert, err := p.certs.Get(body.CertID)
		if err != nil {
			badRequest(c, "证书不存在")
			return
		}
		s, err := p.sites.AttachCert(c.Request.Context(), c.Param("id"), cert.ID, cert.Expiry, cert.Provider)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, s)
	})
}

type errText string

func (e errText) Error() string { return string(e) }

func errString(message string) error { return errText(message) }

func (p *platform) importAppliedCert(s *site.Site) {
	if p.certs == nil || s == nil || p.cfg.DataDir == "" {
		return
	}
	certPEM, err1 := os.ReadFile(p.cfg.DataDir + "/ssl/" + s.ID + "/fullchain.pem")
	keyPEM, err2 := os.ReadFile(p.cfg.DataDir + "/ssl/" + s.ID + "/privkey.pem")
	if err1 != nil || err2 != nil {
		return
	}
	expiry := timeNow()
	if parsed, err := parseTime(s.CertExpiry); err == nil {
		expiry = parsed
	}
	_, _ = p.certs.SaveIssued(s.SSLProvider, s.SSLProvider, append([]string{s.PrimaryDomain}, s.Domains...), expiry, string(certPEM), string(keyPEM))
}

func actorTools(p *platform, c *gin.Context) assistant.Tools {
	return assistant.Tools{
		Host: p.host, Docker: p.docker, Sites: p.sites, Certs: p.certs, Store: p.store,
		Cron: p.cron, Nginx: p.nginx, Audit: p.audit, Actor: c.GetString("actor"), Role: c.GetString("role"),
	}
}

func validRestart(policy string) bool {
	switch policy {
	case "", "no", "always", "on-failure", "unless-stopped":
		return true
	default:
		return false
	}
}
