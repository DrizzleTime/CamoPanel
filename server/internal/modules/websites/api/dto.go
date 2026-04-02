package api

import (
	"encoding/json"
	"strings"
	"time"

	websitesdomain "camopanel/server/internal/modules/websites/domain"
)

func serializeWebsite(item websitesdomain.Website) map[string]any {
	return map[string]any{
		"id":             item.ID,
		"name":           item.Name,
		"type":           item.Type,
		"domain":         item.Domain,
		"domains_json":   mustJSON(item.Domains),
		"site_mode":      item.Type,
		"root_path":      item.RootPath,
		"index_files":    strings.Join(item.IndexFiles, " "),
		"proxy_pass":     item.ProxyPass,
		"php_project_id": item.PHPProjectID,
		"php_port":       item.PHPPort,
		"rewrite_mode":   item.RewriteMode,
		"rewrite_preset": item.RewritePreset,
		"rewrite_rules":  item.RewriteRules,
		"config_path":    item.ConfigPath,
		"status":         item.Status,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func serializeWebsites(items []websitesdomain.Website) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, serializeWebsite(item))
	}
	return result
}

func serializeCertificate(item websitesdomain.Certificate) map[string]any {
	return map[string]any{
		"id":               item.ID,
		"domain":           item.Domain,
		"email":            item.Email,
		"provider":         item.Provider,
		"status":           item.Status,
		"fullchain_path":   item.FullchainPath,
		"private_key_path": item.PrivateKeyPath,
		"last_error":       item.LastError,
		"expires_at":       item.ExpiresAt.UTC().Format(time.RFC3339),
		"created_at":       item.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":       item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func serializeCertificates(items []websitesdomain.Certificate) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, serializeCertificate(item))
	}
	return result
}

func mustJSON(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
