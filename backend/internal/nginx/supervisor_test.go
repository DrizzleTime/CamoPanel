package nginx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"camopanel/backend/internal/testutil"
)

func TestSupervisorInitAndStatus(t *testing.T) {
	tmpDir := t.TempDir()
	fake := testutil.NewDocker()

	sup, err := NewSupervisor(tmpDir, fake, 8080, 8443)
	if err != nil {
		t.Fatalf("new supervisor failed: %v", err)
	}

	// Verify required directories and default confs created
	for _, p := range []string{
		filepath.Join(tmpDir, "nginx", "nginx.conf"),
		filepath.Join(tmpDir, "nginx", "conf.d", "00_default.conf"),
		filepath.Join(tmpDir, "sites"),
		filepath.Join(tmpDir, "ssl"),
		filepath.Join(tmpDir, "acme", ".well-known", "acme-challenge"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected path to exist: %s (%v)", p, err)
		}
	}

	ctx := context.Background()
	if err := sup.EnsureRunning(ctx); err != nil {
		t.Fatalf("ensure running failed: %v", err)
	}

	status := sup.GetStatus(ctx)
	if !status.Installed || !status.Running {
		t.Fatalf("expected supervisor to be installed and running: %+v", status)
	}

	if err := sup.Reload(ctx); err != nil {
		t.Fatalf("reload failed: %v", err)
	}
}
