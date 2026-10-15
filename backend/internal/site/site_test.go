package site

import (
	"context"
	"strings"
	"testing"

	"camopanel/backend/internal/docker"
	"camopanel/backend/internal/nginx"
)

func TestNginxConfigGeneration(t *testing.T) {
	// Test Static Site Config
	staticSite := &Site{
		ID:            "site-static-1",
		Name:          "static-test",
		PrimaryDomain: "static.example.com",
		Domains:       []string{"www.static.example.com"},
		Type:          "static",
		RootPath:      "/var/www/sites/site-static-1",
		SSLEnabled:    false,
		Status:        "running",
	}
	conf := GenerateNginxConfig(staticSite)
	if !strings.Contains(conf, "server_name static.example.com www.static.example.com;") {
		t.Fatalf("expected server_name with primary and alias domains, got:\n%s", conf)
	}
	if !strings.Contains(conf, "root /var/www/sites/site-static-1;") {
		t.Fatalf("expected static root path, got:\n%s", conf)
	}
	if !strings.Contains(conf, "try_files $uri $uri/ /index.html =404;") {
		t.Fatalf("expected try_files, got:\n%s", conf)
	}

	// Test Reverse Proxy Site Config with SSL
	proxySite := &Site{
		ID:            "site-proxy-1",
		Name:          "proxy-test",
		PrimaryDomain: "api.example.com",
		Type:          "proxy",
		ProxyTarget:   "http://127.0.0.1:3000",
		SSLEnabled:    true,
		ForceHTTPS:    true,
		Status:        "running",
	}
	sslConf := GenerateNginxConfig(proxySite)
	if !strings.Contains(sslConf, "return 301 https://$host$request_uri;") {
		t.Fatalf("expected 301 redirect for force HTTPS, got:\n%s", sslConf)
	}
	if !strings.Contains(sslConf, "listen 443 ssl;") {
		t.Fatalf("expected listen 443 ssl, got:\n%s", sslConf)
	}
	if !strings.Contains(sslConf, "proxy_pass http://127.0.0.1:3000;") {
		t.Fatalf("expected proxy_pass, got:\n%s", sslConf)
	}
	if !strings.Contains(sslConf, "Upgrade $http_upgrade;") {
		t.Fatalf("expected websocket upgrade header, got:\n%s", sslConf)
	}

	// Test Stopped Site Config
	stoppedSite := &Site{
		ID:            "site-stopped-1",
		Name:          "stopped-test",
		PrimaryDomain: "down.example.com",
		Status:        "stopped",
	}
	stoppedConf := GenerateNginxConfig(stoppedSite)
	if !strings.Contains(stoppedConf, "return 503") {
		t.Fatalf("expected 503 for stopped site, got:\n%s", stoppedConf)
	}
}

func TestSiteLifecycleAndSelfSignedSSL(t *testing.T) {
	tmpDir := t.TempDir()
	demo := docker.NewDemo()
	sup, err := nginx.NewSupervisor(tmpDir, "demo", demo, 80, 443)
	if err != nil {
		t.Fatalf("new supervisor failed: %v", err)
	}

	sslMgr := NewSSLManager(tmpDir, "admin@example.com")
	svc, err := NewService(tmpDir, sup, sslMgr, demo)
	if err != nil {
		t.Fatalf("new site service failed: %v", err)
	}

	ctx := context.Background()

	// 1. Create static site
	site, err := svc.Create(ctx, CreateSiteRequest{
		Name:          "blog",
		PrimaryDomain: "blog.local",
		Domains:       []string{"www.blog.local"},
		Type:          "static",
	})
	if err != nil {
		t.Fatalf("create site failed: %v", err)
	}
	if site.PrimaryDomain != "blog.local" || site.Type != "static" {
		t.Fatalf("unexpected site: %+v", site)
	}

	// Verify default index.html was created
	html, err := svc.GetStaticFile(ctx, site.ID, "index.html")
	if err != nil || !strings.Contains(html, "blog.local") {
		t.Fatalf("index.html missing or incorrect: %v, %s", err, html)
	}

	// 2. Apply self-signed SSL
	updatedSite, err := svc.ApplySSL(ctx, site.ID, SSLApplyRequest{
		Provider:   "self-signed",
		ForceHTTPS: true,
	})
	if err != nil {
		t.Fatalf("apply self-signed ssl failed: %v", err)
	}
	if !updatedSite.SSLEnabled || updatedSite.SSLProvider != "self-signed" {
		t.Fatalf("ssl not enabled on site: %+v", updatedSite)
	}

	// Verify SSL info
	info, err := svc.GetSSLInfo(ctx, site.ID)
	if err != nil || !info.Enabled {
		t.Fatalf("get ssl info failed: %v, %+v", err, info)
	}

	// 3. Toggle site status
	toggled, err := svc.Toggle(ctx, site.ID)
	if err != nil || toggled.Status != "stopped" {
		t.Fatalf("toggle to stopped failed: %v, %+v", err, toggled)
	}

	// 4. Update site
	up, err := svc.Update(ctx, site.ID, UpdateSiteRequest{
		Name:          "blog-updated",
		PrimaryDomain: "blog.local",
		Domains:       []string{"m.blog.local"},
		Type:          "static",
	})
	if err != nil || up.Name != "blog-updated" {
		t.Fatalf("update failed: %v", err)
	}

	// 5. Delete site
	if err := svc.Delete(ctx, site.ID); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := svc.Get(ctx, site.ID); err == nil {
		t.Fatalf("site should be deleted")
	}
}
