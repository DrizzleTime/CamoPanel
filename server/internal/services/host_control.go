package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const DockerDaemonConfigPath = "/etc/docker/daemon.json"

var ErrHostControlUnavailable = errors.New("host control is unavailable")

type DockerDaemonSettings struct {
	RegistryMirrors []string `json:"registry_mirrors"`
	ControlEnabled  bool     `json:"control_enabled"`
	ConfigPath      string   `json:"config_path"`
	Message         string   `json:"message"`
}

type HostControlService struct {
	helperPath string
}

func NewHostControlService(helperPath string) *HostControlService {
	return &HostControlService{helperPath: strings.TrimSpace(helperPath)}
}

func (s *HostControlService) GetDockerSettings(ctx context.Context) (DockerDaemonSettings, error) {
	settings := DockerDaemonSettings{
		RegistryMirrors: []string{},
		ControlEnabled:  s.available(),
		ConfigPath:      DockerDaemonConfigPath,
	}
	if !settings.ControlEnabled {
		settings.Message = "宿主机控制未启用，请重新执行安装脚本以安装 hostctl 和 sudoers。"
		return settings, nil
	}

	raw, err := s.run(ctx, nil, "docker-settings", "read")
	if err != nil {
		return DockerDaemonSettings{}, err
	}
	mirrors, err := parseRegistryMirrors([]byte(raw))
	if err != nil {
		return DockerDaemonSettings{}, err
	}
	settings.RegistryMirrors = mirrors
	settings.Message = "当前镜像源来自 Docker daemon 配置。"
	return settings, nil
}

func (s *HostControlService) UpdateDockerSettings(ctx context.Context, mirrors []string) (DockerDaemonSettings, error) {
	if !s.available() {
		return DockerDaemonSettings{}, ErrHostControlUnavailable
	}

	current, err := s.run(ctx, nil, "docker-settings", "read")
	if err != nil {
		return DockerDaemonSettings{}, err
	}
	nextContent, err := updateRegistryMirrors([]byte(current), mirrors)
	if err != nil {
		return DockerDaemonSettings{}, err
	}
	if _, err := s.run(ctx, nextContent, "docker-settings", "write"); err != nil {
		return DockerDaemonSettings{}, err
	}

	return s.GetDockerSettings(ctx)
}

func (s *HostControlService) RestartDocker(ctx context.Context) error {
	if !s.available() {
		return ErrHostControlUnavailable
	}
	_, err := s.run(ctx, nil, "docker", "restart")
	return err
}

func (s *HostControlService) available() bool {
	if s.helperPath == "" {
		return false
	}
	if _, err := exec.LookPath("sudo"); err != nil {
		return false
	}
	if _, err := exec.LookPath(s.helperPath); err != nil {
		return false
	}
	return true
}

func (s *HostControlService) run(ctx context.Context, stdin []byte, args ...string) (string, error) {
	if !s.available() {
		return "", ErrHostControlUnavailable
	}

	cmdArgs := append([]string{"-n", s.helperPath}, args...)
	cmd := exec.CommandContext(ctx, "sudo", cmdArgs...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			trimmed = err.Error()
		}
		return "", fmt.Errorf("%w: %s", ErrHostControlUnavailable, trimmed)
	}
	return strings.TrimSpace(string(output)), nil
}

func parseRegistryMirrors(raw []byte) ([]string, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return []string{}, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, fmt.Errorf("Docker daemon 配置不是有效 JSON: %w", err)
	}

	value, ok := payload["registry-mirrors"]
	if !ok || value == nil {
		return []string{}, nil
	}

	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("Docker daemon 配置中的 registry-mirrors 必须是数组")
	}

	mirrors := make([]string, 0, len(items))
	for _, item := range items {
		mirror := strings.TrimSpace(fmt.Sprint(item))
		if mirror == "" || mirror == "<nil>" {
			continue
		}
		mirrors = append(mirrors, mirror)
	}
	sort.Strings(mirrors)
	return mirrors, nil
}

func updateRegistryMirrors(raw []byte, mirrors []string) ([]byte, error) {
	payload := map[string]any{}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
			return nil, fmt.Errorf("Docker daemon 配置不是有效 JSON: %w", err)
		}
	}

	normalized := normalizeRegistryMirrors(mirrors)
	if len(normalized) == 0 {
		delete(payload, "registry-mirrors")
	} else {
		payload["registry-mirrors"] = normalized
	}

	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}

func normalizeRegistryMirrors(mirrors []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(mirrors))
	for _, item := range mirrors {
		mirror := strings.TrimSpace(item)
		if mirror == "" {
			continue
		}
		if _, ok := seen[mirror]; ok {
			continue
		}
		seen[mirror] = struct{}{}
		normalized = append(normalized, mirror)
	}
	sort.Strings(normalized)
	return normalized
}
