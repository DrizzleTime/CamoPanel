package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGoalAI(t *testing.T) {
	router, _ := goalRouter(t, "", "demo")
	settings := callAs(router, "GET", "/api/v1/ai/settings", "", "", "")
	mustCode(t, settings, 200)
	if !strings.Contains(settings.Body.String(), "https://api.openai.com/v1") || strings.Contains(settings.Body.String(), `"key_set":true`) {
		t.Fatalf("settings: %s", settings.Body.String())
	}
	empty := callAs(router, "POST", "/api/v1/ai/chat", `{"message":"现在什么情况"}`, "", "")
	mustCode(t, empty, 200)
	if !strings.Contains(empty.Body.String(), "请填写供应商") || !strings.Contains(empty.Body.String(), "主机") {
		t.Fatalf("empty provider: %s", empty.Body.String())
	}

	box := callAs(router, "POST", "/api/v1/docker/containers", `{"name":"uniq-web-777","image":"nginx:alpine"}`, "", "")
	mustCode(t, box, 201)
	cert := callAs(router, "POST", "/api/v1/certificates/issued", `{"provider":"custom","domains":["uniq-cert-777.example"],"expiry":"2030-01-01T00:00:00Z"}`, "", "")
	mustCode(t, cert, 201)

	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		var payload struct {
			Input json.RawMessage `json:"input"`
		}
		_ = json.Unmarshal(raw, &payload)
		inputText := string(payload.Input)
		if strings.Contains(string(raw), "function_call_output") {
			quoted, _ := json.Marshal(string(raw))
			_, _ = w.Write([]byte("data: {\"type\":\"response.output_text.delta\",\"delta\":" + string(quoted) + "}\n\n"))
			_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_2\"}}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		if !strings.Contains(string(raw), "camo-test") || r.Header.Get("X-Test") != "header-secret" || r.Header.Get("Authorization") != "Bearer super-secret-key" {
			http.Error(w, "missing provider fields", 400)
			return
		}
		if strings.Contains(inputText, "请重启这个容器") {
			_, _ = w.Write([]byte("data: {\"type\":\"response.function_call\",\"name\":\"restart_container\",\"call_id\":\"call_r\",\"arguments\":\"{\\\"id\\\":\\\"b2c3d4e5f6a1\\\"}\"}\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			return
		}
		_, _ = w.Write([]byte("data: {\"type\":\"response.function_call\",\"name\":\"list_containers\",\"call_id\":\"call_c\",\"arguments\":\"{}\"}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"function_call\",\"name\":\"list_certificates\",\"call_id\":\"call_k\",\"arguments\":\"{}\"}}\n\n"))
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\"}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer fake.Close()

	saved := callAs(router, "PUT", "/api/v1/ai/settings", `{"base_url":"`+fake.URL+`/v1","model":"camo-test","api_key":"super-secret-key","headers":{"X-Test":"header-secret"}}`, "", "")
	mustCode(t, saved, 200)
	if strings.Contains(saved.Body.String(), "super-secret-key") || strings.Contains(saved.Body.String(), "header-secret") || !strings.Contains(saved.Body.String(), `"key_set":true`) || !strings.Contains(saved.Body.String(), `"secret_set":true`) {
		t.Fatalf("settings leaked: %s", saved.Body.String())
	}
	answer := callAs(router, "POST", "/api/v1/ai/chat", `{"message":"当前容器和证书","stream":true}`, "", "")
	mustCode(t, answer, 200)
	if !strings.Contains(answer.Body.String(), "uniq-web-777") || !strings.Contains(answer.Body.String(), "uniq-cert-777.example") {
		t.Fatalf("answer missed live data: %s", answer.Body.String())
	}
	before := callAs(router, "GET", "/api/v1/docker/containers/b2c3d4e5f6a1/logs", "", "", "")
	declined := callAs(router, "POST", "/api/v1/ai/chat", `{"message":"请重启这个容器","confirm":false}`, "", "")
	mustCode(t, declined, 200)
	if !strings.Contains(declined.Body.String(), "pending_write") || !strings.Contains(declined.Body.String(), "restart_container") {
		t.Fatalf("pending: %s", declined.Body.String())
	}
	after := callAs(router, "GET", "/api/v1/docker/containers/b2c3d4e5f6a1/logs", "", "", "")
	if before.Body.String() != after.Body.String() {
		t.Fatal("declined confirmation restarted the container")
	}

	mustCode(t, callAs(router, "POST", "/api/v1/auth/bootstrap", `{"password":"correct-horse"}`, "", ""), 201)
	admin := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"correct-horse"}`, "", ""))
	mustCode(t, callAs(router, "POST", "/api/v1/auth/viewer", `{"password":"viewer-pass"}`, "", admin), 201)
	viewer := sessionCookie(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"viewer","password":"viewer-pass"}`, "", ""))
	refused := callAs(router, "POST", "/api/v1/ai/chat", `{"message":"请重启这个容器","confirm":true}`, "", viewer)
	if refused.Code != 403 {
		t.Fatalf("viewer write: %d %s", refused.Code, refused.Body.String())
	}
	afterViewer := callAs(router, "GET", "/api/v1/docker/containers/b2c3d4e5f6a1/logs", "", "", admin)
	if afterViewer.Body.String() != before.Body.String() {
		t.Fatal("viewer write tool restarted the container")
	}
}
