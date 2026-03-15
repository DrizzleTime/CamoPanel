//go:build integration

package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerServiceDeployAndLogs(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker binary not found")
	}

	service := NewDockerService()
	if _, err := service.clientOrErr(context.Background()); err != nil {
		t.Skip("docker daemon unavailable")
	}

	root := t.TempDir()
	composePath := filepath.Join(root, "compose.yaml")
	projectName := "camopanel-int-test"

	compose := `services:
  echo:
    image: busybox:1.36
    command: ["sh", "-c", "echo hello from integration test && sleep 5"]
`
	if err := os.WriteFile(composePath, []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
	defer func() {
		_ = service.Delete(context.Background(), projectName, composePath)
	}()

	if err := service.Deploy(context.Background(), projectName, composePath); err != nil {
		t.Fatalf("deploy: %v", err)
	}

	logs, err := service.ProjectLogs(context.Background(), projectName, 50)
	if err != nil {
		t.Fatalf("project logs: %v", err)
	}
	if !strings.Contains(logs, "hello from integration test") {
		t.Fatalf("expected logs to include container output, got %s", logs)
	}
}
