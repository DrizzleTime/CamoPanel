package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/docker"
	"github.com/gin-gonic/gin"
)

func TestStoreAndSiteEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()

	cfg := config.Config{
		DataDir: tmpDir,
		Mode:    "demo",
	}
	router := New(cfg, docker.NewDemo())

	// 1. App Store Catalog
	resp := call(router, "GET", "/api/v1/store/apps", "", "", "")
	if resp.Code != 200 {
		t.Fatalf("list apps failed: %d %s", resp.Code, resp.Body.String())
	}
	var catalog []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &catalog); err != nil || len(catalog) == 0 {
		t.Fatalf("catalog unmarshal failed or empty: %v", err)
	}

	// 2. Install App with Path Constraint Validation
	// Path with escape attempt should be rejected
	badInstallBody := `{"app_id":"wordpress","name":"bad-app","custom_compose":"services:\n  app:\n    image: wordpress\n    volumes:\n      - /etc/shadow:/etc/shadow"}`
	badResp := call(router, "POST", "/api/v1/store/installed", badInstallBody, "", "")
	if badResp.Code != 400 || !strings.Contains(badResp.Body.String(), "安全约束拦截") {
		t.Fatalf("expected security constraint rejection: %d %s", badResp.Code, badResp.Body.String())
	}

	// Valid install
	installBody := `{"app_id":"redis","name":"my-cache","ports":{"REDIS_PORT":6388}}`
	installResp := call(router, "POST", "/api/v1/store/installed", installBody, "", "")
	if installResp.Code != 201 {
		t.Fatalf("install failed: %d %s", installResp.Code, installResp.Body.String())
	}

	// Verify installed list
	instListResp := call(router, "GET", "/api/v1/store/installed", "", "", "")
	if instListResp.Code != 200 || !strings.Contains(instListResp.Body.String(), "my-cache") {
		t.Fatalf("installed list missing my-cache: %s", instListResp.Body.String())
	}

	// Verify compose content endpoint
	composeResp := call(router, "GET", "/api/v1/store/installed/my-cache/compose", "", "", "")
	if composeResp.Code != 200 || !strings.Contains(composeResp.Body.String(), "redis:7-alpine") {
		t.Fatalf("compose endpoint failed: %s", composeResp.Body.String())
	}

	// App action: stop
	actionResp := call(router, "POST", "/api/v1/store/installed/my-cache/action", `{"action":"stop"}`, "", "")
	if actionResp.Code != 200 {
		t.Fatalf("stop action failed: %s", actionResp.Body.String())
	}

	// 3. Website Endpoints
	// Create Static Site
	createSiteBody := `{"name":"demo-site","primary_domain":"demo.local","domains":["www.demo.local"],"type":"static"}`
	createSiteResp := call(router, "POST", "/api/v1/sites", createSiteBody, "", "")
	if createSiteResp.Code != 201 {
		t.Fatalf("create site failed: %d %s", createSiteResp.Code, createSiteResp.Body.String())
	}
	var createdSite struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(createSiteResp.Body.Bytes(), &createdSite)
	if createdSite.ID == "" {
		t.Fatalf("site ID is empty")
	}

	// Check static index.html file endpoint
	fileResp := call(router, "GET", "/api/v1/sites/"+createdSite.ID+"/files/index.html", "", "", "")
	if fileResp.Code != 200 || !strings.Contains(fileResp.Body.String(), "demo.local") {
		t.Fatalf("get static file failed: %s", fileResp.Body.String())
	}

	// Update static file
	saveFileResp := call(router, "PUT", "/api/v1/sites/"+createdSite.ID+"/files/index.html", `{"content":"<h1>Updated</h1>"}`, "", "")
	if saveFileResp.Code != 200 {
		t.Fatalf("save static file failed: %s", saveFileResp.Body.String())
	}

	// Apply self-signed SSL
	sslBody := `{"provider":"self-signed","force_https":true}`
	sslResp := call(router, "POST", "/api/v1/sites/"+createdSite.ID+"/ssl/apply", sslBody, "", "")
	if sslResp.Code != 200 || !strings.Contains(sslResp.Body.String(), `"ssl_enabled":true`) {
		t.Fatalf("apply ssl failed: %s", sslResp.Body.String())
	}

	// Preview Nginx config
	confResp := call(router, "GET", "/api/v1/sites/"+createdSite.ID+"/config", "", "", "")
	if confResp.Code != 200 || !strings.Contains(confResp.Body.String(), "listen 443 ssl") {
		t.Fatalf("preview config failed: %s", confResp.Body.String())
	}

	// Create Reverse Proxy Site
	proxySiteBody := `{"name":"my-api","primary_domain":"api.demo.local","type":"proxy","proxy_target":"http://127.0.0.1:8080"}`
	proxySiteResp := call(router, "POST", "/api/v1/sites", proxySiteBody, "", "")
	if proxySiteResp.Code != 201 {
		t.Fatalf("create proxy site failed: %d %s", proxySiteResp.Code, proxySiteResp.Body.String())
	}

	// 4. Built-in Nginx Status Endpoint
	nginxResp := call(router, "GET", "/api/v1/nginx/status", "", "", "")
	if nginxResp.Code != 200 || !strings.Contains(nginxResp.Body.String(), `"installed":true`) {
		t.Fatalf("nginx status failed: %s", nginxResp.Body.String())
	}

	// Clean up app uninstall
	uninstResp := call(router, "POST", "/api/v1/store/installed/my-cache/action", `{"action":"uninstall","delete_data":true}`, "", "")
	if uninstResp.Code != 200 {
		t.Fatalf("uninstall app failed: %s", uninstResp.Body.String())
	}
}
