package files

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func serverManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	manager, err := NewWithServerRoot(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	return manager, root
}
func requireOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestServerPathsAndMetadata(t *testing.T) {
	manager, root := serverManager(t)
	path := filepath.Join(root, "nested", "file.txt")
	requireOK(t, manager.Write("server", path, "hello"))
	requireOK(t, os.Chmod(path, 0664))
	requireOK(t, manager.Write("server", path, "updated"))
	info, err := os.Stat(path)
	requireOK(t, err)
	if info.Mode().Perm() != 0664 {
		t.Fatalf("editing changed permissions: %o", info.Mode().Perm())
	}
	entries, err := manager.List("server", filepath.Dir(path))
	requireOK(t, err)
	if len(entries) != 1 || entries[0].Path != path || entries[0].Modified == "" || entries[0].Permissions != "0664" || entries[0].Owner == "" {
		t.Fatalf("incomplete metadata: %+v", entries)
	}
	for _, path := range []string{"../outside", filepath.Join(filepath.Dir(root), "outside")} {
		if _, err := manager.Resolve("server", path); err == nil {
			t.Fatalf("escaped root: %q", path)
		}
	}
	for _, operation := range []func() error{func() error { return manager.Delete("server", root) }, func() error { return manager.Rename("server", root, root+"-moved") }, func() error { return manager.Chmod("server", root, "0777") }} {
		if err := operation(); err == nil {
			t.Fatal("modified server root")
		}
	}
	requireOK(t, manager.Write("server", filepath.Join(root, "name..txt"), "valid filename"))
	if _, err := manager.Read("sites", "/etc/passwd"); err == nil {
		t.Fatal("application root accepted absolute path")
	}
	if err := manager.Write("sites", "secret.key", "private"); err == nil {
		t.Fatal("application key was writable")
	}
}

func TestSymlinkBoundariesAndDeletion(t *testing.T) {
	manager, root := serverManager(t)
	outside := t.TempDir()
	requireOK(t, os.WriteFile(filepath.Join(outside, "protected.txt"), []byte("protected"), 0600))
	link := filepath.Join(root, "outside")
	requireOK(t, os.Symlink(outside, link))
	if _, err := manager.Read("server", filepath.Join(link, "protected.txt")); err == nil {
		t.Fatal("read escaped symlink")
	}
	if err := manager.Write("server", filepath.Join(link, "new.txt"), "bad"); err == nil {
		t.Fatal("write escaped symlink")
	}
	if err := manager.Delete("server", filepath.Join(link, "protected.txt")); err == nil {
		t.Fatal("delete escaped symlink")
	}
	requireOK(t, manager.Delete("server", link))
	content, err := os.ReadFile(filepath.Join(outside, "protected.txt"))
	requireOK(t, err)
	if string(content) != "protected" {
		t.Fatal("deleting a link deleted its target")
	}
	target := filepath.Join(root, "real")
	requireOK(t, os.Mkdir(target, 0755))
	requireOK(t, os.Symlink(target, filepath.Join(root, "alias")))
	requireOK(t, manager.Write("server", filepath.Join(root, "alias", "inside.txt"), "inside"))
	got, err := manager.Read("server", filepath.Join(root, "alias", "inside.txt"))
	requireOK(t, err)
	if got != "inside" {
		t.Fatal(got)
	}
	entries, err := manager.List("server", root)
	requireOK(t, err)
	found := false
	for _, entry := range entries {
		if entry.Name == "alias" {
			found = entry.Symlink && entry.Dir && entry.LinkTarget == target
		}
	}
	if !found {
		t.Fatalf("missing directory link: %+v", entries)
	}
}

func TestBinaryUploadAndAtomicFailure(t *testing.T) {
	manager, root := serverManager(t)
	payload := bytes.Repeat([]byte{0, 1, 255, 128, 13, 10}, 400000)
	path := filepath.Join(root, "payload.bin")
	requireOK(t, manager.Upload("server", path, bytes.NewReader(payload), false))
	file, err := manager.Open("server", path)
	requireOK(t, err)
	got, err := io.ReadAll(file)
	file.Close()
	requireOK(t, err)
	if !bytes.Equal(got, payload) {
		t.Fatalf("binary changed: got %d want %d", len(got), len(payload))
	}
	if _, err := manager.Read("server", path); err == nil {
		t.Fatal("binary opened as text")
	}
	if err := manager.Upload("server", path, strings.NewReader("overwrite"), false); err == nil {
		t.Fatal("upload replaced file without permission")
	}
	manager.mu.Lock()
	err = manager.writeStream("server", path, strings.NewReader("too large"), 3, true)
	manager.mu.Unlock()
	if err == nil {
		t.Fatal("oversize upload accepted")
	}
	got, err = os.ReadFile(path)
	requireOK(t, err)
	if !bytes.Equal(got, payload) {
		t.Fatal("failed upload corrupted original")
	}
	matches, _ := filepath.Glob(filepath.Join(root, ".camopanel-*"))
	if len(matches) > 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestCopyMoveAndConflicts(t *testing.T) {
	manager, root := serverManager(t)
	source := filepath.Join(root, "source")
	requireOK(t, manager.Mkdir("server", filepath.Join(source, "empty")))
	requireOK(t, manager.Write("server", filepath.Join(source, "hello.txt"), "hello"))
	requireOK(t, os.Symlink("hello.txt", filepath.Join(source, "hello-link")))
	copied := filepath.Join(root, "copy")
	requireOK(t, manager.Transfer("server", source, "server", copied, false))
	got, err := manager.Read("server", filepath.Join(copied, "hello.txt"))
	requireOK(t, err)
	if got != "hello" {
		t.Fatal(got)
	}
	link, err := os.Readlink(filepath.Join(copied, "hello-link"))
	requireOK(t, err)
	if link != "hello.txt" {
		t.Fatal(link)
	}
	if err := manager.Transfer("server", source, "server", copied, true); err == nil {
		t.Fatal("move overwrote destination")
	}
	if err := manager.Transfer("server", source, "server", filepath.Join(source, "nested"), false); err == nil {
		t.Fatal("directory copied into itself")
	}
	moved := filepath.Join(root, "moved")
	requireOK(t, manager.Transfer("server", source, "server", moved, true))
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	requireOK(t, manager.Transfer("server", filepath.Join(moved, "hello.txt"), "apps", "cross-root.txt", true))
	if _, err := os.Stat(filepath.Join(moved, "hello.txt")); !os.IsNotExist(err) {
		t.Fatal("cross-root move retained source")
	}
	got, err = manager.Read("apps", "cross-root.txt")
	requireOK(t, err)
	if got != "hello" {
		t.Fatal(got)
	}
}

func TestZIPRoundTripAndTraversal(t *testing.T) {
	manager, root := serverManager(t)
	requireOK(t, manager.Mkdir("server", filepath.Join(root, "data", "empty")))
	requireOK(t, manager.Write("server", filepath.Join(root, "data", "hello.txt"), "hello"))
	archive := filepath.Join(root, "data", "inside.zip")
	requireOK(t, manager.Archive("server", []string{filepath.Join(root, "data")}, archive))
	requireOK(t, manager.Unzip("server", archive, filepath.Join(root, "extracted")))
	got, err := manager.Read("server", filepath.Join(root, "extracted", "data", "hello.txt"))
	requireOK(t, err)
	if got != "hello" {
		t.Fatal(got)
	}
	if _, err := os.Stat(filepath.Join(root, "extracted", "data", "empty")); err != nil {
		t.Fatal("empty directory was lost")
	}
	if err := manager.Unzip("server", archive, filepath.Join(root, "extracted")); err == nil {
		t.Fatal("extraction overwrote an existing file")
	}
	for _, name := range []string{"../outside.txt", "/absolute.txt", "folder/../../outside.txt"} {
		var buffer bytes.Buffer
		writer := zip.NewWriter(&buffer)
		file, err := writer.Create(name)
		requireOK(t, err)
		_, err = file.Write([]byte("bad"))
		requireOK(t, err)
		requireOK(t, writer.Close())
		requireOK(t, manager.Upload("server", filepath.Join(root, "unsafe.zip"), bytes.NewReader(buffer.Bytes()), true))
		if err := manager.Unzip("server", filepath.Join(root, "unsafe.zip"), filepath.Join(root, "safe")); err == nil {
			t.Fatalf("unsafe archive extracted %q", name)
		}
	}
}
