package repo

import "time"

type WebsiteRecord struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Domain        string    `json:"domain"`
	Domains       []string  `json:"domains,omitempty"`
	RootPath      string    `json:"root_path"`
	IndexFiles    []string  `json:"index_files,omitempty"`
	ProxyPass     string    `json:"proxy_pass,omitempty"`
	PHPProjectID  string    `json:"php_project_id,omitempty"`
	PHPPort       int       `json:"php_port,omitempty"`
	RewriteMode   string    `json:"rewrite_mode,omitempty"`
	RewritePreset string    `json:"rewrite_preset,omitempty"`
	RewriteRules  string    `json:"rewrite_rules,omitempty"`
	ConfigPath    string    `json:"config_path"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
