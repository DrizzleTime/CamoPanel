package store

import (
	"crypto/rand"
	"embed"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"sort"
)

//go:embed apps/*
var embeddedAppsFS embed.FS

// LoadEmbeddedCatalog loads all built-in store apps embedded in the binary.
func LoadEmbeddedCatalog() ([]StoreApp, error) {
	entries, err := embeddedAppsFS.ReadDir("apps")
	if err != nil {
		return nil, fmt.Errorf("读取嵌入应用目录失败: %w", err)
	}

	var apps []StoreApp
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		appID := entry.Name()
		appYamlPath := "apps/" + appID + "/app.yaml"
		composePath := "apps/" + appID + "/compose.yaml"

		yamlData, err := embeddedAppsFS.ReadFile(appYamlPath)
		if err != nil {
			// Skip directories that do not contain app.yaml
			continue
		}

		manifest, err := ParseAppManifest(yamlData)
		if err != nil {
			return nil, fmt.Errorf("解析嵌入应用 [%s] 失败: %w", appID, err)
		}

		var composeContent string
		if composeData, err := embeddedAppsFS.ReadFile(composePath); err == nil {
			composeContent = string(composeData)
		}

		apps = append(apps, manifest.ToStoreApp(composeContent))
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].ID < apps[j].ID
	})

	return apps, nil
}

// LoadExternalCatalog scans a local directory (e.g. data/custom_apps) for user-defined or community apps.
func LoadExternalCatalog(dir string) ([]StoreApp, error) {
	if dir == "" {
		return nil, nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取外部自定义应用目录失败: %w", err)
	}

	var apps []StoreApp
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		appID := entry.Name()
		appYamlPath := filepath.Join(dir, appID, "app.yaml")
		composePath := filepath.Join(dir, appID, "compose.yaml")

		yamlData, err := os.ReadFile(appYamlPath)
		if err != nil {
			continue
		}

		manifest, err := ParseAppManifest(yamlData)
		if err != nil {
			return nil, fmt.Errorf("解析外部应用 [%s] 失败: %w", appID, err)
		}

		var composeContent string
		if composeData, err := os.ReadFile(composePath); err == nil {
			composeContent = string(composeData)
		}

		apps = append(apps, manifest.ToStoreApp(composeContent))
	}

	return apps, nil
}

// LoadAllCatalog loads embedded apps and merges any external custom apps.
func LoadAllCatalog(customAppsDir string) ([]StoreApp, error) {
	embedded, err := LoadEmbeddedCatalog()
	if err != nil {
		return nil, err
	}

	appMap := make(map[string]StoreApp)
	for _, app := range embedded {
		appMap[app.ID] = app
	}

	if customAppsDir != "" {
		external, err := LoadExternalCatalog(customAppsDir)
		if err == nil {
			for _, app := range external {
				appMap[app.ID] = app
			}
		}
	}

	result := make([]StoreApp, 0, len(appMap))
	for _, app := range appMap {
		result = append(result, app)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

const passwordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_#@!"

// GenerateRandomPassword generates a cryptographically secure random password of specified length.
func GenerateRandomPassword(length int) string {
	if length <= 0 {
		length = 16
	}
	bytes := make([]byte, length)
	charsetLen := big.NewInt(int64(len(passwordCharset)))
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			bytes[i] = passwordCharset[i%len(passwordCharset)]
		} else {
			bytes[i] = passwordCharset[idx.Int64()]
		}
	}
	return string(bytes)
}

// EnsureFSReadCloser suppresses unused warning
var _ fs.ReadDirFile
