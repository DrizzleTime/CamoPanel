package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
)

var websiteNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validWebsiteName(name string) bool {
	return websiteNamePattern.MatchString(name)
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDomain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDomains(values []string) []string {
	items := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		normalized := normalizeDomain(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		items = append(items, normalized)
	}
	return items
}

func normalizeWebsiteRootPath(openRestyDataDir, name, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return filepath.Join(openRestyDataDir, "www", name)
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Join(openRestyDataDir, "www", filepath.Clean(trimmed))
}

func ensurePathWithin(root, target string) error {
	relativePath, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil || strings.HasPrefix(relativePath, "..") {
		return fmt.Errorf("站点目录必须位于 %s 下", root)
	}
	return nil
}

func normalizeIndexFilesByType(websiteType string, raw string) []string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) > 0 {
		return fields
	}
	if websiteType == websitesdomain.TypePHP {
		return []string{"index.php", "index.html", "index.htm"}
	}
	return []string{"index.html", "index.htm"}
}

func normalizeRewriteConfig(mode, preset, rules string) (string, string, string, error) {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "", "off":
		return "off", "", "", nil
	case "preset":
		preset = strings.TrimSpace(preset)
		if !slices.Contains([]string{"spa", "front_controller"}, preset) {
			return "", "", "", fmt.Errorf("请选择合法的伪静态预设")
		}
		return "preset", preset, "", nil
	case "custom":
		rules = strings.TrimSpace(rules)
		if rules == "" {
			return "", "", "", fmt.Errorf("请输入自定义伪静态规则")
		}
		return "custom", "", rules, nil
	default:
		return "", "", "", fmt.Errorf("不支持的伪静态模式")
	}
}

func findPHPEnvironmentProject(ctx context.Context, projects ProjectReader, projectID string) (projectsdomain.Project, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return projectsdomain.Project{}, fmt.Errorf("请选择 PHP 环境")
	}
	project, err := projects.FindByID(ctx, projectID)
	if err != nil {
		return projectsdomain.Project{}, err
	}
	if project.TemplateID != "php-fpm" {
		return projectsdomain.Project{}, fmt.Errorf("所选项目不是 PHP 环境")
	}
	return project, nil
}

func projectConfigPort(project projectsdomain.Project) (int, error) {
	port, ok := project.Config["port"]
	if !ok {
		return 0, fmt.Errorf("PHP 环境缺少有效端口配置")
	}
	switch value := port.(type) {
	case int:
		if value > 0 {
			return value, nil
		}
	case float64:
		if int(value) > 0 {
			return int(value), nil
		}
	case string:
		value = strings.TrimSpace(value)
		if value != "" {
			var number int
			if _, err := fmt.Sscanf(value, "%d", &number); err == nil && number > 0 {
				return number, nil
			}
		}
	}

	raw, _ := json.Marshal(project.Config)
	_ = raw
	return 0, fmt.Errorf("PHP 环境缺少有效端口配置")
}
