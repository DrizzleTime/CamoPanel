package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/files"
	"camopanel/backend/internal/host"
	"camopanel/backend/internal/nginx"
	"camopanel/backend/internal/site"
	"camopanel/backend/internal/store"
	"github.com/gin-gonic/gin"
)

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)
var idPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func failure(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "服务暂时不可用"
	var dockerErr *docker.Error
	if errors.As(err, &dockerErr) {
		status, message = dockerErr.Status, dockerErr.Message
	}
	c.JSON(status, gin.H{"error": gin.H{"message": message}})
}
func badRequest(c *gin.Context, message string) {
	c.JSON(400, gin.H{"error": gin.H{"message": message}})
}

func New(cfg config.Config, service docker.Service) *gin.Engine {
	if cfg.DataDir == "" {
		if dir, err := os.MkdirTemp("", "camopanel-"); err == nil {
			cfg.DataDir = dir
		}
	}
	dep := openPlatform(cfg, service)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Header("X-Frame-Options", "DENY")
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	})
	r.GET("/api/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "service": "camopanel"}) })
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		if origin := c.GetHeader("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != c.Request.Host {
				c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"message": "请求来源不匹配"}})
				return
			}
		}
		dep.gate(c)
		if c.IsAborted() {
			dep.auditMutating(c)
			return
		}
		bodyLimit := int64(1 << 20)
		timeout := 45 * time.Second
		if strings.HasPrefix(c.Request.URL.Path, "/api/v1/files") {
			timeout = 10 * time.Minute
			if c.Request.URL.Path == "/api/v1/files/upload" && strings.HasPrefix(c.ContentType(), "multipart/") {
				bodyLimit = files.MaxUploadSize + (1 << 20)
			} else if c.Request.URL.Path == "/api/v1/files" && c.Request.Method == http.MethodPut || c.Request.URL.Path == "/api/v1/files/upload" {
				bodyLimit = files.MaxTextSize*6 + 1024
			}
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit)
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		dep.auditMutating(c)
	})
	mountHost(api, host.New())
	api.GET("/docker/overview", func(c *gin.Context) {
		s, err := service.Snapshot(c.Request.Context())
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, s)
	})
	api.POST("/docker/containers", dep.createContainer)
	api.POST("/docker/containers/:id/:action", func(c *gin.Context) {
		id, action := c.Param("id"), c.Param("action")
		if !idPattern.MatchString(id) {
			badRequest(c, "容器 ID 无效")
			return
		}
		switch action {
		case "start", "stop", "restart", "remove", "pause", "unpause":
		default:
			badRequest(c, "不支持的容器操作")
			return
		}
		if err := service.Action(c.Request.Context(), id, action); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"message": "操作完成", "at": time.Now().UTC().Format(time.RFC3339)})
	})
	api.GET("/docker/containers/:id/logs", func(c *gin.Context) {
		if !idPattern.MatchString(c.Param("id")) {
			badRequest(c, "容器 ID 无效")
			return
		}
		logs, err := service.Logs(c.Request.Context(), c.Param("id"))
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"logs": logs})
	})

	// Initialize Nginx supervisor, SSL manager, site service, and store service
	nginxSup, _ := nginx.NewSupervisor(cfg.DataDir, service, cfg.NginxPort, cfg.NginxSSLPort)
	if nginxSup != nil {
		_ = nginxSup.EnsureRunning(context.Background())
	}
	sslMgr := site.NewSSLManager(cfg.DataDir, cfg.ACMEEmail)
	sslMgr.SetDirectory(cfg.ACMEDirectory)
	siteSvc, _ := site.NewService(cfg.DataDir, nginxSup, sslMgr, service)
	storeSvc, _ := store.NewService(cfg.DataDir, service)

	// App Store Endpoints
	api.GET("/store/apps", func(c *gin.Context) {
		catalog, err := storeSvc.ListCatalog(c.Request.Context())
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, catalog)
	})
	api.GET("/store/apps/:id", func(c *gin.Context) {
		app, err := storeSvc.GetCatalogApp(c.Param("id"))
		if err != nil {
			c.JSON(404, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(200, app)
	})
	api.GET("/store/installed", func(c *gin.Context) {
		list, err := storeSvc.ListInstalled(c.Request.Context())
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, list)
	})
	api.GET("/store/installed/:id", func(c *gin.Context) {
		app, err := storeSvc.GetInstalled(c.Param("id"))
		if err != nil {
			c.JSON(404, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(200, app)
	})
	api.POST("/store/installed", func(c *gin.Context) {
		var req store.InstallRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, "请填写完整应用安装信息")
			return
		}
		installed, err := storeSvc.Install(c.Request.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "已经安装") {
				c.JSON(409, gin.H{"error": gin.H{"message": err.Error()}})
				return
			}
			if strings.Contains(err.Error(), "安全约束拦截") || strings.Contains(err.Error(), "不合法") {
				badRequest(c, err.Error())
				return
			}
			failure(c, err)
			return
		}
		c.JSON(201, installed)
	})
	api.POST("/store/installed/:id/action", func(c *gin.Context) {
		var req store.ActionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, "操作类型无效")
			return
		}
		if err := storeSvc.Action(c.Request.Context(), c.Param("id"), req.Action, req.DeleteData); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "操作完成", "action": req.Action})
	})
	api.GET("/store/installed/:id/compose", func(c *gin.Context) {
		compose, err := storeSvc.GetCompose(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"compose": compose})
	})
	api.PUT("/store/installed/:id/compose", func(c *gin.Context) {
		var body struct {
			Compose string `json:"compose" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "Compose 内容不能为空")
			return
		}
		if err := storeSvc.UpdateCompose(c.Request.Context(), c.Param("id"), body.Compose); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "Compose 配置已更新"})
	})
	api.GET("/store/installed/:id/logs", func(c *gin.Context) {
		logs, err := storeSvc.GetLogs(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"logs": logs})
	})

	// Websites Endpoints
	api.GET("/sites", func(c *gin.Context) {
		sites, err := siteSvc.List(c.Request.Context())
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, sites)
	})
	api.GET("/sites/:id", func(c *gin.Context) {
		s, err := siteSvc.Get(c.Request.Context(), c.Param("id"))
		if err != nil {
			c.JSON(404, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(200, s)
	})
	api.POST("/sites", func(c *gin.Context) {
		var req site.CreateSiteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, "请填写网站名称、主域名和网站类型")
			return
		}
		s, err := siteSvc.Create(c.Request.Context(), req)
		if err != nil {
			if strings.Contains(err.Error(), "已被网站") {
				c.JSON(409, gin.H{"error": gin.H{"message": err.Error()}})
				return
			}
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, s)
	})
	api.PUT("/sites/:id", func(c *gin.Context) {
		var req site.UpdateSiteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, "请检查修改参数")
			return
		}
		s, err := siteSvc.Update(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, s)
	})
	api.DELETE("/sites/:id", func(c *gin.Context) {
		if err := siteSvc.Delete(c.Request.Context(), c.Param("id")); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "网站已删除"})
	})
	api.POST("/sites/:id/toggle", func(c *gin.Context) {
		s, err := siteSvc.Toggle(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, s)
	})
	api.POST("/sites/:id/ssl/apply", func(c *gin.Context) {
		var req site.SSLApplyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			badRequest(c, "请指定证书申请方式 (letsencrypt / self-signed / custom)")
			return
		}
		s, err := siteSvc.ApplySSL(c.Request.Context(), c.Param("id"), req)
		if err != nil {
			failure(c, err)
			return
		}
		dep.importAppliedCert(s)
		c.JSON(200, s)
	})
	api.GET("/sites/:id/ssl", func(c *gin.Context) {
		info, err := siteSvc.GetSSLInfo(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, info)
	})
	api.GET("/sites/:id/config", func(c *gin.Context) {
		config, err := siteSvc.GetNginxConfig(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"config": config})
	})
	api.GET("/sites/:id/files/:filename", func(c *gin.Context) {
		content, err := siteSvc.GetStaticFile(c.Request.Context(), c.Param("id"), c.Param("filename"))
		if err != nil {
			c.JSON(404, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(200, gin.H{"filename": c.Param("filename"), "content": content})
	})
	api.PUT("/sites/:id/files/:filename", func(c *gin.Context) {
		var body struct {
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "文件内容不能为空")
			return
		}
		if err := siteSvc.SaveStaticFile(c.Request.Context(), c.Param("id"), c.Param("filename"), body.Content); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "文件已保存"})
	})

	// Built-in Nginx Supervisor Endpoints
	api.GET("/nginx/status", func(c *gin.Context) {
		if nginxSup == nil {
			c.JSON(200, gin.H{"installed": false, "running": false})
			return
		}
		status := nginxSup.GetStatus(c.Request.Context())
		c.JSON(200, status)
	})
	api.POST("/nginx/action", func(c *gin.Context) {
		var body struct {
			Action string `json:"action" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "操作名称不能为空")
			return
		}
		if nginxSup == nil {
			badRequest(c, "Nginx 服务未初始化")
			return
		}
		if err := nginxSup.Action(c.Request.Context(), body.Action); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"message": "Nginx 操作完成", "action": body.Action})
	})
	dep.finish(api, siteSvc, nginxSup, storeSvc)
	if cfg.WebDir != "" {
		root, _ := filepath.Abs(cfg.WebDir)
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") || (c.Request.Method != "GET" && c.Request.Method != "HEAD") {
				c.JSON(404, gin.H{"error": gin.H{"message": "接口不存在"}})
				return
			}
			path := filepath.Join(root, filepath.Clean("/"+c.Request.URL.Path))
			if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
				c.File(path)
				return
			}
			if filepath.Ext(c.Request.URL.Path) != "" {
				c.Status(404)
				return
			}
			c.File(filepath.Join(root, "index.html"))
		})
	}
	return r
}
