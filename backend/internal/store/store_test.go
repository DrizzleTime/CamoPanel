package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"camopanel/backend/internal/docker"
)

func TestPathConstraintsAndVolumeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "apps", "my-app")

	if err := EnsureAppDirectoryConstraints(appDir); err != nil {
		t.Fatalf("ensure app dir failed: %v", err)
	}

	// Subdirectories data, config, logs must exist
	for _, sub := range []string{"data", "config", "logs"} {
		info, err := os.Stat(filepath.Join(appDir, sub))
		if err != nil || !info.IsDir() {
			t.Fatalf("expected subdirectory %s to exist in appDir", sub)
		}
	}

	// Valid relative volume mounts
	validMounts := []string{
		"./data",
		"./config",
		"./logs",
		"./data/mysql",
		"my-named-volume",
	}
	for _, m := range validMounts {
		if err := ValidateVolumeConstraint(appDir, m); err != nil {
			t.Fatalf("valid mount rejected: %s (%v)", m, err)
		}
	}

	// Invalid escaping volume mounts
	invalidMounts := []string{
		"../../etc/shadow",
		"../other-app",
		"/etc",
		"/var/run/docker.sock",
		"/root",
		"/",
	}
	for _, m := range invalidMounts {
		if err := ValidateVolumeConstraint(appDir, m); err == nil {
			t.Fatalf("security violation accepted: %s", m)
		}
	}

	// Test compose yaml volume validation
	safeCompose := `
services:
  web:
    image: nginx:alpine
    volumes:
      - ./data:/usr/share/nginx/html
      - ./config:/etc/nginx/conf.d
`
	if err := ValidateComposeVolumes(appDir, safeCompose); err != nil {
		t.Fatalf("safe compose rejected: %v", err)
	}

	unsafeCompose := `
services:
  web:
    image: nginx:alpine
    volumes:
      - /etc/shadow:/etc/shadow
`
	if err := ValidateComposeVolumes(appDir, unsafeCompose); err == nil {
		t.Fatalf("unsafe compose with /etc/shadow accepted")
	}
}

func TestStoreAppLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	demo := docker.NewDemo()
	svc, err := NewService(tmpDir, "demo", demo)
	if err != nil {
		t.Fatalf("new store service failed: %v", err)
	}

	ctx := context.Background()

	// List catalog
	catalog, err := svc.ListCatalog(ctx)
	if err != nil || len(catalog) == 0 {
		t.Fatalf("list catalog failed: %v", err)
	}

	// Install WordPress
	installed, err := svc.Install(ctx, InstallRequest{
		AppID: "wordpress",
		Name:  "my-blog",
		Ports: map[string]int{"WEB_PORT": 8099},
		Env:   map[string]string{"MYSQL_ROOT_PASSWORD": "secret_root_pwd"},
	})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if installed.Name != "my-blog" || installed.Status != "running" {
		t.Fatalf("invalid installed app: %+v", installed)
	}

	// Check directory constraint files created
	composePath := filepath.Join(installed.AppDir, "docker-compose.yml")
	composeBytes, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("compose file missing: %v", err)
	}
	if !strings.Contains(string(composeBytes), "8099:80") {
		t.Fatalf("port was not substituted: %s", string(composeBytes))
	}

	// Check catalog now reflects installed
	catApp, err := svc.GetCatalogApp("wordpress")
	if err != nil || !catApp.Installed {
		t.Fatalf("catalog should mark wordpress as installed: %+v", catApp)
	}

	// Action stop, restart, uninstall
	if err := svc.Action(ctx, "my-blog", "stop", false); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if app, _ := svc.GetInstalled("my-blog"); app.Status != "stopped" {
		t.Fatalf("status should be stopped: %s", app.Status)
	}

	if err := svc.Action(ctx, "my-blog", "restart", false); err != nil {
		t.Fatalf("restart failed: %v", err)
	}
	if app, _ := svc.GetInstalled("my-blog"); app.Status != "running" {
		t.Fatalf("status should be running: %s", app.Status)
	}

	// Uninstall with deleteData
	if err := svc.Action(ctx, "my-blog", "uninstall", true); err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}
	if _, err := svc.GetInstalled("my-blog"); err == nil {
		t.Fatalf("app should be deleted")
	}
}

func TestAppStoreSpecificationAndLoader(t *testing.T) {
	// 1. Test LoadEmbeddedCatalog
	apps, err := LoadEmbeddedCatalog()
	if err != nil {
		t.Fatalf("LoadEmbeddedCatalog failed: %v", err)
	}
	if len(apps) < 8 {
		t.Fatalf("expected at least 8 apps, got %d", len(apps))
	}

	appMap := make(map[string]StoreApp)
	for _, app := range apps {
		appMap[app.ID] = app
	}

	// 2. Verify all standard apps exist
	expectedApps := []string{"nginx", "wordpress", "mysql", "redis", "postgres", "nextcloud", "uptime-kuma", "vaultwarden", "gitea"}
	for _, id := range expectedApps {
		app, exists := appMap[id]
		if !exists {
			t.Fatalf("expected app %s to be embedded", id)
		}
		if app.Name == "" || app.Category == "" {
			t.Fatalf("app %s has invalid metadata", id)
		}
		if app.ComposeTemplate == "" {
			t.Fatalf("app %s missing compose template", id)
		}

		// Verify that all compose volumes obey strict directory constraints
		tmpAppDir := filepath.Join(t.TempDir(), "apps", id)
		if err := ValidateComposeVolumes(tmpAppDir, app.ComposeTemplate); err != nil {
			t.Fatalf("app %s compose volumes violation: %v", id, err)
		}
	}

	// 3. Test WordPress rich port and params
	wp := appMap["wordpress"]
	if !wp.Gateway.IsWebApp || wp.Gateway.PrimaryPortKey != "WEB_PORT" {
		t.Fatalf("wordpress gateway spec incorrect: %+v", wp.Gateway)
	}
	foundPrimaryPort := false
	for _, p := range wp.Ports {
		if p.Key == "WEB_PORT" {
			foundPrimaryPort = true
			if !p.Proxyable || p.ContainerPort != 80 || p.Default != 8080 {
				t.Fatalf("wordpress WEB_PORT spec incorrect: %+v", p)
			}
		}
	}
	if !foundPrimaryPort {
		t.Fatalf("wordpress WEB_PORT not found in ports")
	}

	foundSecret := false
	for _, p := range wp.Params {
		if p.Key == "MYSQL_ROOT_PASSWORD" {
			foundSecret = true
			if !p.IsSecret || !p.AutoGenerate {
				t.Fatalf("MYSQL_ROOT_PASSWORD expected secret with auto_generate: %+v", p)
			}
		}
	}
	if !foundSecret {
		t.Fatalf("MYSQL_ROOT_PASSWORD not found in params")
	}

	// 4. Test Password Generator
	pwd1 := GenerateRandomPassword(16)
	pwd2 := GenerateRandomPassword(16)
	if len(pwd1) != 16 || len(pwd2) != 16 || pwd1 == pwd2 {
		t.Fatalf("password generator failed: %s vs %s", pwd1, pwd2)
	}

	// 5. Test Auto password generation during Install
	tmpDir := t.TempDir()
	demo := docker.NewDemo()
	svc, err := NewService(tmpDir, "demo", demo)
	if err != nil {
		t.Fatalf("new service failed: %v", err)
	}

	inst, err := svc.Install(context.Background(), InstallRequest{
		AppID: "wordpress",
		Name:  "wp-auto-pwd",
	})
	if err != nil {
		t.Fatalf("install with auto pwd failed: %v", err)
	}
	if inst.PrimaryPort != 8080 {
		t.Fatalf("expected PrimaryPort to be 8080, got %d", inst.PrimaryPort)
	}
	rootPwd := inst.Env["MYSQL_ROOT_PASSWORD"]
	if rootPwd == "" || rootPwd == "auto" {
		t.Fatalf("expected generated password, got: %s", rootPwd)
	}

	// 6. Test External App Catalog loading
	customDir := filepath.Join(tmpDir, "custom_apps")
	customAppDir := filepath.Join(customDir, "custom-hello")
	if err := os.MkdirAll(customAppDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	customYaml := `
schema_version: "1.0"
id: "custom-hello"
name: "Custom Hello App"
category: "tools"
version: "1.0"
icon: "globe"
description: "A custom test app"
ports:
  - key: "PORT"
    label: "Port"
    container_port: 80
    default: 9000
    proxyable: true
`
	customCompose := `
services:
  app:
    image: hello-world
    ports:
      - "${PORT}:80"
`
	if err := os.WriteFile(filepath.Join(customAppDir, "app.yaml"), []byte(customYaml), 0644); err != nil {
		t.Fatalf("write custom app.yaml failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(customAppDir, "compose.yaml"), []byte(customCompose), 0644); err != nil {
		t.Fatalf("write custom compose.yaml failed: %v", err)
	}

	merged, err := LoadAllCatalog(customDir)
	if err != nil {
		t.Fatalf("LoadAllCatalog failed: %v", err)
	}
	foundCustom := false
	for _, a := range merged {
		if a.ID == "custom-hello" {
			foundCustom = true
			if a.DefaultPorts["PORT"] != 9000 {
				t.Fatalf("custom app default port incorrect: %+v", a)
			}
		}
	}
	if !foundCustom {
		t.Fatalf("custom app was not loaded in merged catalog")
	}
}

