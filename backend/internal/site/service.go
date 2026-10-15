package site

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/nginx"
)

var validSiteNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{1,62}$`)

type Service struct {
	mu         sync.RWMutex
	dataDir    string
	supervisor *nginx.Supervisor
	sslManager *SSLManager
	docker     docker.Service
	sites      map[string]*Site
}

func NewService(dataDir string, supervisor *nginx.Supervisor, sslManager *SSLManager, boxes docker.Service) (*Service, error) {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	s := &Service{
		dataDir:    absDataDir,
		supervisor: supervisor,
		sslManager: sslManager,
		docker:     boxes,
		sites:      make(map[string]*Site),
	}
	_ = s.loadPersisted()
	return s, nil
}

func (s *Service) metaFilePath() string {
	return filepath.Join(s.dataDir, "sites.json")
}

func (s *Service) loadPersisted() error {
	path := s.metaFilePath()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []*Site
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, site := range list {
		s.sites[site.ID] = site
	}
	return nil
}

func (s *Service) savePersisted() error {
	list := make([]*Site, 0, len(s.sites))
	for _, site := range s.sites {
		list = append(list, site)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaFilePath(), b, 0644)
}

func (s *Service) List(ctx context.Context) ([]*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]*Site, 0, len(s.sites))
	for _, site := range s.sites {
		fillDays(site)
		list = append(list, site)
	}
	return list, nil
}

func (s *Service) Get(ctx context.Context, id string) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[id]
	if !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}
	fillDays(site)
	return site, nil
}

func (s *Service) Create(ctx context.Context, req CreateSiteRequest) (*Site, error) {
	if !validSiteNamePattern.MatchString(req.Name) {
		return nil, fmt.Errorf("网站名称格式不正确，须为 2-63 位字母、数字、点、横线或下划线")
	}
	req.PrimaryDomain = strings.TrimSpace(req.PrimaryDomain)
	if req.PrimaryDomain == "" {
		return nil, fmt.Errorf("主域名不能为空")
	}
	if err := prepareSiteRequest(req.Type, req.Runtime, req.ProxyTarget, req.CustomConfig); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check domain conflict
	for _, exist := range s.sites {
		if exist.PrimaryDomain == req.PrimaryDomain {
			return nil, fmt.Errorf("主域名 [%s] 已被网站 [%s] 绑定", req.PrimaryDomain, exist.Name)
		}
	}

	var randBytes [4]byte
	_, _ = rand.Read(randBytes[:])
	siteID := fmt.Sprintf("%s-%s", req.Name, hex.EncodeToString(randBytes[:]))

	now := time.Now().UTC().Format(time.RFC3339)
	site := &Site{
		ID:            siteID,
		Name:          req.Name,
		PrimaryDomain: req.PrimaryDomain,
		Domains:       req.Domains,
		Type:          req.Type,
		RootPath:      fmt.Sprintf("/var/www/sites/%s", siteID),
		ProxyTarget:   req.ProxyTarget,
		SSLEnabled:    false,
		ForceHTTPS:    req.ForceHTTPS,
		Status:        "running",
		CustomConfig:  req.CustomConfig,
		Redirects:     req.Redirects,
		Locations:     req.Locations,
		BasicAuthUser: req.BasicAuthUser,
		Hotlink:       req.Hotlink,
		RateLimit:     req.RateLimit,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if req.Type == "runtime" {
		site.Runtime = req.Runtime
		site.ProxyTarget = fmt.Sprintf("http://runtime-%s-%s:8080", req.Runtime, siteID)
		if err := s.ensureRuntime(ctx, fmt.Sprintf("runtime-%s-%s", req.Runtime, siteID)); err != nil {
			return nil, err
		}
	}
	if req.BasicAuthUser != "" && req.BasicAuthPassword != "" {
		site.BasicAuthFile = s.writeBasicAuth(siteID, req.BasicAuthUser, req.BasicAuthPassword)
	}
	s.writeSiteLogs(siteID)

	// If static site, create default directory and index.html
	if site.Type == "static" {
		siteDir := filepath.Join(s.dataDir, "sites", siteID)
		if err := os.MkdirAll(siteDir, 0755); err != nil {
			return nil, err
		}
		indexPath := filepath.Join(siteDir, "index.html")
		defaultHTML := fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - CamoPanel 托管网站</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: grid; place-items: center; min-height: 90vh; background: #f8fafc; color: #0f172a; margin: 0; padding: 20px; }
        .card { background: white; border-radius: 12px; padding: 40px; max-width: 540px; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.06); text-align: center; }
        h1 { color: #d85c49; margin-top: 0; font-size: 26px; }
        p { color: #475569; line-height: 1.6; }
        .tag { display: inline-block; background: #f1f5f9; padding: 4px 12px; border-radius: 9999px; font-size: 13px; color: #334155; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="card">
        <span class="tag">静态站点已创建</span>
        <h1>欢迎访问 %s</h1>
        <p>您的静态网站已通过 CamoPanel 成功部署并由内置 Nginx 托管。</p>
        <p>您可以随时在控制面板中编辑 <code>index.html</code> 或上传您自己的静态网页文件。</p>
    </div>
</body>
</html>`, req.Name, req.PrimaryDomain)
		_ = os.WriteFile(indexPath, []byte(defaultHTML), 0644)
	}

	// Write Nginx server block
	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", siteID))
	nginxConf := GenerateNginxConfig(site)
	if err := os.WriteFile(confPath, []byte(nginxConf), 0644); err != nil {
		return nil, fmt.Errorf("写入 Nginx 配置文件失败: %w", err)
	}

	// Reload Nginx
	if s.supervisor != nil {
		_ = s.supervisor.EnsureRunning(ctx)
		_ = s.supervisor.Reload(ctx)
	}

	s.sites[siteID] = site
	_ = s.savePersisted()

	return site, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateSiteRequest) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[id]
	if !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}
	if err := prepareSiteRequest(req.Type, req.Runtime, req.ProxyTarget, req.CustomConfig); err != nil {
		return nil, err
	}

	site.Name = req.Name
	site.PrimaryDomain = req.PrimaryDomain
	site.Domains = req.Domains
	site.Type = req.Type
	site.Runtime = req.Runtime
	site.ProxyTarget = req.ProxyTarget
	if req.Type == "runtime" {
		site.ProxyTarget = fmt.Sprintf("http://runtime-%s-%s:8080", req.Runtime, id)
		if err := s.ensureRuntime(ctx, fmt.Sprintf("runtime-%s-%s", req.Runtime, id)); err != nil {
			return nil, err
		}
	}
	site.ForceHTTPS = req.ForceHTTPS
	site.CustomConfig = req.CustomConfig
	site.Redirects = req.Redirects
	site.Locations = req.Locations
	site.Hotlink = req.Hotlink
	site.RateLimit = req.RateLimit
	if req.BasicAuthUser != "" {
		site.BasicAuthUser = req.BasicAuthUser
		if req.BasicAuthPassword != "" {
			site.BasicAuthFile = s.writeBasicAuth(id, req.BasicAuthUser, req.BasicAuthPassword)
		}
	}
	site.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Regenerate Nginx config
	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	nginxConf := GenerateNginxConfig(site)
	if err := os.WriteFile(confPath, []byte(nginxConf), 0644); err != nil {
		return nil, fmt.Errorf("更新 Nginx 配置文件失败: %w", err)
	}

	if s.supervisor != nil {
		_ = s.supervisor.Reload(ctx)
	}

	_ = s.savePersisted()
	return site, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[id]
	if !ok {
		return fmt.Errorf("网站不存在: %s", id)
	}

	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	_ = os.Remove(confPath)

	if site.Type == "static" {
		siteDir := filepath.Join(s.dataDir, "sites", id)
		_ = os.RemoveAll(siteDir)
	}

	sslDir := filepath.Join(s.dataDir, "ssl", id)
	_ = os.RemoveAll(sslDir)

	delete(s.sites, id)
	_ = s.savePersisted()

	if s.supervisor != nil {
		_ = s.supervisor.Reload(ctx)
	}
	return nil
}

func (s *Service) Toggle(ctx context.Context, id string) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[id]
	if !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}

	if site.Status == "running" {
		site.Status = "stopped"
	} else {
		site.Status = "running"
	}
	site.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	nginxConf := GenerateNginxConfig(site)
	_ = os.WriteFile(confPath, []byte(nginxConf), 0644)

	if s.supervisor != nil {
		_ = s.supervisor.Reload(ctx)
	}

	_ = s.savePersisted()
	return site, nil
}

func (s *Service) ApplySSL(ctx context.Context, id string, req SSLApplyRequest) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.sites[id]
	if !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}

	allDomains := append([]string{site.PrimaryDomain}, site.Domains...)
	var expiry string
	var err error

	switch req.Provider {
	case "letsencrypt":
		expiry, err = s.sslManager.IssueLetsEncryptCert(ctx, id, allDomains, req.Email)
	case "self-signed":
		expiry, err = s.sslManager.IssueSelfSignedCert(id, allDomains)
	case "custom":
		expiry, err = s.sslManager.SaveCustomCert(id, req.CertPEM, req.KeyPEM)
	default:
		return nil, fmt.Errorf("不支持的 SSL 证书类型: %s", req.Provider)
	}

	if err != nil {
		return nil, fmt.Errorf("申请 SSL 证书失败: %w", err)
	}

	site.SSLEnabled = true
	site.SSLProvider = req.Provider
	site.CertExpiry = expiry
	site.ForceHTTPS = req.ForceHTTPS
	site.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Regenerate Nginx config with SSL directives
	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	nginxConf := GenerateNginxConfig(site)
	if err := os.WriteFile(confPath, []byte(nginxConf), 0644); err != nil {
		return nil, fmt.Errorf("更新 Nginx 配置文件失败: %w", err)
	}

	if s.supervisor != nil {
		_ = s.supervisor.Reload(ctx)
	}

	_ = s.savePersisted()
	return site, nil
}

func (s *Service) GetNginxConfig(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	site, ok := s.sites[id]
	if !ok {
		return "", fmt.Errorf("网站不存在: %s", id)
	}
	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	b, err := os.ReadFile(confPath)
	if err != nil {
		return GenerateNginxConfig(site), nil
	}
	return string(b), nil
}

func (s *Service) GetStaticFile(ctx context.Context, id, filename string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.sites[id]; !ok {
		return "", fmt.Errorf("网站不存在: %s", id)
	}
	filename = filepath.Clean("/" + filename)
	filePath := filepath.Join(s.dataDir, "sites", id, filename)
	b, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取静态文件失败: %w", err)
	}
	return string(b), nil
}

func (s *Service) SaveStaticFile(ctx context.Context, id, filename, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sites[id]; !ok {
		return fmt.Errorf("网站不存在: %s", id)
	}
	filename = filepath.Clean("/" + filename)
	filePath := filepath.Join(s.dataDir, "sites", id, filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0644)
}

func (s *Service) GetSSLInfo(ctx context.Context, id string) (*SSLInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.sites[id]; !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}
	return s.sslManager.GetSSLInfo(id)
}

// RenewHTTP01 submits an HTTP-01 order for a stored certificate through the ACME client.
func (s *Service) RenewHTTP01(ctx context.Context, id string, domains []string) (string, error) {
	if s.sslManager == nil {
		return "", fmt.Errorf("ACME 客户端不可用")
	}
	return s.sslManager.IssueLetsEncryptCert(ctx, id, domains, "")
}

// ensureRuntime creates the proxied runtime container when it is not already present.
// The image is one already in the local image catalog; creation does not pull.
func (s *Service) ensureRuntime(ctx context.Context, name string) error {
	if s.docker == nil {
		return fmt.Errorf("运行时容器需要容器引擎")
	}
	existing, err := s.docker.FindContainerByName(ctx, name)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	id, err := s.docker.Create(ctx, docker.CreateRequest{
		Name:  name,
		Image: "nginx:alpine",
		Ports: []docker.PublishedPort{{ContainerPort: 8080, Protocol: "tcp"}},
	})
	if err != nil {
		return fmt.Errorf("创建运行时容器失败: %w", err)
	}
	if err := s.docker.Action(ctx, id, "start"); err != nil {
		return fmt.Errorf("启动运行时容器失败: %w", err)
	}
	return nil
}

func prepareSiteRequest(kind, runtime, proxy, snippet string) error {
	if err := ValidateSnippet(snippet); err != nil {
		return err
	}
	switch kind {
	case "static", "proxy", "runtime":
	default:
		return fmt.Errorf("网站类型只支持 static、proxy 或 runtime")
	}
	if kind == "proxy" && strings.TrimSpace(proxy) == "" {
		return fmt.Errorf("反向代理网站必须指定目标代理地址")
	}
	if kind == "runtime" {
		switch runtime {
		case "php", "node", "java", "go", "python":
		default:
			return fmt.Errorf("运行环境只支持 php、node、java、go、python")
		}
	}
	return nil
}

func fillDays(site *Site) {
	if site == nil || site.CertExpiry == "" {
		return
	}
	expiry, err := time.Parse(time.RFC3339, site.CertExpiry)
	if err != nil {
		return
	}
	site.DaysRemaining = int(time.Until(expiry).Hours() / 24)
}

func (s *Service) writeBasicAuth(id, user, password string) string {
	dir := filepath.Join(s.dataDir, "nginx", "auth")
	_ = os.MkdirAll(dir, 0755)
	sum := sha1.Sum([]byte(password))
	line := user + ":{SHA}" + base64.StdEncoding.EncodeToString(sum[:]) + "\n"
	path := filepath.Join(dir, id+".htpasswd")
	_ = os.WriteFile(path, []byte(line), 0640)
	return path
}

func (s *Service) writeSiteLogs(id string) {
	dir := filepath.Join(s.dataDir, "nginx", "logs")
	_ = os.MkdirAll(dir, 0755)
	access := filepath.Join(dir, id+".access.log")
	errLog := filepath.Join(dir, id+".error.log")
	if _, err := os.Stat(access); err != nil {
		_ = os.WriteFile(access, []byte("camopanel access log\n"), 0644)
	}
	if _, err := os.Stat(errLog); err != nil {
		_ = os.WriteFile(errLog, []byte("camopanel error log\n"), 0644)
	}
}

func (s *Service) ReadLogs(id string) (string, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.sites[id]; !ok {
		return "", "", fmt.Errorf("网站不存在: %s", id)
	}
	dir := filepath.Join(s.dataDir, "nginx", "logs")
	access, _ := os.ReadFile(filepath.Join(dir, id+".access.log"))
	errLog, _ := os.ReadFile(filepath.Join(dir, id+".error.log"))
	return string(access), string(errLog), nil
}

func (s *Service) AttachCert(ctx context.Context, id, certID, expiry, provider string) (*Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	site, ok := s.sites[id]
	if !ok {
		return nil, fmt.Errorf("网站不存在: %s", id)
	}
	site.CertID = certID
	site.CertExpiry = expiry
	site.SSLEnabled = true
	if provider != "" {
		site.SSLProvider = provider
	}
	site.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	fillDays(site)
	confPath := filepath.Join(s.dataDir, "nginx", "conf.d", fmt.Sprintf("%s.conf", id))
	if err := os.WriteFile(confPath, []byte(GenerateNginxConfig(site)), 0644); err != nil {
		return nil, err
	}
	if s.supervisor != nil {
		_ = s.supervisor.Reload(ctx)
	}
	_ = s.savePersisted()
	return site, nil
}
