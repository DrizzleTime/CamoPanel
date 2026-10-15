package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"camopanel/backend/internal/docker"
)

var validAppNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{1,62}$`)

type Service struct {
	mu            sync.RWMutex
	dataDir       string
	dockerMode    string
	dockerService docker.Service
	installed     map[string]*InstalledApp
}

func NewService(dataDir, dockerMode string, dockerService docker.Service) (*Service, error) {
	absDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}
	s := &Service{
		dataDir:       absDataDir,
		dockerMode:    dockerMode,
		dockerService: dockerService,
		installed:     make(map[string]*InstalledApp),
	}
	if err := os.MkdirAll(filepath.Join(absDataDir, "apps"), 0755); err != nil {
		return nil, err
	}
	customDir := filepath.Join(absDataDir, "custom_apps")
	if loaded, err := LoadAllCatalog(customDir); err == nil && len(loaded) > 0 {
		Catalog = loaded
	}
	_ = s.loadPersisted()
	return s, nil
}

func (s *Service) metaFilePath() string {
	return filepath.Join(s.dataDir, "apps.json")
}

func (s *Service) loadPersisted() error {
	path := s.metaFilePath()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var list []*InstalledApp
	if err := json.Unmarshal(b, &list); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, app := range list {
		s.installed[app.ID] = app
	}
	return nil
}

func (s *Service) savePersisted() error {
	list := make([]*InstalledApp, 0, len(s.installed))
	for _, app := range s.installed {
		list = append(list, app)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaFilePath(), b, 0644)
}

func (s *Service) ListCatalog(ctx context.Context) ([]StoreApp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]StoreApp, len(Catalog))
	for i, item := range Catalog {
		app := item
		for _, inst := range s.installed {
			if inst.AppID == app.ID {
				app.Installed = true
				break
			}
		}
		result[i] = app
	}
	return result, nil
}

func (s *Service) GetCatalogApp(id string) (*StoreApp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range Catalog {
		if item.ID == id {
			app := item
			for _, inst := range s.installed {
				if inst.AppID == app.ID {
					app.Installed = true
					break
				}
			}
			return &app, nil
		}
	}
	return nil, fmt.Errorf("应用商店中未找到该应用: %s", id)
}

func (s *Service) ListInstalled(ctx context.Context) ([]*InstalledApp, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]*InstalledApp, 0, len(s.installed))
	for _, app := range s.installed {
		// update status based on container presence if dockerService is available
		if s.dockerService != nil && len(app.ContainerNames) > 0 {
			anyRunning := false
			for _, cname := range app.ContainerNames {
				c, err := s.dockerService.FindContainerByName(ctx, cname)
				if err == nil && c != nil && c.State == "running" {
					anyRunning = true
					break
				}
			}
			if anyRunning {
				app.Status = "running"
			} else {
				app.Status = "stopped"
			}
		}
		list = append(list, app)
	}
	return list, nil
}

func (s *Service) GetInstalled(id string) (*InstalledApp, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.installed[id]
	if !ok {
		return nil, fmt.Errorf("应用未安装或不存在: %s", id)
	}
	return app, nil
}

func (s *Service) Install(ctx context.Context, req InstallRequest) (*InstalledApp, error) {
	if !validAppNamePattern.MatchString(req.Name) {
		return nil, fmt.Errorf("应用安装名称不合法，须为 2-63 位字母、数字、点、横线或下划线")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.installed[req.Name]; exists {
		return nil, fmt.Errorf("应用名称 [%s] 已经安装，请更换名称", req.Name)
	}

	var template *StoreApp
	for _, item := range Catalog {
		if item.ID == req.AppID {
			template = &item
			break
		}
	}
	if template == nil && req.CustomCompose == "" {
		return nil, fmt.Errorf("未知的应用模板: %s", req.AppID)
	}

	appDir, err := AppDirectory(s.dataDir, req.Name)
	if err != nil {
		return nil, err
	}

	// Enforce strictly constrained subdirectories:
	// <appDir>/data, <appDir>/config, <appDir>/logs
	if err := EnsureAppDirectoryConstraints(appDir); err != nil {
		return nil, err
	}
	if template != nil {
		for _, sub := range template.DataDirs {
			if sub != "" && !strings.Contains(sub, "..") {
				_ = os.MkdirAll(filepath.Join(appDir, sub), 0755)
			}
		}
	}

	composeContent := req.CustomCompose
	if composeContent == "" && template != nil {
		composeContent = template.ComposeTemplate
	}

	// Replace variables in compose template
	vars := make(map[string]string)
	vars["APP_NAME"] = req.Name
	vars["APP_DIR"] = appDir
	finalPorts := make(map[string]int)

	if template != nil {
		for k, v := range template.DefaultPorts {
			vars[k] = strconv.Itoa(v)
			finalPorts[k] = v
		}
		for _, p := range template.EnvParams {
			if p.Default != "" {
				vars[p.Key] = p.Default
			}
		}
		// Rich params auto_generate & default evaluation
		for _, p := range template.Params {
			val := ""
			if p.Default != nil {
				val = fmt.Sprintf("%v", p.Default)
			}
			if p.AutoGenerate && (val == "auto" || val == "") {
				val = GenerateRandomPassword(16)
			}
			if val != "" {
				vars[p.Key] = val
			}
		}
	}
	for k, v := range req.Ports {
		vars[k] = strconv.Itoa(v)
		finalPorts[k] = v
	}
	for k, v := range req.Env {
		if v == "auto" {
			v = GenerateRandomPassword(16)
		}
		vars[k] = v
	}

	for k, v := range vars {
		composeContent = strings.ReplaceAll(composeContent, fmt.Sprintf("${%s}", k), v)
	}

	// Validate volume constraints: all host volume mounts MUST be inside appDir
	if err := ValidateComposeVolumes(appDir, composeContent); err != nil {
		return nil, err
	}

	composePath := filepath.Join(appDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return nil, fmt.Errorf("写入 compose 文件失败: %w", err)
	}

	// Write .env
	var envLines []string
	for k, v := range vars {
		envLines = append(envLines, fmt.Sprintf("%s=%s", k, v))
	}
	_ = os.WriteFile(filepath.Join(appDir, ".env"), []byte(strings.Join(envLines, "\n")), 0644)

	// Deploy containers
	containerNames := s.extractContainerNames(req.Name, composeContent)
	if s.dockerMode == "docker" {
		// If docker compose CLI is present, run docker compose up -d
		if _, err := exec.LookPath("docker"); err == nil {
			cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "up", "-d")
			cmd.Dir = appDir
			if out, err := cmd.CombinedOutput(); err != nil {
				return nil, fmt.Errorf("docker compose 启动失败: %s (%w)", string(out), err)
			}
		} else if s.dockerService != nil {
			// Fallback to running through Docker Engine API
			for _, cname := range containerNames {
				_, _ = s.dockerService.RunContainer(ctx, docker.RunContainerRequest{
					Name:   cname,
					Image:  template.ID + ":latest",
					Labels: map[string]string{"io.camopanel.app": req.Name},
				})
			}
		}
	} else if s.dockerService != nil {
		// Demo mode: simulate containers in docker service
		for _, cname := range containerNames {
			_, _ = s.dockerService.RunContainer(ctx, docker.RunContainerRequest{
				Name:   cname,
				Image:  template.ID + ":alpine",
				Labels: map[string]string{"io.camopanel.app": req.Name},
			})
		}
	}

	category := "tools"
	var gateway GatewaySpec
	primaryPort := 0
	if template != nil {
		category = template.Category
		gateway = template.Gateway
		if gateway.PrimaryPortKey != "" && finalPorts[gateway.PrimaryPortKey] > 0 {
			primaryPort = finalPorts[gateway.PrimaryPortKey]
		}
	}
	if primaryPort == 0 && len(finalPorts) > 0 {
		for _, p := range finalPorts {
			primaryPort = p
			break
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	installedApp := &InstalledApp{
		ID:             req.Name,
		AppID:          req.AppID,
		Name:           req.Name,
		Category:       category,
		Status:         "running",
		AppDir:         appDir,
		ComposePath:    composePath,
		Ports:          finalPorts,
		PrimaryPort:    primaryPort,
		Gateway:        gateway,
		Env:            vars,
		ContainerNames: containerNames,
		InstalledAt:    now,
		UpdatedAt:      now,
	}

	s.installed[req.Name] = installedApp
	_ = s.savePersisted()

	return installedApp, nil
}

func (s *Service) extractContainerNames(appName, composeContent string) []string {
	var names []string
	lines := strings.Split(composeContent, "\n")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "container_name:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "container_name:"))
			val = strings.Trim(val, `"' `)
			if val != "" {
				names = append(names, val)
			}
		}
	}
	if len(names) == 0 {
		names = append(names, appName+"-app")
	}
	return names
}

func (s *Service) Action(ctx context.Context, id, action string, deleteData bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.installed[id]
	if !ok {
		return fmt.Errorf("应用不存在: %s", id)
	}

	switch action {
	case "start":
		if s.dockerMode == "docker" {
			if _, err := exec.LookPath("docker"); err == nil {
				cmd := exec.CommandContext(ctx, "docker", "compose", "-f", app.ComposePath, "start")
				cmd.Dir = app.AppDir
				_ = cmd.Run()
			}
		}
		if s.dockerService != nil {
			for _, cname := range app.ContainerNames {
				c, err := s.dockerService.FindContainerByName(ctx, cname)
				if err == nil && c != nil {
					_ = s.dockerService.Action(ctx, c.ID, "start")
				}
			}
		}
		app.Status = "running"
		app.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	case "stop":
		if s.dockerMode == "docker" {
			if _, err := exec.LookPath("docker"); err == nil {
				cmd := exec.CommandContext(ctx, "docker", "compose", "-f", app.ComposePath, "stop")
				cmd.Dir = app.AppDir
				_ = cmd.Run()
			}
		}
		if s.dockerService != nil {
			for _, cname := range app.ContainerNames {
				c, err := s.dockerService.FindContainerByName(ctx, cname)
				if err == nil && c != nil {
					_ = s.dockerService.Action(ctx, c.ID, "stop")
				}
			}
		}
		app.Status = "stopped"
		app.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	case "restart":
		if s.dockerMode == "docker" {
			if _, err := exec.LookPath("docker"); err == nil {
				cmd := exec.CommandContext(ctx, "docker", "compose", "-f", app.ComposePath, "restart")
				cmd.Dir = app.AppDir
				_ = cmd.Run()
			}
		}
		if s.dockerService != nil {
			for _, cname := range app.ContainerNames {
				c, err := s.dockerService.FindContainerByName(ctx, cname)
				if err == nil && c != nil {
					_ = s.dockerService.Action(ctx, c.ID, "restart")
				}
			}
		}
		app.Status = "running"
		app.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	case "uninstall":
		if s.dockerMode == "docker" {
			if _, err := exec.LookPath("docker"); err == nil {
				cmd := exec.CommandContext(ctx, "docker", "compose", "-f", app.ComposePath, "down")
				cmd.Dir = app.AppDir
				_ = cmd.Run()
			}
		}
		if s.dockerService != nil {
			for _, cname := range app.ContainerNames {
				c, err := s.dockerService.FindContainerByName(ctx, cname)
				if err == nil && c != nil {
					_ = s.dockerService.Action(ctx, c.ID, "stop")
					_ = s.dockerService.Action(ctx, c.ID, "remove")
				}
			}
		}
		delete(s.installed, id)
		_ = s.savePersisted()

		if deleteData && app.AppDir != "" {
			_ = os.RemoveAll(app.AppDir)
		}
		return nil

	default:
		return fmt.Errorf("不支持的操作: %s", action)
	}

	_ = s.savePersisted()
	return nil
}

func (s *Service) GetCompose(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	app, ok := s.installed[id]
	if !ok {
		return "", fmt.Errorf("应用不存在: %s", id)
	}
	b, err := os.ReadFile(app.ComposePath)
	if err != nil {
		return "", fmt.Errorf("读取 compose 文件失败: %w", err)
	}
	return string(b), nil
}

func (s *Service) UpdateCompose(ctx context.Context, id, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	app, ok := s.installed[id]
	if !ok {
		return fmt.Errorf("应用不存在: %s", id)
	}
	// Check volume constraints
	if err := ValidateComposeVolumes(app.AppDir, content); err != nil {
		return err
	}
	if err := os.WriteFile(app.ComposePath, []byte(content), 0644); err != nil {
		return err
	}
	app.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	_ = s.savePersisted()
	return nil
}

func (s *Service) GetLogs(ctx context.Context, id string) (string, error) {
	s.mu.RLock()
	app, ok := s.installed[id]
	s.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("应用不存在: %s", id)
	}

	var sb strings.Builder
	if s.dockerService != nil {
		for _, cname := range app.ContainerNames {
			c, err := s.dockerService.FindContainerByName(ctx, cname)
			if err == nil && c != nil {
				logs, err := s.dockerService.Logs(ctx, c.ID)
				if err == nil {
					sb.WriteString(fmt.Sprintf("=== [%s] 日志 ===\n%s\n", cname, logs))
				}
			}
		}
	}
	if sb.Len() == 0 {
		sb.WriteString(fmt.Sprintf("%s [app-store] 应用 %s 运行正常，暂无额外控制台日志\n", time.Now().UTC().Format(time.RFC3339), id))
	}
	return sb.String(), nil
}
