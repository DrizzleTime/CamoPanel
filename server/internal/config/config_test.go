package config

import (
	"path/filepath"
	"testing"
)

func TestLoadResolvesPathsToAbsolute(t *testing.T) {
	t.Setenv("CAMO_DATA_DIR", "./tmp-data")
	t.Setenv("CAMO_TEMPLATES_DIR", "../templates")
	t.Setenv("CAMO_ADMIN_PASSWORD", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, path := range []string{
		cfg.DataDir,
		cfg.DatabasePath,
		cfg.ProjectsDir,
		cfg.TemplatesDir,
		cfg.OpenRestyDataDir,
	} {
		if !filepath.IsAbs(path) {
			t.Fatalf("expected absolute path, got %s", path)
		}
	}
}
