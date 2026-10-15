package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"camopanel/backend/internal/auth"
	"camopanel/backend/internal/config"
)

func TestGoalAuthAudit(t *testing.T) {
	router, dir := goalRouter(t, "test-token", "demo")

	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"wrong-pass"}`, "", ""), 401)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/bootstrap", `{"password":"correct-horse"}`, "", ""), 201)
	mustCode(t, callAs(router, "GET", "/api/v1/docker/overview", "", "", ""), 401)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"nope-nope"}`, "", ""), 401)

	admin := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"correct-horse"}`, "", ""))
	mustCode(t, callAs(router, "GET", "/api/v1/docker/overview", "", "", admin), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/password", `{"current":"correct-horse","next":"new-horse-1"}`, "", admin), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"correct-horse"}`, "", ""), 401)
	admin = sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"new-horse-1"}`, "", ""))

	mustCode(t, callAs(router, "POST", "/api/v1/auth/viewer", `{"password":"viewer-pass"}`, "", admin), 201)
	viewer := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"viewer","password":"viewer-pass"}`, "", ""))
	for _, path := range []string{"/api/v1/host/overview", "/api/v1/sites", "/api/v1/docker/overview", "/api/v1/docker/containers/a1b2c3d4e5f6/logs"} {
		mustCode(t, callAs(router, "GET", path, "", "", viewer), 200)
	}
	denied := []struct{ method, path, body string }{
		{"POST", "/api/v1/docker/containers", `{"name":"nope","image":"nginx:alpine"}`},
		{"DELETE", "/api/v1/sites/missing", ""},
		{"POST", "/api/v1/store/installed", `{"app_id":"redis","name":"nope"}`},
		{"POST", "/api/v1/certificates/self-signed", `{"domains":["a.example"]}`},
		{"PUT", "/api/v1/store/installed/missing/compose", `{"compose":"services: {}"}`},
		{"POST", "/api/v1/files/upload", `{"root":"sites","path":"a.txt","content":"x"}`},
		{"POST", "/api/v1/firewall/rules", `{"port":22,"policy":"allow"}`},
		{"POST", "/api/v1/terminals/host", `{}`},
		{"POST", "/api/v1/terminals/containers/a1b2c3d4e5f6", `{}`},
	}
	for _, item := range denied {
		rec := callAs(router, item.method, item.path, item.body, "", viewer)
		if rec.Code != 403 {
			t.Fatalf("%s %s status=%d body=%s", item.method, item.path, rec.Code, rec.Body.String())
		}
	}

	install := callAs(router, "POST", "/api/v1/store/installed", `{"app_id":"redis","name":"audit-cache","ports":{"REDIS_PORT":6399},"env":{"REDIS_PASSWORD":"super-secret-db-pass"}}`, "", admin)
	mustCode(t, install, 201)
	created := callAs(router, "POST", "/api/v1/sites", `{"name":"audit-site","primary_domain":"audit.local","type":"static"}`, "", admin)
	mustCode(t, created, 201)
	siteID := jsonField(t, created.Body.Bytes(), "id")
	mustCode(t, callAs(router, "DELETE", "/api/v1/sites/"+siteID, "", "", admin), 200)
	raw, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	logText := string(raw)
	if !strings.Contains(logText, `"/api/v1/store/installed"`) || !strings.Contains(logText, `"/api/v1/sites/`+siteID+`"`) {
		t.Fatalf("audit missing install or delete: %s", logText)
	}
	if !strings.Contains(logText, `"actor":"admin"`) || !strings.Contains(logText, `"method"`) || !strings.Contains(logText, `"status"`) {
		t.Fatalf("audit missing fields: %s", logText)
	}
	if strings.Contains(logText, "super-secret-db-pass") || strings.Contains(logText, "new-horse-1") || strings.Contains(logText, "BEGIN PRIVATE") {
		t.Fatalf("audit stored secret material: %s", logText)
	}

	began := callAs(router, "POST", "/api/v1/auth/totp/begin", `{}`, "", admin)
	mustCode(t, began, 200)
	secret := jsonField(t, began.Body.Bytes(), "secret")
	code, err := auth.CodeAt(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	mustCode(t, callAs(router, "POST", "/api/v1/auth/totp/enable", `{"code":"`+code+`"}`, "", admin), 200)
	alone := callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"new-horse-1"}`, "", "")
	if alone.Code != 401 || !strings.Contains(alone.Body.String(), "二次验证码") {
		t.Fatalf("totp login: %d %s", alone.Code, alone.Body.String())
	}
	if sessionCookieOK(alone) {
		t.Fatal("password alone created a session")
	}
	code, _ = auth.CodeAt(secret, time.Now())
	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"new-horse-1","code":"`+code+`"}`, "", ""), 200)
	mustCode(t, callAs(router, "GET", "/api/v1/host/overview", "", "test-token", ""), 200)

	t.Setenv("CAMO_ADDR", "0.0.0.0:8080")
	t.Setenv("CAMO_API_TOKEN", "")
	t.Setenv("CAMO_DATA_DIR", t.TempDir())
	t.Setenv("DOCKER_MODE", "demo")
	if _, err := config.Load(); err == nil {
		t.Fatal("public listener accepted without a token or password")
	}
	passDir := os.Getenv("CAMO_DATA_DIR")
	store, err := auth.Open(passDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Bootstrap("public-pass"); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(); err != nil {
		t.Fatal(err)
	}

	exposure := callAs(router, "GET", "/api/v1/settings/exposure", "", "", "")
	mustCode(t, exposure, 200)
	if !strings.Contains(exposure.Body.String(), "公网暴露面板需要 HTTPS") || strings.Contains(exposure.Body.String(), `"https":true`) {
		t.Fatalf("exposure: %s", exposure.Body.String())
	}

	certFile, keyFile := writeSelfSigned(t.TempDir())
	httpsRouter, _ := goalRouter(t, "", "demo")
	ln, err := listenLocal()
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	server := &http.Server{Addr: addr, Handler: httpsRouter}
	go func() { _ = Serve(server, certFile, keyFile) }()
	defer server.Shutdown(context.Background())
	var resp *http.Response
	var last error
	for i := 0; i < 20; i++ {
		resp, last = tlsClient().Get("https://" + addr + "/api/health")
		if last == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if last != nil {
		t.Fatal(last)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 || !resp.TLS.HandshakeComplete {
		t.Fatalf("https status=%d tls=%v", resp.StatusCode, resp.TLS != nil)
	}
}

func sessionCookieOK(rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.SessionCookie && c.Value != "" && c.MaxAge >= 0 {
			return true
		}
	}
	return false
}

func listenLocal() (netListener, error) {
	return netListen()
}
