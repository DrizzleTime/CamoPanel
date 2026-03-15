package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var ErrOpenRestyUnavailable = errors.New("openresty is unavailable")

const (
	openRestyContainerConfDir = "/etc/openresty/conf.d"
	openRestyContainerSiteDir = "/var/www/openresty"
)

type OpenRestyStatus struct {
	Exists          bool   `json:"exists"`
	Ready           bool   `json:"ready"`
	ContainerName   string `json:"container_name"`
	ContainerStatus string `json:"container_status"`
	HostConfigDir   string `json:"host_config_dir"`
	HostSiteDir     string `json:"host_site_dir"`
	Message         string `json:"message"`
}

type WebsiteSpec struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Domain    string `json:"domain"`
	ProxyPass string `json:"proxy_pass,omitempty"`
}

type WebsiteMaterialized struct {
	RootPath   string
	ConfigPath string
}

type OpenRestyManager interface {
	Status(context.Context) OpenRestyStatus
	EnsureReady(context.Context) error
	CreateWebsite(context.Context, WebsiteSpec) (WebsiteMaterialized, error)
}

type OpenRestyService struct {
	docker      ContainerOperator
	container   string
	dataDir     string
	hostConfDir string
	hostSiteDir string
}

func NewOpenRestyService(docker ContainerOperator, containerName, dataDir string) *OpenRestyService {
	return &OpenRestyService{
		docker:      docker,
		container:   strings.TrimSpace(containerName),
		dataDir:     dataDir,
		hostConfDir: filepath.Join(dataDir, "conf.d"),
		hostSiteDir: filepath.Join(dataDir, "www"),
	}
}

func (s *OpenRestyService) Status(ctx context.Context) OpenRestyStatus {
	status := OpenRestyStatus{
		ContainerName: strings.TrimSpace(s.container),
		HostConfigDir: s.hostConfDir,
		HostSiteDir:   s.hostSiteDir,
	}

	containerStatus, err := s.docker.InspectContainer(ctx, s.container)
	if err != nil {
		status.Message = "Docker 当前不可用"
		return status
	}
	if !containerStatus.Exists {
		status.Message = "未找到 OpenResty 容器"
		return status
	}

	status.Exists = true
	status.ContainerStatus = containerStatus.Status

	if !containerStatus.Running {
		status.Message = "OpenResty 容器未运行"
		return status
	}
	if !s.hasRequiredMounts(containerStatus) {
		status.Message = "OpenResty 容器挂载目录不符合约定"
		return status
	}

	status.Ready = true
	status.Message = "OpenResty 容器可用"
	return status
}

func (s *OpenRestyService) EnsureReady(ctx context.Context) error {
	if err := os.MkdirAll(s.hostConfDir, 0o755); err != nil {
		return fmt.Errorf("创建 OpenResty 配置目录失败: %w", err)
	}
	if err := os.MkdirAll(s.hostSiteDir, 0o755); err != nil {
		return fmt.Errorf("创建 OpenResty 站点目录失败: %w", err)
	}

	containerStatus, err := s.docker.InspectContainer(ctx, s.container)
	if err != nil {
		return fmt.Errorf("%w: Docker 当前不可用", ErrOpenRestyUnavailable)
	}
	if !containerStatus.Exists {
		return fmt.Errorf("%w: 未找到 OpenResty 容器 %s", ErrOpenRestyUnavailable, s.container)
	}
	if !containerStatus.Running {
		return fmt.Errorf("%w: OpenResty 容器未运行", ErrOpenRestyUnavailable)
	}
	if !s.hasRequiredMounts(containerStatus) {
		return fmt.Errorf("%w: OpenResty 容器必须挂载 %s 和 %s", ErrOpenRestyUnavailable, s.hostConfDir, s.hostSiteDir)
	}
	return nil
}

func (s *OpenRestyService) CreateWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error) {
	if err := s.EnsureReady(ctx); err != nil {
		return WebsiteMaterialized{}, err
	}

	configPath := filepath.Join(s.hostConfDir, spec.Name+".conf")
	if _, err := os.Stat(configPath); err == nil {
		return WebsiteMaterialized{}, fmt.Errorf("站点配置已存在")
	}

	rootPath := ""
	containerRootPath := ""
	if spec.Type == "static" {
		rootPath = filepath.Join(s.hostSiteDir, spec.Name)
		containerRootPath = path.Join(openRestyContainerSiteDir, spec.Name)
		if err := os.MkdirAll(rootPath, 0o755); err != nil {
			return WebsiteMaterialized{}, fmt.Errorf("创建站点目录失败: %w", err)
		}
		if err := os.WriteFile(filepath.Join(rootPath, "index.html"), []byte(defaultIndexHTML(spec.Domain)), 0o644); err != nil {
			return WebsiteMaterialized{}, fmt.Errorf("写入默认首页失败: %w", err)
		}
	}

	configBody, err := renderWebsiteConfig(spec, containerRootPath)
	if err != nil {
		return WebsiteMaterialized{}, err
	}
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		return WebsiteMaterialized{}, fmt.Errorf("写入站点配置失败: %w", err)
	}

	if _, err := s.docker.ExecInContainer(ctx, s.container, "openresty", "-t"); err != nil {
		_ = os.Remove(configPath)
		return WebsiteMaterialized{}, err
	}
	if _, err := s.docker.ExecInContainer(ctx, s.container, "openresty", "-s", "reload"); err != nil {
		_ = os.Remove(configPath)
		return WebsiteMaterialized{}, err
	}

	return WebsiteMaterialized{
		RootPath:   rootPath,
		ConfigPath: configPath,
	}, nil
}

func (s *OpenRestyService) hasRequiredMounts(status ContainerStatus) bool {
	hasConf := false
	hasSite := false

	expectedConf := filepath.Clean(s.hostConfDir)
	expectedSite := filepath.Clean(s.hostSiteDir)

	for _, mount := range status.Mounts {
		source := filepath.Clean(mount.Source)
		switch path.Clean(mount.Destination) {
		case openRestyContainerConfDir:
			hasConf = source == expectedConf
		case openRestyContainerSiteDir:
			hasSite = source == expectedSite
		}
	}

	return hasConf && hasSite
}

func renderWebsiteConfig(spec WebsiteSpec, rootPath string) (string, error) {
	switch spec.Type {
	case "static":
		return fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.html index.htm;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, spec.Domain, rootPath), nil
	case "proxy":
		target, err := url.Parse(spec.ProxyPass)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return "", fmt.Errorf("代理地址格式不正确")
		}
		return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass %s;
    }
}
`, spec.Domain, spec.ProxyPass), nil
	default:
		return "", fmt.Errorf("不支持的网站类型: %s", spec.Type)
	}
}

func defaultIndexHTML(domain string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>%s</title>
  </head>
  <body>
    <h1>%s</h1>
    <p>站点已创建，当前页面由 OpenResty 容器提供服务。</p>
  </body>
</html>
`, domain, domain)
}
