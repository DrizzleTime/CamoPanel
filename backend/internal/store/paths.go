package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AppDirectory returns the absolute path for an application given its base data dir and app name.
func AppDirectory(baseDataDir, appName string) (string, error) {
	if appName == "" || strings.Contains(appName, "..") || strings.Contains(appName, "/") || strings.Contains(appName, "\\") {
		return "", fmt.Errorf("非法应用名称: %s", appName)
	}
	absBase, err := filepath.Abs(baseDataDir)
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(absBase, "apps", appName)
	return appDir, nil
}

// EnsureAppDirectoryConstraints creates the strictly constrained directory structure for an app:
// - <appDir>/data
// - <appDir>/config
// - <appDir>/logs
func EnsureAppDirectoryConstraints(appDir string) error {
	dirs := []string{
		appDir,
		filepath.Join(appDir, "data"),
		filepath.Join(appDir, "config"),
		filepath.Join(appDir, "logs"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建应用受控目录失败 %s: %w", dir, err)
		}
	}
	return nil
}

// ValidateVolumeConstraint checks whether a host volume binding is strictly constrained within the app directory.
// Allowed:
// - Relative paths like ./data, ./config, ./logs, ./data/db (resolves inside appDir)
// - Named volumes (no slash or starting with volume name)
// - Absolute paths strictly prefixed with appDir
// Disallowed:
// - Paths traversing outside appDir with ../
// - Arbitrary host absolute paths like /, /etc, /var/run, /root
func ValidateVolumeConstraint(appDir, hostMount string) error {
	hostMount = strings.TrimSpace(hostMount)
	if hostMount == "" {
		return nil
	}
	// Named volumes have no slashes or dots prefix
	if !strings.HasPrefix(hostMount, ".") && !strings.HasPrefix(hostMount, "/") && !strings.HasPrefix(hostMount, "~") && !strings.Contains(hostMount, "/") {
		return nil
	}

	cleanMount := filepath.Clean(hostMount)
	if filepath.IsAbs(cleanMount) {
		cleanAppDir := filepath.Clean(appDir)
		rel, err := filepath.Rel(cleanAppDir, cleanMount)
		if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
			return fmt.Errorf("安全约束拦截: 挂载路径 [%s] 超出应用专属安装目录 [%s]", hostMount, appDir)
		}
		return nil
	}

	// Relative path
	target := filepath.Join(appDir, cleanMount)
	targetClean := filepath.Clean(target)
	rel, err := filepath.Rel(filepath.Clean(appDir), targetClean)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return fmt.Errorf("安全约束拦截: 挂载路径 [%s] 试图逃逸应用专属安装目录", hostMount)
	}

	return nil
}

// ValidateComposeVolumes inspects lines of a compose yaml string to ensure all volume mounts obey path constraints.
func ValidateComposeVolumes(appDir, composeContent string) error {
	lines := strings.Split(composeContent, "\n")
	inVolumesBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "volumes:") {
			inVolumesBlock = true
			continue
		}
		// If at top-level key or other block, reset if indentation dropped
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(trimmed, "volumes:") {
			inVolumesBlock = false
		}
		if strings.HasPrefix(trimmed, "- ") {
			item := strings.TrimPrefix(trimmed, "- ")
			item = strings.Trim(item, `"' `)
			parts := strings.Split(item, ":")
			if len(parts) >= 2 {
				hostPart := parts[0]
				// If hostPart looks like a mount path, validate it
				if strings.HasPrefix(hostPart, ".") || strings.HasPrefix(hostPart, "/") {
					if err := ValidateVolumeConstraint(appDir, hostPart); err != nil {
						return err
					}
				}
			}
		}
	}
	_ = inVolumesBlock
	return nil
}
