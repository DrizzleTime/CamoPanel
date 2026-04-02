package bootstrap

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"camopanel/server/internal/config"
)

func TestNewInitializesRouterModulesAndHealthRoute(t *testing.T) {
	tplDir := t.TempDir()
	writeTestTemplate(t, tplDir)

	root := t.TempDir()
	cfg := config.Config{
		HTTPAddr:           ":0",
		DataDir:            root,
		DatabasePath:       filepath.Join(root, "camopanel.db"),
		ProjectsDir:        filepath.Join(root, "projects"),
		TemplatesDir:       tplDir,
		SessionSecret:      "test-secret",
		CookieName:         "camopanel_session",
		AdminUsername:      "admin",
		AdminPassword:      "admin123",
		BridgeNetworkName:  "camopanel",
		HostControlHelper:  "/usr/local/bin/camopanel-hostctl",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("new application: %v", err)
	}

	if application.Router() == nil {
		t.Fatal("router should not be nil")
	}

	modules := application.Modules()
	if modules.Projects.Name == "" || modules.Copilot.Name == "" {
		t.Fatal("key modules should be registered")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	application.Router().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected health check 200, got %d", w.Code)
	}
}

func TestBootstrapRouterBridgesLegacyLogoutRoute(t *testing.T) {
	tplDir := t.TempDir()
	writeTestTemplate(t, tplDir)

	root := t.TempDir()
	cfg := config.Config{
		HTTPAddr:           ":0",
		DataDir:            root,
		DatabasePath:       filepath.Join(root, "camopanel.db"),
		ProjectsDir:        filepath.Join(root, "projects"),
		TemplatesDir:       tplDir,
		SessionSecret:      "test-secret",
		CookieName:         "camopanel_session",
		AdminUsername:      "admin",
		AdminPassword:      "admin123",
		BridgeNetworkName:  "camopanel",
		HostControlHelper:  "/usr/local/bin/camopanel-hostctl",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("new application: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	application.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("expected logout body to include ok=true, got %s", w.Body.String())
	}
}

func TestBootstrapRouterServesRuntimeDockerRoute(t *testing.T) {
	tplDir := t.TempDir()
	writeTestTemplate(t, tplDir)

	root := t.TempDir()
	cfg := config.Config{
		HTTPAddr:           ":0",
		DataDir:            root,
		DatabasePath:       filepath.Join(root, "camopanel.db"),
		ProjectsDir:        filepath.Join(root, "projects"),
		TemplatesDir:       tplDir,
		SessionSecret:      "test-secret",
		CookieName:         "camopanel_session",
		AdminUsername:      "admin",
		AdminPassword:      "admin123",
		BridgeNetworkName:  "camopanel",
		HostControlHelper:  "/usr/local/bin/camopanel-hostctl",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("new application: %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin123"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	application.Router().ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d body=%s", loginRec.Code, loginRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/docker/containers", nil)
	for _, cookie := range loginRec.Result().Cookies() {
		req.AddCookie(cookie)
	}

	rec := httptest.NewRecorder()
	application.Router().ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Fatalf("expected runtime route to be registered, got 404 body=%s", rec.Body.String())
	}
	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("expected auth cookie to reach runtime route, got 401 body=%s", rec.Body.String())
	}
}

func writeTestTemplate(t *testing.T, root string) {
	t.Helper()

	templateDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte("id: demo\nname: Demo\nversion: \"1\"\nparams: []\n"), 0o644); err != nil {
		t.Fatalf("write template spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "compose.yaml.tmpl"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write compose template: %v", err)
	}
}
