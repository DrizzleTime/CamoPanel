package usecase

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	platformfilesystem "camopanel/server/internal/platform/filesystem"
)

type FileSystem interface {
	NormalizePath(raw string, allowRootDefault bool) (string, error)
	List(targetPath string) (platformfilesystem.ListResult, error)
	Read(targetPath string) (platformfilesystem.ReadResult, error)
	Write(targetPath, content string) error
	Create(targetPath, content string) error
	Mkdir(targetPath string) error
	Move(fromPath, toPath string) error
	Delete(targetPath string) error
	PrepareDownload(targetPath string) (platformfilesystem.DownloadResult, error)
}

type Service struct {
	files FileSystem
}

func NewService(files FileSystem) *Service { return &Service{files: files} }

func (s *Service) List(rawPath string) (platformfilesystem.ListResult, error) {
	targetPath, err := s.files.NormalizePath(rawPath, true)
	if err != nil {
		return platformfilesystem.ListResult{}, err
	}
	return s.files.List(targetPath)
}

func (s *Service) Read(rawPath string) (platformfilesystem.ReadResult, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return platformfilesystem.ReadResult{}, err
	}
	return s.files.Read(targetPath)
}

func (s *Service) Write(rawPath, content string) (string, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return "", err
	}
	return targetPath, s.files.Write(targetPath, content)
}

func (s *Service) Create(rawPath, content string) (string, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return "", err
	}
	return targetPath, s.files.Create(targetPath, content)
}

func (s *Service) Mkdir(rawPath string) (string, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return "", err
	}
	return targetPath, s.files.Mkdir(targetPath)
}

func (s *Service) Move(rawFromPath, rawToPath string) (string, error) {
	fromPath, err := s.files.NormalizePath(rawFromPath, false)
	if err != nil {
		return "", err
	}
	toPath, err := s.files.NormalizePath(rawToPath, false)
	if err != nil {
		return "", err
	}
	return toPath, s.files.Move(fromPath, toPath)
}

func (s *Service) Delete(rawPath string) (string, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return "", err
	}
	return targetPath, s.files.Delete(targetPath)
}

func (s *Service) Upload(rawPath string, files []*multipart.FileHeader) ([]string, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("上传目标必须是目录")
	}

	result := make([]string, 0, len(files))
	for _, fileHeader := range files {
		filename := filepath.Base(fileHeader.Filename)
		savePath := filepath.Join(targetPath, filename)

		src, err := fileHeader.Open()
		if err != nil {
			return nil, err
		}
		dst, err := os.Create(savePath)
		if err != nil {
			src.Close()
			return nil, err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return nil, err
		}
		_ = dst.Close()
		_ = src.Close()
		result = append(result, savePath)
	}
	return result, nil
}

func (s *Service) Download(rawPath string) (platformfilesystem.DownloadResult, error) {
	targetPath, err := s.files.NormalizePath(rawPath, false)
	if err != nil {
		return platformfilesystem.DownloadResult{}, err
	}
	return s.files.PrepareDownload(targetPath)
}
