package system

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v4/mem"
)

type SystemConfig struct {
	Hostname string   `json:"hostname"`
	DNS      []string `json:"dns"`
	Timezone string   `json:"timezone"`
	Swap     SwapInfo `json:"swap"`
}

type SwapInfo struct {
	Total int64  `json:"total"`
	Used  int64  `json:"used"`
	File  string `json:"file"`
}

func (s *Service) GetSystemConfig(ctx context.Context) (SystemConfig, error) {
	cfg := SystemConfig{DNS: []string{}}
	cfg.Hostname = readHostname()
	cfg.DNS = readDNS()
	cfg.Timezone = readTimezone(ctx)
	cfg.Swap = readSwapInfo()
	return cfg, nil
}

func (s *Service) UpdateHostname(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("主机名不能为空")
	}

	if path, _ := exec.LookPath("hostnamectl"); path != "" {
		if err := exec.CommandContext(ctx, "hostnamectl", "set-hostname", name).Run(); err != nil {
			return fmt.Errorf("设置主机名失败: %w", err)
		}
	} else {
		if err := os.WriteFile("/etc/hostname", []byte(name+"\n"), 0644); err != nil {
			return fmt.Errorf("写入 /etc/hostname 失败: %w", err)
		}
	}

	patchHostsFile(name)
	return nil
}

func (s *Service) UpdateDNS(_ context.Context, servers []string) error {
	if len(servers) == 0 {
		return fmt.Errorf("DNS 列表不能为空")
	}

	existing := readResolvConfNonNameserver()
	var buf strings.Builder
	for _, line := range existing {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	for _, srv := range servers {
		srv = strings.TrimSpace(srv)
		if srv != "" {
			buf.WriteString("nameserver ")
			buf.WriteString(srv)
			buf.WriteByte('\n')
		}
	}

	if err := writeFileAtomic("/etc/resolv.conf", []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("写入 /etc/resolv.conf 失败: %w", err)
	}
	return nil
}

func (s *Service) UpdateTimezone(ctx context.Context, tz string) error {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		return fmt.Errorf("时区不能为空")
	}

	zoneFile := filepath.Join("/usr/share/zoneinfo", tz)
	if _, err := os.Stat(zoneFile); err != nil {
		return fmt.Errorf("无效的时区: %s", tz)
	}

	if path, _ := exec.LookPath("timedatectl"); path != "" {
		if err := exec.CommandContext(ctx, "timedatectl", "set-timezone", tz).Run(); err != nil {
			return fmt.Errorf("设置时区失败: %w", err)
		}
	} else {
		if err := os.Remove("/etc/localtime"); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("移除 /etc/localtime 失败: %w", err)
		}
		if err := os.Symlink(zoneFile, "/etc/localtime"); err != nil {
			return fmt.Errorf("创建时区符号链接失败: %w", err)
		}
		_ = os.WriteFile("/etc/timezone", []byte(tz+"\n"), 0644)
	}
	return nil
}

func (s *Service) CreateSwap(ctx context.Context, sizeMB int) (SwapInfo, error) {
	if sizeMB < 64 {
		return SwapInfo{}, fmt.Errorf("Swap 最小 64 MB")
	}

	swapFile := "/swapfile"

	if isSwapActive(swapFile) {
		_ = exec.CommandContext(ctx, "swapoff", swapFile).Run()
	}

	dd := exec.CommandContext(ctx, "dd", "if=/dev/zero", "of="+swapFile,
		"bs=1M", fmt.Sprintf("count=%d", sizeMB), "status=none")
	if out, err := dd.CombinedOutput(); err != nil {
		return SwapInfo{}, fmt.Errorf("创建 swap 文件失败: %s", strings.TrimSpace(string(out)))
	}
	if err := os.Chmod(swapFile, 0600); err != nil {
		return SwapInfo{}, fmt.Errorf("chmod 失败: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "mkswap", swapFile).CombinedOutput(); err != nil {
		return SwapInfo{}, fmt.Errorf("mkswap 失败: %s", strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "swapon", swapFile).CombinedOutput(); err != nil {
		return SwapInfo{}, fmt.Errorf("swapon 失败: %s", strings.TrimSpace(string(out)))
	}

	ensureFstabEntry(swapFile)
	return readSwapInfo(), nil
}

func (s *Service) RemoveSwap(ctx context.Context) (SwapInfo, error) {
	swapFile := "/swapfile"

	if isSwapActive(swapFile) {
		if out, err := exec.CommandContext(ctx, "swapoff", swapFile).CombinedOutput(); err != nil {
			return SwapInfo{}, fmt.Errorf("swapoff 失败: %s", strings.TrimSpace(string(out)))
		}
	}
	_ = os.Remove(swapFile)
	removeFstabEntry(swapFile)
	return readSwapInfo(), nil
}

// --- helpers ---

func readHostname() string {
	name, _ := os.Hostname()
	return name
}

func readDNS() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()

	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}
	return servers
}

func readResolvConfNonNameserver() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "nameserver") {
			lines = append(lines, line)
		}
	}
	return lines
}

func readTimezone(ctx context.Context) string {
	if path, _ := exec.LookPath("timedatectl"); path != "" {
		out, err := exec.CommandContext(ctx, "timedatectl", "show", "--property=Timezone", "--value").Output()
		if err == nil {
			if tz := strings.TrimSpace(string(out)); tz != "" {
				return tz
			}
		}
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		if tz := strings.TrimSpace(string(data)); tz != "" {
			return tz
		}
	}
	if target, err := os.Readlink("/etc/localtime"); err == nil {
		const prefix = "/usr/share/zoneinfo/"
		if idx := strings.Index(target, prefix); idx >= 0 {
			return target[idx+len(prefix):]
		}
	}
	return "UTC"
}

func readSwapInfo() SwapInfo {
	info := SwapInfo{}
	if swap, err := mem.SwapMemory(); err == nil {
		info.Total = int64(swap.Total)
		info.Used = int64(swap.Used)
	}
	if _, err := os.Stat("/swapfile"); err == nil {
		info.File = "/swapfile"
	}
	return info
}

func isSwapActive(file string) bool {
	data, err := os.ReadFile("/proc/swaps")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), file)
}

func patchHostsFile(hostname string) {
	data, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "127.0.1.1") {
			lines[i] = "127.0.1.1\t" + hostname
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "127.0.1.1\t"+hostname)
	}
	_ = writeFileAtomic("/etc/hosts", []byte(strings.Join(lines, "\n")), 0644)
}

func ensureFstabEntry(swapFile string) {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return
	}
	if strings.Contains(string(data), swapFile) {
		return
	}
	entry := swapFile + " none swap sw 0 0\n"
	_ = os.WriteFile("/etc/fstab", append(data, []byte(entry)...), 0644)
}

func removeFstabEntry(swapFile string) {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.Contains(line, swapFile) {
			filtered = append(filtered, line)
		}
	}
	_ = writeFileAtomic("/etc/fstab", []byte(strings.Join(filtered, "\n")), 0644)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".camopanel-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}
