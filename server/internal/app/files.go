package app

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"camopanel/server/internal/model"

	"github.com/gin-gonic/gin"
)

const maxEditableFileSize = 1 << 20

type fileEntryResponse struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
}

type fileListResponse struct {
	CurrentPath string              `json:"current_path"`
	ParentPath  string              `json:"parent_path"`
	Items       []fileEntryResponse `json:"items"`
}

type fileReadResponse struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
	Content    string    `json:"content"`
	IsBinary   bool      `json:"is_binary"`
}

type fileWriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type fileCreateRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type fileMkdirRequest struct {
	Path string `json:"path"`
}

type fileMoveRequest struct {
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}

type fileDeleteRequest struct {
	Path string `json:"path"`
}

func (a *App) handleFileList(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	targetPath, err := normalizeAbsolutePath(c.Query("path"), true)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	items, err := listFiles(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(c, http.StatusNotFound, "路径不存在")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, fileListResponse{
		CurrentPath: targetPath,
		ParentPath:  parentPath(targetPath),
		Items:       items,
	})
}

func (a *App) handleFileRead(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	targetPath, err := normalizeAbsolutePath(c.Query("path"), false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	response, err := readFile(targetPath)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			writeError(c, http.StatusNotFound, "文件不存在")
		default:
			writeError(c, http.StatusBadRequest, err.Error())
		}
		return
	}

	c.JSON(http.StatusOK, response)
}

func (a *App) handleFileWrite(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req fileWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	targetPath, err := normalizeAbsolutePath(req.Path, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.WriteFile(targetPath, []byte(req.Content), 0o644); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "file_write", "file", targetPath, nil)
	c.JSON(http.StatusOK, gin.H{"path": targetPath})
}

func (a *App) handleFileCreate(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req fileCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	targetPath, err := normalizeAbsolutePath(req.Path, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(targetPath); err == nil {
		writeError(c, http.StatusBadRequest, "文件已存在")
		return
	}

	if err := os.WriteFile(targetPath, []byte(req.Content), 0o644); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "file_create", "file", targetPath, nil)
	c.JSON(http.StatusOK, gin.H{"path": targetPath})
}

func (a *App) handleFileMkdir(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req fileMkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	targetPath, err := normalizeAbsolutePath(req.Path, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.Mkdir(targetPath, 0o755); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "file_mkdir", "file", targetPath, nil)
	c.JSON(http.StatusOK, gin.H{"path": targetPath})
}

func (a *App) handleFileMove(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req fileMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	fromPath, err := normalizeAbsolutePath(req.FromPath, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	toPath, err := normalizeAbsolutePath(req.ToPath, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.Rename(fromPath, toPath); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "file_move", "file", fromPath, map[string]any{"to_path": toPath})
	c.JSON(http.StatusOK, gin.H{"path": toPath})
}

func (a *App) handleFileDelete(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req fileDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	targetPath, err := normalizeAbsolutePath(req.Path, false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(targetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(c, http.StatusNotFound, "路径不存在")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := os.RemoveAll(targetPath); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "file_delete", "file", targetPath, nil)
	c.JSON(http.StatusOK, gin.H{"path": targetPath})
}

func (a *App) handleFileUpload(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	targetPath, err := normalizeAbsolutePath(c.PostForm("path"), false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(c, http.StatusNotFound, "目录不存在")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if !info.IsDir() {
		writeError(c, http.StatusBadRequest, "上传目标必须是目录")
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		writeError(c, http.StatusBadRequest, "上传数据格式错误")
		return
	}

	files := append(form.File["files"], form.File["file"]...)
	if len(files) == 0 {
		writeError(c, http.StatusBadRequest, "请选择要上传的文件")
		return
	}

	uploaded := make([]string, 0, len(files))
	for _, fileHeader := range files {
		filename := filepath.Base(fileHeader.Filename)
		savePath := filepath.Join(targetPath, filename)
		if err := c.SaveUploadedFile(fileHeader, savePath); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}
		uploaded = append(uploaded, savePath)
	}

	_ = a.recordAudit(currentUser(c).ID, "file_upload", "file", targetPath, map[string]any{"count": len(uploaded)})
	c.JSON(http.StatusOK, gin.H{"items": uploaded})
}

func (a *App) handleFileDownload(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	targetPath, err := normalizeAbsolutePath(c.Query("path"), false)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeError(c, http.StatusNotFound, "文件不存在")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	if info.IsDir() {
		writeError(c, http.StatusBadRequest, "暂不支持下载目录")
		return
	}

	file, err := os.Open(targetPath)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	defer file.Close()

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(targetPath)))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			writeError(c, http.StatusInternalServerError, seekErr.Error())
			return
		}
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size()))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filepath.Base(targetPath)))
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, file)
}

func requireSuperAdmin(c *gin.Context) bool {
	if currentUser(c).Role != model.RoleSuperAdmin {
		writeError(c, http.StatusForbidden, "没有权限")
		return false
	}
	return true
}

func normalizeAbsolutePath(raw string, allowRootDefault bool) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		if allowRootDefault {
			return string(filepath.Separator), nil
		}
		return "", fmt.Errorf("路径不能为空")
	}

	if !filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("只支持绝对路径")
	}

	return filepath.Clean(trimmed), nil
}

func listFiles(targetPath string) ([]fileEntryResponse, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("目标不是目录")
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, err
	}

	items := make([]fileEntryResponse, 0, len(entries))
	for _, entry := range entries {
		entryInfo, err := entry.Info()
		if err != nil {
			continue
		}

		itemType := "file"
		switch {
		case entry.IsDir():
			itemType = "directory"
		case entry.Type()&os.ModeSymlink != 0:
			itemType = "symlink"
		}

		items = append(items, fileEntryResponse{
			Name:       entry.Name(),
			Path:       filepath.Join(targetPath, entry.Name()),
			Type:       itemType,
			Size:       entryInfo.Size(),
			Mode:       entryInfo.Mode().String(),
			ModifiedAt: entryInfo.ModTime(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Type == items[j].Type {
			return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		}
		if items[i].Type == "directory" {
			return true
		}
		if items[j].Type == "directory" {
			return false
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return items, nil
}

func readFile(targetPath string) (fileReadResponse, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return fileReadResponse{}, err
	}
	if info.IsDir() {
		return fileReadResponse{}, fmt.Errorf("目录不能直接打开")
	}
	if info.Size() > maxEditableFileSize {
		return fileReadResponse{}, fmt.Errorf("文件过大，暂不支持在线编辑，请直接下载")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return fileReadResponse{}, err
	}

	response := fileReadResponse{
		Path:       targetPath,
		Name:       filepath.Base(targetPath),
		Size:       info.Size(),
		Mode:       info.Mode().String(),
		ModifiedAt: info.ModTime(),
		IsBinary:   looksBinary(content),
	}
	if !response.IsBinary {
		response.Content = string(content)
	}

	return response, nil
}

func parentPath(targetPath string) string {
	if targetPath == string(filepath.Separator) {
		return ""
	}

	parent := filepath.Dir(targetPath)
	if parent == "." || parent == targetPath {
		return string(filepath.Separator)
	}
	return parent
}

func looksBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}
	if !utf8.Valid(content) {
		return true
	}
	for _, b := range content {
		if b == 0 {
			return true
		}
	}
	return false
}
