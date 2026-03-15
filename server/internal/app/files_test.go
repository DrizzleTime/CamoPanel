package app

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"camopanel/server/internal/model"
)

func TestFileManagementLifecycle(t *testing.T) {
	instance := newTestApp(t)
	router := instance.router()
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	listResp := performAuthedRequest(t, instance, router, http.MethodGet, "/api/files/list?path="+url.QueryEscape(root), nil, "")
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", listResp.Code)
	}

	var listed fileListResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if listed.CurrentPath != root {
		t.Fatalf("expected current path %s, got %s", root, listed.CurrentPath)
	}
	if len(listed.Items) != 1 || listed.Items[0].Name != "hello.txt" {
		t.Fatalf("unexpected list items: %+v", listed.Items)
	}

	readResp := performAuthedRequest(t, instance, router, http.MethodGet, "/api/files/read?path="+url.QueryEscape(filepath.Join(root, "hello.txt")), nil, "")
	if readResp.Code != http.StatusOK {
		t.Fatalf("expected read status 200, got %d", readResp.Code)
	}

	var read fileReadResponse
	if err := json.Unmarshal(readResp.Body.Bytes(), &read); err != nil {
		t.Fatalf("unmarshal read response: %v", err)
	}
	if read.Content != "hello" {
		t.Fatalf("expected hello content, got %q", read.Content)
	}

	mkdirResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/mkdir",
		strings.NewReader(`{"path":"`+filepath.Join(root, "docs")+`"}`),
		"application/json",
	)
	if mkdirResp.Code != http.StatusOK {
		t.Fatalf("expected mkdir status 200, got %d", mkdirResp.Code)
	}

	createResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/create",
		strings.NewReader(`{"path":"`+filepath.Join(root, "docs", "note.txt")+`","content":"draft"}`),
		"application/json",
	)
	if createResp.Code != http.StatusOK {
		t.Fatalf("expected create status 200, got %d", createResp.Code)
	}

	writeResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/write",
		strings.NewReader(`{"path":"`+filepath.Join(root, "docs", "note.txt")+`","content":"updated"}`),
		"application/json",
	)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("expected write status 200, got %d", writeResp.Code)
	}

	rawFile, err := os.ReadFile(filepath.Join(root, "docs", "note.txt"))
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(rawFile) != "updated" {
		t.Fatalf("expected updated file content, got %q", string(rawFile))
	}

	moveResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/move",
		strings.NewReader(`{"from_path":"`+filepath.Join(root, "docs", "note.txt")+`","to_path":"`+filepath.Join(root, "docs", "renamed.txt")+`"}`),
		"application/json",
	)
	if moveResp.Code != http.StatusOK {
		t.Fatalf("expected move status 200, got %d", moveResp.Code)
	}

	if _, err := os.Stat(filepath.Join(root, "docs", "renamed.txt")); err != nil {
		t.Fatalf("expected moved file to exist: %v", err)
	}

	deleteResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/delete",
		strings.NewReader(`{"path":"`+filepath.Join(root, "docs", "renamed.txt")+`"}`),
		"application/json",
	)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d", deleteResp.Code)
	}

	if _, err := os.Stat(filepath.Join(root, "docs", "renamed.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected deleted file to disappear, got err=%v", err)
	}
}

func TestFileUploadAndDownload(t *testing.T) {
	instance := newTestApp(t)
	router := instance.router()
	root := t.TempDir()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("path", root); err != nil {
		t.Fatalf("write multipart path: %v", err)
	}
	part, err := writer.CreateFormFile("files", "upload.txt")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := io.WriteString(part, "upload-body"); err != nil {
		t.Fatalf("write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	uploadResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodPost,
		"/api/files/upload",
		body,
		writer.FormDataContentType(),
	)
	if uploadResp.Code != http.StatusOK {
		t.Fatalf("expected upload status 200, got %d body=%s", uploadResp.Code, uploadResp.Body.String())
	}

	uploadedContent, err := os.ReadFile(filepath.Join(root, "upload.txt"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(uploadedContent) != "upload-body" {
		t.Fatalf("unexpected uploaded content: %q", string(uploadedContent))
	}

	downloadResp := performAuthedRequest(
		t,
		instance,
		router,
		http.MethodGet,
		"/api/files/download?path="+url.QueryEscape(filepath.Join(root, "upload.txt")),
		nil,
		"",
	)
	if downloadResp.Code != http.StatusOK {
		t.Fatalf("expected download status 200, got %d", downloadResp.Code)
	}
	if downloadResp.Body.String() != "upload-body" {
		t.Fatalf("unexpected download content: %q", downloadResp.Body.String())
	}
}

func TestFileAPIRequiresAbsolutePath(t *testing.T) {
	instance := newTestApp(t)
	router := instance.router()

	resp := performAuthedRequest(t, instance, router, http.MethodGet, "/api/files/list?path=relative/path", nil, "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d", resp.Code)
	}
}

func performAuthedRequest(t *testing.T, instance *App, router http.Handler, method, target string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, target, body)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.AddCookie(authCookie(t, instance))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func authCookie(t *testing.T, instance *App) *http.Cookie {
	t.Helper()

	var user model.User
	if err := instance.db.First(&user, "username = ?", instance.cfg.AdminUsername).Error; err != nil {
		t.Fatalf("load admin user: %v", err)
	}

	token, err := instance.auth.IssueSession(user)
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	return &http.Cookie{
		Name:  instance.cfg.CookieName,
		Value: token,
		Path:  "/",
	}
}
