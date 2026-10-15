package nginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"camopanel/backend/internal/docker"
)

type Status struct {
	Installed       bool   `json:"installed"`
	Running         bool   `json:"running"`
	ContainerID     string `json:"container_id"`
	HTTPPort        int    `json:"http_port"`
	HTTPSPort       int    `json:"https_port"`
	ConfigPath      string `json:"config_path"`
	LastReloadAt    string `json:"last_reload_at"`
	ActiveSiteCount int    `json:"active_site_count"`
}

type Supervisor struct {
	mu            sync.RWMutex
	dataDir       string
	dockerMode    string
	dockerService docker.Service
	httpPort      int
	httpsPort     int
	lastReloadAt  string
	containerID   string
}

func NewSupervisor(dataDir, dockerMode string, dockerService docker.Service, httpPort, httpsPort int) (*Supervisor, error) {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	s := &Supervisor{
		dataDir:       absDataDir,
		dockerMode:    dockerMode,
		dockerService: dockerService,
		httpPort:      httpPort,
		httpsPort:     httpsPort,
		lastReloadAt:  time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.initDirectoriesAndConfig(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Supervisor) initDirectoriesAndConfig() error {
	dirs := []string{
		filepath.Join(s.dataDir, "nginx"),
		filepath.Join(s.dataDir, "nginx", "conf.d"),
		filepath.Join(s.dataDir, "sites"),
		filepath.Join(s.dataDir, "ssl"),
		filepath.Join(s.dataDir, "acme", ".well-known", "acme-challenge"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建 Nginx 目录失败 %s: %w", dir, err)
		}
	}

	mainConfPath := filepath.Join(s.dataDir, "nginx", "nginx.conf")
	if _, err := os.Stat(mainConfPath); os.IsNotExist(err) {
		mainConf := `user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;

events {
    worker_connections  1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    tcp_nopush      on;
    tcp_nodelay     on;
    keepalive_timeout  65;
    gzip  on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;

    include /etc/nginx/conf.d/*.conf;
}
`
		if err := os.WriteFile(mainConfPath, []byte(mainConf), 0644); err != nil {
			return err
		}
	}

	// Default fallback virtual host
	defaultConfPath := filepath.Join(s.dataDir, "nginx", "conf.d", "00_default.conf")
	if _, err := os.Stat(defaultConfPath); os.IsNotExist(err) {
		defaultConf := `server {
    listen 80 default_server;
    server_name _;

    location /.well-known/acme-challenge/ {
        root /var/www/acme;
        try_files $uri =404;
    }

    location / {
        default_type text/html;
        return 200 '<!DOCTYPE html><html><head><meta charset="utf-8"><title>CamoPanel Gateway</title><style>body{font-family:system-ui,sans-serif;display:grid;place-items:center;min-height:90vh;background:#f8fafc;color:#1e293b}div{text-align:center;padding:32px;background:#fff;border-radius:12px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.1)}h1{color:#e11d48;margin:0 0 8px}p{color:#64748b;margin:0}</style></head><body><div><h1>CamoPanel Nginx 网站网关</h1><p>内置 Web 服务已就绪。当前尚未为该域名配置站点，请在控制台添加站点并绑定域名。</p></div></body></html>';
    }
}
`
		_ = os.WriteFile(defaultConfPath, []byte(defaultConf), 0644)
	}

	return nil
}

func (s *Supervisor) EnsureRunning(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	containerName := "camo-nginx"
	if s.dockerService != nil {
		c, err := s.dockerService.FindContainerByName(ctx, containerName)
		if err == nil && c != nil {
			s.containerID = c.ID
			if c.State != "running" {
				return s.dockerService.Action(ctx, c.ID, "start")
			}
			return nil
		}

		// Launch container
		binds := []string{
			fmt.Sprintf("%s:/etc/nginx/nginx.conf:ro", filepath.Join(s.dataDir, "nginx", "nginx.conf")),
			fmt.Sprintf("%s:/etc/nginx/conf.d", filepath.Join(s.dataDir, "nginx", "conf.d")),
			fmt.Sprintf("%s:/var/www/sites", filepath.Join(s.dataDir, "sites")),
			fmt.Sprintf("%s:/etc/nginx/ssl:ro", filepath.Join(s.dataDir, "ssl")),
			fmt.Sprintf("%s:/var/www/acme:ro", filepath.Join(s.dataDir, "acme")),
		}
		ports := []docker.Port{
			{IP: "0.0.0.0", PrivatePort: 80, PublicPort: s.httpPort, Type: "tcp"},
			{IP: "0.0.0.0", PrivatePort: 443, PublicPort: s.httpsPort, Type: "tcp"},
		}

		id, err := s.dockerService.RunContainer(ctx, docker.RunContainerRequest{
			Name:          containerName,
			Image:         "nginx:alpine",
			Ports:         ports,
			Binds:         binds,
			RestartPolicy: "always",
			Labels: map[string]string{
				"io.camopanel.managed": "true",
				"io.camopanel.role":    "gateway-nginx",
			},
		})
		if err != nil {
			return fmt.Errorf("启动内置 Nginx 容器失败: %w", err)
		}
		s.containerID = id
	}
	return nil
}

func (s *Supervisor) Reload(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastReloadAt = time.Now().UTC().Format(time.RFC3339)
	if s.dockerService != nil && s.containerID != "" {
		_ = s.dockerService.Action(ctx, s.containerID, "restart")
	}
	return nil
}

func (s *Supervisor) Action(ctx context.Context, action string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dockerService == nil || s.containerID == "" {
		return nil
	}
	switch action {
	case "start", "stop", "restart":
		return s.dockerService.Action(ctx, s.containerID, action)
	case "reload":
		s.lastReloadAt = time.Now().UTC().Format(time.RFC3339)
		return s.dockerService.Action(ctx, s.containerID, "restart")
	default:
		return fmt.Errorf("不支持的 Nginx 操作: %s", action)
	}
}

func (s *Supervisor) GetStatus(ctx context.Context) Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	running := false
	if s.dockerService != nil {
		c, err := s.dockerService.FindContainerByName(ctx, "camo-nginx")
		if err == nil && c != nil {
			s.containerID = c.ID
			running = (c.State == "running")
		}
	}

	confEntries, _ := os.ReadDir(filepath.Join(s.dataDir, "nginx", "conf.d"))
	activeSites := 0
	for _, entry := range confEntries {
		if filepath.Ext(entry.Name()) == ".conf" && entry.Name() != "00_default.conf" {
			activeSites++
		}
	}

	return Status{
		Installed:       true,
		Running:         running,
		ContainerID:     s.containerID,
		HTTPPort:        s.httpPort,
		HTTPSPort:       s.httpsPort,
		ConfigPath:      filepath.Join(s.dataDir, "nginx", "conf.d"),
		LastReloadAt:    s.lastReloadAt,
		ActiveSiteCount: activeSites,
	}
}
