package config

import "testing"

func TestPublicListenerRequiresToken(t *testing.T) {
	t.Setenv("CAMO_ADDR", "0.0.0.0:8080")
	t.Setenv("CAMO_API_TOKEN", "")
	t.Setenv("DOCKER_MODE", "demo")
	if _, err := Load(); err == nil {
		t.Fatal("public listener accepted without a token")
	}
	t.Setenv("CAMO_API_TOKEN", "token")
	if _, err := Load(); err != nil {
		t.Fatal(err)
	}
}
func TestRejectInvalidMode(t *testing.T) {
	t.Setenv("DOCKER_MODE", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

func TestEnvironmentLoginConfiguration(t *testing.T) {
	t.Setenv("CAMO_ADDR", "127.0.0.1:8080")
	t.Setenv("CAMO_DATA_DIR", t.TempDir())
	t.Setenv("CAMO_API_TOKEN", "")
	t.Setenv("DOCKER_MODE", "demo")
	t.Setenv("CAMO_ADMIN_USERNAME", "operator")
	t.Setenv("CAMO_ADMIN_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("unconfigured local panel accepted")
	}
	t.Setenv("CAMO_ADMIN_PASSWORD", "short")
	if _, err := Load(); err == nil {
		t.Fatal("short environment password accepted")
	}
	t.Setenv("CAMO_ADMIN_PASSWORD", "environment-password")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminUsername != "operator" || cfg.AdminPassword != "environment-password" {
		t.Fatal("environment credentials not loaded")
	}
	t.Setenv("CAMO_ADDR", "0.0.0.0:8080")
	if _, err := Load(); err != nil {
		t.Fatal("authenticated public panel rejected")
	}
}
