package filesystem

import (
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
)

const MaxEditableFileSize = 1 << 20

type Entry struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
}

type ListResult struct {
	CurrentPath string  `json:"current_path"`
	ParentPath  string  `json:"parent_path"`
	Items       []Entry `json:"items"`
}

type ReadResult struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Mode       string    `json:"mode"`
	ModifiedAt time.Time `json:"modified_at"`
	Content    string    `json:"content"`
	IsBinary   bool      `json:"is_binary"`
}

type DownloadResult struct {
	File        *os.File
	Name        string
	ContentType string
	Size        int64
}

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) NormalizePath(raw string, allowRootDefault bool) (string, error) {
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

func (s *Service) List(targetPath string) (ListResult, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return ListResult{}, err
	}
	if !info.IsDir() {
		return ListResult{}, fmt.Errorf("目标不是目录")
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return ListResult{}, err
	}

	items := make([]Entry, 0, len(entries))
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

		items = append(items, Entry{
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

	return ListResult{
		CurrentPath: targetPath,
		ParentPath:  parentPath(targetPath),
		Items:       items,
	}, nil
}

func (s *Service) Read(targetPath string) (ReadResult, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return ReadResult{}, err
	}
	if info.IsDir() {
		return ReadResult{}, fmt.Errorf("目录不能直接打开")
	}
	if info.Size() > MaxEditableFileSize {
		return ReadResult{}, fmt.Errorf("文件过大，暂不支持在线编辑，请直接下载")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return ReadResult{}, err
	}

	result := ReadResult{
		Path:       targetPath,
		Name:       filepath.Base(targetPath),
		Size:       info.Size(),
		Mode:       info.Mode().String(),
		ModifiedAt: info.ModTime(),
		IsBinary:   looksBinary(content),
	}
	if !result.IsBinary {
		result.Content = string(content)
	}
	return result, nil
}

func (s *Service) Write(targetPath, content string) error {
	return os.WriteFile(targetPath, []byte(content), 0o644)
}

func (s *Service) Create(targetPath, content string) error {
	if _, err := os.Stat(targetPath); err == nil {
		return fmt.Errorf("文件已存在")
	}
	return os.WriteFile(targetPath, []byte(content), 0o644)
}

func (s *Service) Mkdir(targetPath string) error {
	return os.Mkdir(targetPath, 0o755)
}

func (s *Service) Move(fromPath, toPath string) error {
	return os.Rename(fromPath, toPath)
}

func (s *Service) Delete(targetPath string) error {
	return os.RemoveAll(targetPath)
}

func (s *Service) PrepareDownload(targetPath string) (DownloadResult, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		return DownloadResult{}, err
	}
	if info.IsDir() {
		return DownloadResult{}, fmt.Errorf("暂不支持下载目录")
	}
	file, err := os.Open(targetPath)
	if err != nil {
		return DownloadResult{}, err
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(targetPath)))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
			_ = file.Close()
			return DownloadResult{}, seekErr
		}
	}

	return DownloadResult{
		File:        file,
		Name:        filepath.Base(targetPath),
		ContentType: contentType,
		Size:        info.Size(),
	}, nil
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
