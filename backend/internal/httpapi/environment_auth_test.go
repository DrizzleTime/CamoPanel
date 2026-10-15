package httpapi

import (
	"encoding/json"
	"strings"
	"testing"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/testutil"
)

func TestEnvironmentLoginProtectsPanel(t *testing.T) {
	router := New(config.Config{DataDir: t.TempDir(), AdminUsername: "operator", AdminPassword: "environment-password", Address: "127.0.0.1:8080"}, testutil.NewDocker())
	status := callAs(router, "GET", "/api/v1/auth/status", "", "", "")
	mustCode(t, status, 200)
	var info map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info["configured"] != true || info["source"] != "environment" {
		t.Fatal(status.Body.String())
	}
	if strings.Contains(status.Body.String(), "operator") || strings.Contains(status.Body.String(), "password") {
		t.Fatal("public status leaked credentials")
	}
	for _, path := range []string{"/api/v1/auth/session", "/api/v1/host/overview", "/api/v1/files/roots", "/api/v1/sites", "/api/v1/docker/overview"} {
		mustCode(t, callAs(router, "GET", path, "", "", ""), 401)
	}
	mustCode(t, callAs(router, "POST", "/api/v1/auth/bootstrap", `{"password":"another-password"}`, "", ""), 409)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"operator","password":"wrong-password"}`, "", ""), 401)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"environment-password"}`, "", ""), 401)
	login := callAs(router, "POST", "/api/v1/auth/login", `{"username":"operator","password":"environment-password"}`, "", "")
	mustCode(t, login, 200)
	cookie := sessionCookie(t, login)
	for _, item := range login.Result().Cookies() {
		if item.Name == "camo_session" && (!item.HttpOnly || item.MaxAge != 86400) {
			t.Fatal("session cookie attributes incorrect")
		}
	}
	mustCode(t, callAs(router, "GET", "/api/v1/auth/session", "", "", cookie), 200)
	mustCode(t, callAs(router, "GET", "/api/v1/files/roots", "", "", cookie), 200)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/password", `{"current":"environment-password","next":"another-password"}`, "", cookie), 409)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/viewer", `{"password":"another-password"}`, "", cookie), 409)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/logout", `{}`, "", cookie), 200)
	mustCode(t, callAs(router, "GET", "/api/v1/auth/session", "", "", cookie), 401)
}
