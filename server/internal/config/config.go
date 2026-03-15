package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type AIConfig struct {
	BaseURL string
	Model   string
	APIKey  string
}

type Config struct {
	HTTPAddr           string
	DataDir            string
	DatabasePath       string
	ProjectsDir        string
	TemplatesDir       string
	SessionSecret      string
	CookieName         string
	AdminUsername      string
	AdminPassword      string
	OpenRestyContainer string
	OpenRestyDataDir   string
	AI                 AIConfig
}

func Load() (Config, error) {
	dataDir := env("CAMO_DATA_DIR", filepath.Join(".", "data"))
	templatesDir := env("CAMO_TEMPLATES_DIR", detectTemplatesDir())
	sessionSecret := env("CAMO_SESSION_SECRET", "camo-dev-secret-change-me")

	cfg := Config{
		HTTPAddr:           env("CAMO_HTTP_ADDR", ":8080"),
		DataDir:            dataDir,
		DatabasePath:       filepath.Join(dataDir, "camopanel.db"),
		ProjectsDir:        filepath.Join(dataDir, "projects"),
		TemplatesDir:       templatesDir,
		SessionSecret:      sessionSecret,
		CookieName:         env("CAMO_COOKIE_NAME", "camopanel_session"),
		AdminUsername:      env("CAMO_ADMIN_USERNAME", "admin"),
		AdminPassword:      env("CAMO_ADMIN_PASSWORD", "admin123"),
		OpenRestyContainer: env("CAMO_OPENRESTY_CONTAINER", "camopanel-openresty"),
		OpenRestyDataDir:   filepath.Join(dataDir, "openresty"),
		AI: AIConfig{
			BaseURL: env("CAMO_AI_BASE_URL", ""),
			Model:   env("CAMO_AI_MODEL", ""),
			APIKey:  env("CAMO_AI_API_KEY", ""),
		},
	}

	if cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("CAMO_ADMIN_PASSWORD can not be empty")
	}

	return cfg, nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func detectTemplatesDir() string {
	candidates := []string{
		filepath.Join(".", "templates"),
		filepath.Join("..", "templates"),
	}

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	return filepath.Join(".", "templates")
}
