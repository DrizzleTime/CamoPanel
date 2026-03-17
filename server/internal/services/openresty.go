package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var ErrOpenRestyUnavailable = errors.New("openresty is unavailable")

const (
	openRestyContainerConfDir       = "/etc/nginx/conf.d"
	openRestyLegacyContainerConfDir = "/etc/openresty/conf.d"
	openRestyContainerSiteDir       = "/var/www/openresty"
	openRestyContainerCertDir       = "/etc/camopanel/certs"
	certbotChallengeDir             = "/var/www/certbot"
	certbotConfigDir                = "/etc/letsencrypt"
)

type OpenRestyStatus struct {
	CertificateReady bool   `json:"certificate_ready"`
	Exists           bool   `json:"exists"`
	Ready            bool   `json:"ready"`
	ContainerName    string `json:"container_name"`
	ContainerStatus  string `json:"container_status"`
	HostConfigDir    string `json:"host_config_dir"`
	HostSiteDir      string `json:"host_site_dir"`
	Message          string `json:"message"`
}

type WebsiteSpec struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Domain        string   `json:"domain"`
	Domains       []string `json:"domains,omitempty"`
	RootPath      string   `json:"root_path,omitempty"`
	IndexFiles    []string `json:"index_files,omitempty"`
	ProxyPass     string   `json:"proxy_pass,omitempty"`
	PHPPort       int      `json:"php_port,omitempty"`
	RewriteMode   string   `json:"rewrite_mode,omitempty"`
	RewritePreset string   `json:"rewrite_preset,omitempty"`
	RewriteRules  string   `json:"rewrite_rules,omitempty"`
}

type WebsiteMaterialized struct {
	RootPath   string
	ConfigPath string
}

type CertificateSpec struct {
	Domain             string `json:"domain"`
	Email              string `json:"email"`
	UseExistingWebsite bool   `json:"use_existing_website"`
}

type CertificateMaterialized struct {
	Provider       string
	FullchainPath  string
	PrivateKeyPath string
	ExpiresAt      time.Time
}

type OpenRestyManager interface {
	Status(context.Context) OpenRestyStatus
	EnsureReady(context.Context) error
	CreateWebsite(context.Context, WebsiteSpec) (WebsiteMaterialized, error)
	UpdateWebsite(context.Context, WebsiteSpec, string) (WebsiteMaterialized, error)
	DeleteWebsite(context.Context, string) error
	PreviewWebsiteConfig(WebsiteSpec) (string, error)
	SyncWebsite(context.Context, WebsiteSpec) (WebsiteMaterialized, error)
	IssueCertificate(context.Context, CertificateSpec) (CertificateMaterialized, error)
	DeleteCertificate(context.Context, string) error
}

type OpenRestyService struct {
	docker         ContainerOperator
	dockerBinPath  string
	container      string
	dataDir        string
	hostConfDir    string
	hostSiteDir    string
	hostCertDir    string
	hostChallenge  string
	containerChall string
}

type websiteTLS struct {
	FullchainPath  string
	PrivateKeyPath string
}

func NewOpenRestyService(docker ContainerOperator, containerName, dataDir string) *OpenRestyService {
	dockerBinPath, _ := exec.LookPath("docker")
	hostSiteDir := filepath.Join(dataDir, "www")
	return &OpenRestyService{
		docker:         docker,
		dockerBinPath:  dockerBinPath,
		container:      strings.TrimSpace(containerName),
		dataDir:        dataDir,
		hostConfDir:    filepath.Join(dataDir, "conf.d"),
		hostSiteDir:    hostSiteDir,
		hostCertDir:    filepath.Join(dataDir, "certs"),
		hostChallenge:  filepath.Join(hostSiteDir, "_acme-challenge"),
		containerChall: path.Join(openRestyContainerSiteDir, "_acme-challenge"),
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
	status.CertificateReady = s.hasCertificateMount(containerStatus)
	if status.CertificateReady {
		status.Message = "OpenResty 容器可用"
		return status
	}
	status.Message = "OpenResty 容器可用，但缺少证书目录挂载"
	return status
}

func (s *OpenRestyService) EnsureReady(ctx context.Context) error {
	if err := os.MkdirAll(s.hostConfDir, 0o755); err != nil {
		return fmt.Errorf("创建 OpenResty 配置目录失败: %w", err)
	}
	if err := os.MkdirAll(s.hostSiteDir, 0o755); err != nil {
		return fmt.Errorf("创建 OpenResty 站点目录失败: %w", err)
	}
	if err := os.MkdirAll(s.hostChallenge, 0o755); err != nil {
		return fmt.Errorf("创建 ACME challenge 目录失败: %w", err)
	}
	if err := os.MkdirAll(s.hostCertDir, 0o755); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
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
	return s.writeWebsiteConfig(ctx, spec, filepath.Join(s.hostConfDir, spec.Name+".conf"), true)
}

func (s *OpenRestyService) UpdateWebsite(ctx context.Context, spec WebsiteSpec, configPath string) (WebsiteMaterialized, error) {
	return s.writeWebsiteConfig(ctx, spec, configPath, false)
}

func (s *OpenRestyService) DeleteWebsite(ctx context.Context, configPath string) error {
	if err := s.EnsureReady(ctx); err != nil {
		return err
	}
	if err := os.Remove(configPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除站点配置失败: %w", err)
	}
	return s.reloadOpenResty(ctx)
}

func (s *OpenRestyService) PreviewWebsiteConfig(spec WebsiteSpec) (string, error) {
	containerRootPath, err := s.containerRootPath(spec)
	if err != nil {
		return "", err
	}
	tlsConfig, err := s.resolveWebsiteTLS(context.Background(), spec)
	if err != nil {
		return "", err
	}
	return renderWebsiteConfig(spec, containerRootPath, s.containerChall, tlsConfig)
}

func (s *OpenRestyService) SyncWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error) {
	return s.writeWebsiteConfig(ctx, spec, filepath.Join(s.hostConfDir, spec.Name+".conf"), false)
}

func (s *OpenRestyService) IssueCertificate(ctx context.Context, spec CertificateSpec) (materialized CertificateMaterialized, err error) {
	if err := s.EnsureReady(ctx); err != nil {
		return CertificateMaterialized{}, err
	}
	if err := s.ensureCertificateMountReady(ctx); err != nil {
		return CertificateMaterialized{}, err
	}
	if strings.TrimSpace(spec.Domain) == "" {
		return CertificateMaterialized{}, fmt.Errorf("域名不能为空")
	}
	if strings.TrimSpace(spec.Email) == "" {
		return CertificateMaterialized{}, fmt.Errorf("邮箱不能为空")
	}
	if s.dockerBinPath == "" {
		return CertificateMaterialized{}, fmt.Errorf("未找到 docker 命令，无法申请证书")
	}

	tempConfigPath := ""
	if !spec.UseExistingWebsite {
		tempConfigPath, err = s.ensureValidationConfig(ctx, spec.Domain)
		if err != nil {
			return CertificateMaterialized{}, err
		}
		defer func() {
			if tempConfigPath == "" {
				return
			}
			_ = os.Remove(tempConfigPath)
			_ = s.reloadOpenResty(ctx)
		}()
	}

	cmd := exec.CommandContext(
		ctx,
		s.dockerBinPath,
		"run", "--rm",
		"-v", s.hostChallenge+":"+certbotChallengeDir,
		"-v", s.hostCertDir+":"+certbotConfigDir,
		"certbot/certbot",
		"certonly",
		"--webroot",
		"-w", certbotChallengeDir,
		"-d", strings.TrimSpace(spec.Domain),
		"--email", strings.TrimSpace(spec.Email),
		"--agree-tos",
		"--no-eff-email",
		"--non-interactive",
		"--keep-until-expiring",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return CertificateMaterialized{}, fmt.Errorf("申请证书失败: %v", err)
		}
		return CertificateMaterialized{}, fmt.Errorf("申请证书失败: %s", trimmed)
	}

	fullchainPath, privateKeyPath := s.certificateHostPaths(spec.Domain)
	expiresAt, err := certificateExpiry(fullchainPath)
	if err != nil {
		return CertificateMaterialized{}, err
	}

	return CertificateMaterialized{
		Provider:       "letsencrypt",
		FullchainPath:  fullchainPath,
		PrivateKeyPath: privateKeyPath,
		ExpiresAt:      expiresAt,
	}, nil
}

func (s *OpenRestyService) DeleteCertificate(_ context.Context, domain string) error {
	trimmedDomain := strings.TrimSpace(domain)
	if trimmedDomain == "" {
		return fmt.Errorf("域名不能为空")
	}

	liveDir := filepath.Join(s.hostCertDir, "live", trimmedDomain)
	archiveDir := filepath.Join(s.hostCertDir, "archive", trimmedDomain)
	renewalPath := filepath.Join(s.hostCertDir, "renewal", trimmedDomain+".conf")

	if err := os.RemoveAll(liveDir); err != nil {
		return fmt.Errorf("删除证书目录失败: %w", err)
	}
	if err := os.RemoveAll(archiveDir); err != nil {
		return fmt.Errorf("删除证书归档失败: %w", err)
	}
	if err := os.Remove(renewalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除证书续期配置失败: %w", err)
	}
	return nil
}

func (s *OpenRestyService) writeWebsiteConfig(ctx context.Context, spec WebsiteSpec, configPath string, createMode bool) (WebsiteMaterialized, error) {
	if err := s.EnsureReady(ctx); err != nil {
		return WebsiteMaterialized{}, err
	}

	previousConfig, hadPrevious, err := readFileIfExists(configPath)
	if err != nil {
		return WebsiteMaterialized{}, err
	}
	if createMode && hadPrevious {
		return WebsiteMaterialized{}, fmt.Errorf("站点配置已存在")
	}

	rootPath := ""
	if spec.Type == "static" || spec.Type == "php" {
		rootPath = spec.RootPath
		if err := os.MkdirAll(rootPath, 0o755); err != nil {
			return WebsiteMaterialized{}, fmt.Errorf("创建站点目录失败: %w", err)
		}
		if createMode {
			if spec.Type == "php" {
				indexPath := filepath.Join(rootPath, "index.php")
				if _, err := os.Stat(indexPath); errors.Is(err, os.ErrNotExist) {
					if err := os.WriteFile(indexPath, []byte(defaultIndexPHP(spec.Domain)), 0o644); err != nil {
						return WebsiteMaterialized{}, fmt.Errorf("写入默认首页失败: %w", err)
					}
				}
			} else {
				indexPath := filepath.Join(rootPath, "index.html")
				if _, err := os.Stat(indexPath); errors.Is(err, os.ErrNotExist) {
					if err := os.WriteFile(indexPath, []byte(defaultIndexHTML(spec.Domain)), 0o644); err != nil {
						return WebsiteMaterialized{}, fmt.Errorf("写入默认首页失败: %w", err)
					}
				}
			}
		}
	}
	containerRootPath, err := s.containerRootPath(spec)
	if err != nil {
		return WebsiteMaterialized{}, err
	}

	tlsConfig, err := s.resolveWebsiteTLS(ctx, spec)
	if err != nil {
		return WebsiteMaterialized{}, err
	}

	configBody, err := renderWebsiteConfig(spec, containerRootPath, s.containerChall, tlsConfig)
	if err != nil {
		return WebsiteMaterialized{}, err
	}
	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		return WebsiteMaterialized{}, fmt.Errorf("写入站点配置失败: %w", err)
	}

	if err := s.reloadOpenResty(ctx); err != nil {
		if hadPrevious {
			_ = os.WriteFile(configPath, previousConfig, 0o644)
		} else {
			_ = os.Remove(configPath)
		}
		return WebsiteMaterialized{}, err
	}

	return WebsiteMaterialized{
		RootPath:   rootPath,
		ConfigPath: configPath,
	}, nil
}

func (s *OpenRestyService) ensureValidationConfig(ctx context.Context, domain string) (string, error) {
	configPath := filepath.Join(s.hostConfDir, "_acme_"+sanitizeDomain(domain)+".conf")
	configBody := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 404;
    }
}
`, domain, s.containerChall)

	if err := os.WriteFile(configPath, []byte(configBody), 0o644); err != nil {
		return "", fmt.Errorf("写入 ACME 校验配置失败: %w", err)
	}
	if err := s.reloadOpenResty(ctx); err != nil {
		_ = os.Remove(configPath)
		return "", err
	}
	return configPath, nil
}

func (s *OpenRestyService) resolveWebsiteTLS(ctx context.Context, spec WebsiteSpec) (*websiteTLS, error) {
	if len(spec.Domains) > 0 {
		return nil, nil
	}

	fullchainPath, privateKeyPath := s.certificateHostPaths(spec.Domain)
	if _, err := os.Stat(fullchainPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取证书失败: %w", err)
	}
	if _, err := os.Stat(privateKeyPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取私钥失败: %w", err)
	}
	if err := s.ensureCertificateMountReady(ctx); err != nil {
		return nil, err
	}

	fullchainContainerPath, privateKeyContainerPath := s.certificateContainerPaths(spec.Domain)
	return &websiteTLS{
		FullchainPath:  fullchainContainerPath,
		PrivateKeyPath: privateKeyContainerPath,
	}, nil
}

func (s *OpenRestyService) containerRootPath(spec WebsiteSpec) (string, error) {
	if spec.Type != "static" && spec.Type != "php" {
		return "", nil
	}
	relativePath, err := filepath.Rel(s.hostSiteDir, spec.RootPath)
	if err != nil || strings.HasPrefix(relativePath, "..") {
		return "", fmt.Errorf("站点目录必须位于 %s 下", s.hostSiteDir)
	}
	return path.Join(openRestyContainerSiteDir, filepath.ToSlash(relativePath)), nil
}

func (s *OpenRestyService) reloadOpenResty(ctx context.Context) error {
	if _, err := s.docker.ExecInContainer(ctx, s.container, "openresty", "-t"); err != nil {
		return err
	}
	if _, err := s.docker.ExecInContainer(ctx, s.container, "openresty", "-s", "reload"); err != nil {
		return err
	}
	return nil
}

func (s *OpenRestyService) hasRequiredMounts(status ContainerStatus) bool {
	hasConf := false
	hasSite := false

	expectedConf := filepath.Clean(s.hostConfDir)
	expectedSite := filepath.Clean(s.hostSiteDir)

	for _, mount := range status.Mounts {
		source := filepath.Clean(mount.Source)
		switch path.Clean(mount.Destination) {
		case openRestyContainerConfDir, openRestyLegacyContainerConfDir:
			hasConf = source == expectedConf
		case openRestyContainerSiteDir:
			hasSite = source == expectedSite
		}
	}

	return hasConf && hasSite
}

func (s *OpenRestyService) hasCertificateMount(status ContainerStatus) bool {
	expectedCertDir := filepath.Clean(s.hostCertDir)
	for _, mount := range status.Mounts {
		if path.Clean(mount.Destination) == openRestyContainerCertDir {
			return filepath.Clean(mount.Source) == expectedCertDir
		}
	}
	return false
}

func (s *OpenRestyService) ensureCertificateMountReady(ctx context.Context) error {
	containerStatus, err := s.docker.InspectContainer(ctx, s.container)
	if err != nil {
		return fmt.Errorf("%w: Docker 当前不可用", ErrOpenRestyUnavailable)
	}
	if !containerStatus.Exists || !containerStatus.Running {
		return fmt.Errorf("%w: OpenResty 容器未运行", ErrOpenRestyUnavailable)
	}
	if !s.hasCertificateMount(containerStatus) {
		return fmt.Errorf("OpenResty 缺少证书目录挂载，请重新部署 OpenResty 模板")
	}
	return nil
}

func (s *OpenRestyService) certificateHostPaths(domain string) (string, string) {
	liveDir := filepath.Join(s.hostCertDir, "live", strings.TrimSpace(domain))
	return filepath.Join(liveDir, "fullchain.pem"), filepath.Join(liveDir, "privkey.pem")
}

func (s *OpenRestyService) certificateContainerPaths(domain string) (string, string) {
	liveDir := path.Join(openRestyContainerCertDir, "live", strings.TrimSpace(domain))
	return path.Join(liveDir, "fullchain.pem"), path.Join(liveDir, "privkey.pem")
}

func renderWebsiteConfig(spec WebsiteSpec, rootPath, challengePath string, tlsConfig *websiteTLS) (string, error) {
	serverNames := strings.Join(append([]string{spec.Domain}, spec.Domains...), " ")
	indexFiles := strings.Join(defaultIndexFiles(spec.IndexFiles), " ")

	switch spec.Type {
	case "static":
		staticLocation := staticLocationBlock(spec.RewriteMode, spec.RewritePreset, spec.RewriteRules)
		if tlsConfig == nil {
			return fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
%s
    }
}
`, serverNames, rootPath, indexFiles, challengePath, indentBlock(staticLocation, 8)), nil
		}
		return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;
    root %s;
    index %s;
    ssl_certificate %s;
    ssl_certificate_key %s;

    location / {
%s
    }
}
`, serverNames, challengePath, serverNames, rootPath, indexFiles, tlsConfig.FullchainPath, tlsConfig.PrivateKeyPath, indentBlock(staticLocation, 8)), nil
	case "php":
		if spec.PHPPort <= 0 {
			return "", fmt.Errorf("PHP 环境端口无效")
		}
		phpLocation := phpLocationBlock(spec.RewriteMode, spec.RewritePreset, spec.RewriteRules, spec.PHPPort)
		if tlsConfig == nil {
			return fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

%s
}
`, serverNames, rootPath, indexFiles, challengePath, phpLocation), nil
		}
		return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;
    root %s;
    index %s;
    ssl_certificate %s;
    ssl_certificate_key %s;

%s
}
`, serverNames, challengePath, serverNames, rootPath, indexFiles, tlsConfig.FullchainPath, tlsConfig.PrivateKeyPath, phpLocation), nil
	case "proxy":
		target, err := url.Parse(spec.ProxyPass)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return "", fmt.Errorf("代理地址格式不正确")
		}
		if tlsConfig == nil {
			return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass %s;
    }
}
`, serverNames, challengePath, spec.ProxyPass), nil
		}
		return fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location ^~ /.well-known/acme-challenge/ {
        alias %s/;
        default_type text/plain;
        try_files $uri =404;
    }

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name %s;
    ssl_certificate %s;
    ssl_certificate_key %s;

    location / {
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_pass %s;
    }
}
`, serverNames, challengePath, serverNames, tlsConfig.FullchainPath, tlsConfig.PrivateKeyPath, spec.ProxyPass), nil
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

func defaultIndexPHP(domain string) string {
	return fmt.Sprintf(`<?php
header('Content-Type: text/html; charset=utf-8');
?><!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>%s</title>
  </head>
  <body>
    <h1>%s</h1>
    <p>PHP 站点已创建，当前请求由 OpenResty 转发到 PHP-FPM。</p>
  </body>
</html>
`, domain, domain)
}

func readFileIfExists(path string) ([]byte, bool, error) {
	content, err := os.ReadFile(path)
	if err == nil {
		return content, true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("读取配置文件失败: %w", err)
}

func sanitizeDomain(domain string) string {
	return strings.NewReplacer(".", "_", "*", "_", "/", "_").Replace(strings.TrimSpace(domain))
}

func defaultIndexFiles(items []string) []string {
	if len(items) == 0 {
		return []string{"index.html", "index.htm"}
	}
	return items
}

func staticLocationBlock(mode string, preset string, rules string) string {
	switch strings.TrimSpace(mode) {
	case "preset":
		if strings.TrimSpace(preset) == "front_controller" {
			return "try_files $uri $uri/ /index.php?$query_string;"
		}
		return "try_files $uri $uri/ /index.html;"
	case "custom":
		trimmed := strings.TrimSpace(rules)
		if trimmed != "" {
			return trimmed
		}
	}
	return "try_files $uri $uri/ =404;"
}

func phpLocationBlock(mode string, preset string, rules string, port int) string {
	locationBlock := fmt.Sprintf(`location / {
%s
}

location ~ \.php$ {
    include fastcgi_params;
    fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    fastcgi_param DOCUMENT_ROOT $document_root;
    fastcgi_index index.php;
    fastcgi_pass 127.0.0.1:%d;
}`, indentBlock(phpRewriteBlock(mode, preset, rules), 4), port)
	return indentBlock(locationBlock, 4)
}

func phpRewriteBlock(mode string, preset string, rules string) string {
	switch strings.TrimSpace(mode) {
	case "preset":
		if strings.TrimSpace(preset) == "spa" {
			return "try_files $uri $uri/ /index.html;"
		}
		return "try_files $uri $uri/ /index.php?$query_string;"
	case "custom":
		trimmed := strings.TrimSpace(rules)
		if trimmed != "" {
			return trimmed
		}
	}
	return "try_files $uri $uri/ =404;"
}

func indentBlock(content string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimSpace(content), "\n")
	for i, line := range lines {
		lines[i] = indent + strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func certificateExpiry(fullchainPath string) (time.Time, error) {
	content, err := os.ReadFile(fullchainPath)
	if err != nil {
		return time.Time{}, fmt.Errorf("读取证书失败: %w", err)
	}

	block, _ := pem.Decode(content)
	if block == nil {
		return time.Time{}, fmt.Errorf("解析证书失败")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, fmt.Errorf("解析证书失败: %w", err)
	}
	return cert.NotAfter.UTC(), nil
}
