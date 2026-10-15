package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGoalHostSecurity(t *testing.T) {
	router, _ := goalRouter(t, "", "demo")
	overview := callAs(router, "GET", "/api/v1/host/overview", "", "", "")
	mustCode(t, overview, 200)
	var hostBody struct {
		Mode    string           `json:"mode"`
		CPU     map[string]any   `json:"cpu"`
		Memory  map[string]any   `json:"memory"`
		Disks   []map[string]any `json:"disks"`
		History []map[string]any `json:"history"`
	}
	if err := json.Unmarshal(overview.Body.Bytes(), &hostBody); err != nil {
		t.Fatal(err)
	}
	if hostBody.Mode != "demo" || len(hostBody.CPU) == 0 || len(hostBody.Memory) == 0 || len(hostBody.Disks) == 0 || len(hostBody.History) == 0 {
		t.Fatalf("overview incomplete: %s", overview.Body.String())
	}
	procs := callAs(router, "GET", "/api/v1/host/processes?sort=mem", "", "", "")
	mustCode(t, procs, 200)
	var procBody struct {
		Processes []map[string]any `json:"processes"`
	}
	if err := json.Unmarshal(procs.Body.Bytes(), &procBody); err != nil || len(procBody.Processes) == 0 || len(procBody.Processes) > 10 {
		t.Fatalf("processes: %s", procs.Body.String())
	}
	mustCode(t, callAs(router, "GET", "/api/v1/host/processes?limit=50&sort=cpu", "", "", ""), 200)
	mustCode(t, callAs(router, "GET", "/api/v1/host/processes?limit=51", "", "", ""), 400)

	allow := callAs(router, "POST", "/api/v1/firewall/rules", `{"port":22,"policy":"allow"}`, "", "")
	mustCode(t, allow, 201)
	denied := callAs(router, "POST", "/api/v1/firewall/rules", `{"port":8080,"policy":"deny"}`, "", "")
	if denied.Code != 400 || !strings.Contains(denied.Body.String(), "关掉面板端口") {
		t.Fatalf("panel lockout: %d %s", denied.Code, denied.Body.String())
	}
	mustCode(t, callAs(router, "POST", "/api/v1/firewall/rules", `{"port":8080,"policy":"deny","confirm":true}`, "", ""), 201)

	empty := t.TempDir()
	t.Setenv("PATH", empty)
	systemRouter, _ := goalRouter(t, "", "system")
	status := callAs(systemRouter, "GET", "/api/v1/firewall", "", "", "")
	mustCode(t, status, 200)
	if !strings.Contains(status.Body.String(), "当前系统没有可管理的防火墙") {
		t.Fatalf("firewall status: %s", status.Body.String())
	}
	missing := callAs(systemRouter, "POST", "/api/v1/firewall/rules", `{"port":22,"policy":"allow"}`, "", "")
	if missing.Code != 400 || missing.Code == 500 {
		t.Fatalf("missing firewall: %d %s", missing.Code, missing.Body.String())
	}

	var hits int
	hook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(204)
	}))
	defer hook.Close()
	mustCode(t, callAs(router, "POST", "/api/v1/alerts", `{"kind":"disk","threshold":10,"webhook":"`+hook.URL+`"}`, "", ""), 201)
	evaluated := callAs(router, "POST", "/api/v1/alerts/evaluate", `{"disk_percent":40}`, "", "")
	mustCode(t, evaluated, 200)
	if hits != 1 || !strings.Contains(evaluated.Body.String(), `"sent":1`) {
		t.Fatalf("webhook hits=%d body=%s", hits, evaluated.Body.String())
	}

	mustCode(t, callAs(router, "POST", "/api/v1/auth/bootstrap", `{"password":"correct-horse"}`, "", ""), 201)
	admin := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"correct-horse"}`, "", ""))
	mustCode(t, callAs(router, "POST", "/api/v1/auth/viewer", `{"password":"viewer-pass"}`, "", admin), 201)
	viewer := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"viewer","password":"viewer-pass"}`, "", ""))
	for _, path := range []string{"/api/v1/firewall/rules", "/api/v1/terminals/host", "/api/v1/terminals/containers/a1b2c3d4e5f6"} {
		rec := callAs(router, "POST", path, `{"port":22,"policy":"allow"}`, "", viewer)
		if rec.Code != 403 {
			t.Fatalf("%s status=%d", path, rec.Code)
		}
	}
	before, _ := os.ReadFile(routerData(router))
	_ = before
	hostTerm := callAs(router, "POST", "/api/v1/terminals/host", `{"command":"rm -rf /"}`, "", admin)
	mustCode(t, hostTerm, 200)
	boxTerm := callAs(router, "POST", "/api/v1/terminals/containers/a1b2c3d4e5f6", `{"command":"ls"}`, "", admin)
	mustCode(t, boxTerm, 200)
}

func routerData(router any) string { return "" }
