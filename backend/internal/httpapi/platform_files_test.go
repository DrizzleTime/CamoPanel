package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"camopanel/backend/internal/config"
	"camopanel/backend/internal/testutil"
)

func TestServerFileHTTPBinaryAndBatch(t *testing.T) {
	root := t.TempDir()
	router := New(config.Config{DataDir: t.TempDir(), FilesRoot: root, Address: "127.0.0.1:8080"}, testutil.NewDocker())
	roots := callAs(router, "GET", "/api/v1/files/roots", "", "", "")
	mustCode(t, roots, 200)
	if !strings.Contains(roots.Body.String(), root) {
		t.Fatal("missing configured root")
	}
	payload := bytes.Repeat([]byte{0, 255, 13, 10, 128, 70}, 400000)
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	_ = writer.WriteField("root", "server")
	_ = writer.WriteField("path", filepath.Join(root, "二进制.bin"))
	part, err := writer.CreateFormFile("file", "二进制.bin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(payload)
	_ = writer.Close()
	req := httptest.NewRequest("POST", "/api/v1/files/upload", &buffer)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	uploaded := httptest.NewRecorder()
	router.ServeHTTP(uploaded, req)
	mustCode(t, uploaded, 201)
	filePath := url.QueryEscape(filepath.Join(root, "二进制.bin"))
	download := callAs(router, "GET", "/api/v1/files/download?root=server&path="+filePath, "", "", "")
	mustCode(t, download, 200)
	if !bytes.Equal(download.Body.Bytes(), payload) {
		t.Fatalf("binary truncated: %d / %d", download.Body.Len(), len(payload))
	}
	if !strings.Contains(download.Header().Get("Content-Disposition"), "filename*") {
		t.Fatal("download did not retain unicode filename")
	}
	req = httptest.NewRequest("GET", "/api/v1/files/download?root=server&path="+filePath, nil)
	req.Header.Set("Range", "bytes=10-29")
	partial := httptest.NewRecorder()
	router.ServeHTTP(partial, req)
	mustCode(t, partial, 206)
	if !bytes.Equal(partial.Body.Bytes(), payload[10:30]) {
		t.Fatal("bad range download")
	}
	body, _ := json.Marshal(map[string]any{"root": "server", "path": filepath.Join(root, "destination"), "directory": true})
	mustCode(t, callAs(router, "POST", "/api/v1/files/create", string(body), "", ""), 201)
	body, _ = json.Marshal(map[string]any{"root": "server", "paths": []string{filepath.Join(root, "二进制.bin"), filepath.Join(root, "missing.bin")}, "dest_root": "server", "dest": filepath.Join(root, "destination"), "move": false})
	transferred := callAs(router, "POST", "/api/v1/files/transfer", string(body), "", "")
	mustCode(t, transferred, 200)
	var result struct {
		Succeeded int
		Results   []struct{ Error string }
	}
	if err := json.Unmarshal(transferred.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || len(result.Results) != 2 || result.Results[1].Error == "" {
		t.Fatalf("batch result incorrect: %s", transferred.Body.String())
	}
	body, _ = json.Marshal(map[string]any{"root": "server", "path": filepath.Join(root, "二进制.bin"), "mode": "0600"})
	mustCode(t, callAs(router, "POST", "/api/v1/files/permissions", string(body), "", ""), 200)
	info, err := os.Stat(filepath.Join(root, "二进制.bin"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode not updated: %v", err)
	}
	body, _ = json.Marshal(map[string]any{"root": "server", "paths": []string{root, filepath.Join(root, "二进制.bin")}})
	deleted := callAs(router, "POST", "/api/v1/files/delete", string(body), "", "")
	mustCode(t, deleted, 200)
	if err := json.Unmarshal(deleted.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Succeeded != 1 || result.Results[0].Error == "" {
		t.Fatalf("root deletion accepted: %s", deleted.Body.String())
	}
}

func TestViewerCannotModifyServerFiles(t *testing.T) {
	root := t.TempDir()
	router := New(config.Config{DataDir: t.TempDir(), FilesRoot: root, Address: "127.0.0.1:8080"}, testutil.NewDocker())
	mustCode(t, callAs(router, "POST", "/api/v1/auth/bootstrap", `{"password":"admin-password-123"}`, "", ""), 201)
	login := callAs(router, "POST", "/api/v1/auth/login", `{"username":"admin","password":"admin-password-123"}`, "", "")
	mustCode(t, login, 200)
	admin := sessionCookie(t, login)
	mustCode(t, callAs(router, "POST", "/api/v1/auth/viewer", `{"password":"viewer-password-123"}`, "", admin), 201)
	login = callAs(router, "POST", "/api/v1/auth/login", `{"username":"viewer","password":"viewer-password-123"}`, "", "")
	mustCode(t, login, 200)
	viewer := sessionCookie(t, login)
	mustCode(t, callAs(router, "GET", "/api/v1/files?root=server&path="+url.QueryEscape(root), "", "", viewer), 200)
	for _, endpoint := range []string{"create", "delete", "transfer", "permissions", "upload", "rename", "zip", "unzip"} {
		mustCode(t, callAs(router, "POST", "/api/v1/files/"+endpoint, `{}`, "", viewer), 403)
	}
	mustCode(t, callAs(router, "PUT", "/api/v1/files", `{}`, "", viewer), 403)
	roots := callAs(router, "GET", "/api/v1/files/roots", "", "", viewer)
	mustCode(t, roots, 200)
	if !strings.Contains(roots.Body.String(), `"read_only":true`) {
		t.Fatal(roots.Body.String())
	}
}

func TestDownloadLinkRequiresAuthAndIsBoundToFile(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	if err := os.WriteFile(first, []byte("first"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0644); err != nil {
		t.Fatal(err)
	}
	router := New(config.Config{DataDir: t.TempDir(), FilesRoot: root, Token: "download-token", Address: "127.0.0.1:8080"}, testutil.NewDocker())
	endpoint := "/api/v1/files/download-link?root=server&path=" + url.QueryEscape(first)
	mustCode(t, callAs(router, "GET", endpoint, "", "", ""), 401)
	issued := callAs(router, "GET", endpoint, "", "download-token", "")
	mustCode(t, issued, 200)
	var link struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(issued.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	// Neither additional path parameters nor a different scope can widen a ticket.
	downloaded := callAs(router, "GET", link.URL+"&root=server&path="+url.QueryEscape(second), "", "", "")
	mustCode(t, downloaded, 200)
	if downloaded.Body.String() != "first" {
		t.Fatalf("ticket switched file: %s", downloaded.Body.String())
	}
	mustCode(t, callAs(router, "GET", link.URL, "", "", ""), 410)
	mustCode(t, callAs(router, "GET", "/api/v1/files/download?ticket=invalid", "", "", ""), 410)
	store := newFileDownloads()
	ticket, err := store.issue(downloadClaim{root: "server", path: first})
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	claim := store.links[ticket]
	claim.expires = time.Now().Add(-time.Second)
	store.links[ticket] = claim
	store.mu.Unlock()
	if _, ok := store.take(ticket); ok {
		t.Fatal("expired download ticket accepted")
	}
}
