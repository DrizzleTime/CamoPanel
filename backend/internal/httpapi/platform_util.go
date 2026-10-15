package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func timeNow() time.Time { return time.Now().UTC() }

func writeScript(dataDir, name, content string) error {
	dir := filepath.Join(dataDir, "cron", "scripts")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

func rotateLogs(dataDir string) (string, error) {
	dir := filepath.Join(dataDir, "nginx", "logs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "没有日志", nil
		}
		return "", err
	}
	n := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		_ = os.Rename(path, path+".1")
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			return "", err
		}
		n++
	}
	return fmt.Sprintf("rotated %d", n), nil
}

func fetchURL(ctx context.Context, raw string) (string, error) {
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return "", fmt.Errorf("只允许 http 或 https")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.Status, nil
}
