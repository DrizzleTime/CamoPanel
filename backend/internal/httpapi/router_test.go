package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/docker"
	"github.com/gin-gonic/gin"
)

func call(handler http.Handler, method, path, body, token, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}
func TestAuthAndOriginProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(config.Config{Token: "test-token"}, docker.NewDemo())
	for _, test := range []struct {
		name, path, token, origin string
		status                    int
	}{
		{"health", "/api/health", "", "", 200}, {"unauthorized", "/api/v1/docker/overview", "", "", 401}, {"wrong token", "/api/v1/docker/overview", "incorrect", "", 401}, {"authorized", "/api/v1/docker/overview", "test-token", "", 200}, {"cross origin", "/api/v1/docker/overview", "test-token", "https://evil.example", 403}, {"same origin", "/api/v1/docker/overview", "test-token", "http://example.com", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := call(router, "GET", test.path, "", test.token, test.origin)
			if response.Code != test.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestContainerLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(config.Config{}, docker.NewDemo())
	created := call(router, "POST", "/api/v1/docker/containers", `{"name":"test-web","image":"nginx:alpine","host_port":8090,"container_port":80}`, "", "")
	if created.Code != 201 {
		t.Fatalf("create failed: %s", created.Body.String())
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	base := "/api/v1/docker/containers/" + body.ID
	for _, test := range []struct {
		action string
		status int
	}{{"start", 200}, {"restart", 200}, {"remove", 409}, {"stop", 200}, {"remove", 200}, {"start", 404}} {
		result := call(router, "POST", base+"/"+test.action, "", "", "")
		if result.Code != test.status {
			t.Fatalf("action=%s status=%d body=%s", test.action, result.Code, result.Body.String())
		}
	}
}

func TestHostOverviewAndProcesses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(config.Config{Token: "test-token"}, docker.NewDemo())
	denied := call(router, "GET", "/api/v1/host/overview", "", "", "")
	if denied.Code != 401 {
		t.Fatalf("status=%d", denied.Code)
	}
	overview := call(router, "GET", "/api/v1/host/overview", "", "test-token", "")
	if overview.Code != 200 || !strings.Contains(overview.Body.String(), `"mode":"demo"`) {
		t.Fatalf("overview=%d %s", overview.Code, overview.Body.String())
	}
	processes := call(router, "GET", "/api/v1/host/processes", "", "test-token", "")
	if processes.Code != 200 || !strings.Contains(processes.Body.String(), `"name":"postgres"`) {
		t.Fatalf("processes=%s", processes.Body.String())
	}
	for _, path := range []string{"/api/v1/host/processes?limit=51", "/api/v1/host/processes?limit=0", "/api/v1/host/processes?limit=abc", "/api/v1/host/processes?sort=disk"} {
		result := call(router, "GET", path, "", "test-token", "")
		if result.Code != 400 {
			t.Fatalf("path=%s status=%d body=%s", path, result.Code, result.Body.String())
		}
	}
}

func TestInvalidCreateAndAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := New(config.Config{}, docker.NewDemo())
	for _, body := range []string{`{}`, `{"name":"bad/name","image":"nginx:alpine"}`, `{"name":"valid","image":"nginx:alpine","host_port":8088}`, `{"name":"valid","image":"nginx:alpine","host_port":70000,"container_port":80}`, `{"name":"valid","image":" "}`} {
		result := call(router, "POST", "/api/v1/docker/containers", body, "", "")
		if result.Code != 400 {
			t.Fatalf("invalid body accepted: %s", body)
		}
	}
	result := call(router, "POST", "/api/v1/docker/containers/a1b2c3d4e5f6/exec", "", "", "")
	if result.Code != 400 {
		t.Fatalf("unsupported action accepted: %d", result.Code)
	}
	logs := call(router, "GET", "/api/v1/docker/containers/a1b2c3d4e5f6/logs", "", "", "")
	if logs.Code != 200 || !strings.Contains(logs.Body.String(), "[demo]") {
		t.Fatalf("invalid logs: %s", logs.Body.String())
	}
}
