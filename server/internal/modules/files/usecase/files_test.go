package usecase_test

import (
	"os"
	"path/filepath"
	"testing"

	filesusecase "camopanel/server/internal/modules/files/usecase"
	platformfilesystem "camopanel/server/internal/platform/filesystem"
)

func TestServiceListReadWriteLifecycle(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "demo.txt")
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	svc := filesusecase.NewService(platformfilesystem.NewService())

	list, err := svc.List(root)
	if err != nil {
		t.Fatalf("list files: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Name != "demo.txt" {
		t.Fatalf("unexpected list result: %+v", list.Items)
	}

	read, err := svc.Read(target)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if read.Content != "hello" {
		t.Fatalf("expected hello, got %q", read.Content)
	}

	if _, err := svc.Write(target, "updated"); err != nil {
		t.Fatalf("write file: %v", err)
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(raw) != "updated" {
		t.Fatalf("expected updated, got %q", string(raw))
	}
}
