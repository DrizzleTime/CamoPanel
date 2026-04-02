package system

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type CleanupItem struct {
	Size        int64  `json:"size"`
	Description string `json:"description"`
}

type CleanupScanResult struct {
	Items map[string]CleanupItem `json:"items"`
}

type CleanupResult struct {
	Cleaned int `json:"cleaned"`
}

func (s *Service) ScanCleanup(_ context.Context) (CleanupScanResult, error) {
	items := map[string]CleanupItem{}

	items["apt_cache"] = CleanupItem{
		Size:        dirSize("/var/cache/apt"),
		Description: "APT 软件包缓存",
	}

	yumSize := dirSize("/var/cache/yum")
	if yumSize == 0 {
		yumSize = dirSize("/var/cache/dnf")
	}
	items["yum_cache"] = CleanupItem{
		Size:        yumSize,
		Description: "YUM/DNF 软件包缓存",
	}

	items["journal_logs"] = CleanupItem{
		Size:        journalDiskUsage(),
		Description: "Systemd 日志",
	}

	items["tmp_files"] = CleanupItem{
		Size:        dirSize("/tmp"),
		Description: "临时文件 (/tmp)",
	}

	items["old_logs"] = CleanupItem{
		Size:        rotatedLogsSize("/var/log"),
		Description: "过期日志文件",
	}

	items["user_cache"] = CleanupItem{
		Size:        dirSize("/root/.cache"),
		Description: "用户缓存 (~/.cache)",
	}

	return CleanupScanResult{Items: items}, nil
}

func (s *Service) ExecuteCleanup(ctx context.Context, categories []string) (CleanupResult, error) {
	set := map[string]bool{}
	for _, c := range categories {
		set[c] = true
	}

	cleaned := 0

	if set["apt_cache"] {
		if path, _ := exec.LookPath("apt-get"); path != "" {
			_ = exec.CommandContext(ctx, "apt-get", "clean", "-y").Run()
			cleaned++
		}
	}

	if set["yum_cache"] {
		if path, _ := exec.LookPath("yum"); path != "" {
			_ = exec.CommandContext(ctx, "yum", "clean", "all").Run()
			cleaned++
		} else if path, _ := exec.LookPath("dnf"); path != "" {
			_ = exec.CommandContext(ctx, "dnf", "clean", "all").Run()
			cleaned++
		}
	}

	if set["journal_logs"] {
		if path, _ := exec.LookPath("journalctl"); path != "" {
			_ = exec.CommandContext(ctx, "journalctl", "--vacuum-time=3d").Run()
			cleaned++
		}
	}

	if set["tmp_files"] {
		cleanOldTmpFiles("/tmp", 24*time.Hour)
		cleaned++
	}

	if set["old_logs"] {
		cleanRotatedLogs("/var/log")
		cleaned++
	}

	if set["user_cache"] {
		entries, _ := os.ReadDir("/root/.cache")
		for _, e := range entries {
			_ = os.RemoveAll(filepath.Join("/root/.cache", e.Name()))
		}
		cleaned++
	}

	return CleanupResult{Cleaned: cleaned}, nil
}

// --- helpers ---

func dirSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return 0
	}

	var total int64
	_ = filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	return total
}

func journalDiskUsage() int64 {
	path, _ := exec.LookPath("journalctl")
	if path == "" {
		return 0
	}
	out, err := exec.Command("journalctl", "--disk-usage").Output()
	if err != nil {
		return 0
	}

	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		for i, f := range fields {
			if i == 0 {
				continue
			}
			size := parseHumanSize(f)
			if size > 0 {
				return size
			}
		}
	}

	return dirSize("/var/log/journal")
}

func parseHumanSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	multiplier := int64(1)
	upper := strings.ToUpper(s)
	switch {
	case strings.HasSuffix(upper, "G"):
		multiplier = 1 << 30
		s = s[:len(s)-1]
	case strings.HasSuffix(upper, "M"):
		multiplier = 1 << 20
		s = s[:len(s)-1]
	case strings.HasSuffix(upper, "K"):
		multiplier = 1 << 10
		s = s[:len(s)-1]
	case strings.HasSuffix(upper, "B"):
		s = s[:len(s)-1]
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(val * float64(multiplier))
}

func rotatedLogsSize(logDir string) int64 {
	var total int64
	_ = filepath.WalkDir(logDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := d.Name()
		if isRotatedLog(name) {
			if fi, err := d.Info(); err == nil {
				total += fi.Size()
			}
		}
		return nil
	})
	return total
}

func isRotatedLog(name string) bool {
	if strings.HasSuffix(name, ".gz") {
		return true
	}
	if strings.HasSuffix(name, ".old") {
		return true
	}
	if strings.HasSuffix(name, ".xz") {
		return true
	}
	if strings.HasSuffix(name, ".bz2") {
		return true
	}
	parts := strings.Split(name, ".")
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		if _, err := strconv.Atoi(last); err == nil {
			return true
		}
	}
	return false
}

func cleanOldTmpFiles(dir string, maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		full := filepath.Join(dir, e.Name())
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.ModTime().Before(cutoff) {
			_ = os.RemoveAll(full)
		}
	}
}

func cleanRotatedLogs(logDir string) {
	_ = filepath.WalkDir(logDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if isRotatedLog(d.Name()) {
			_ = os.Remove(path)
		}
		return nil
	})
}

func formatSize(bytes int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := float64(bytes)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unit])
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}
