package model

import "time"

const (
	RoleSuperAdmin = "super_admin"

	ActionStart    = "start"
	ActionStop     = "stop"
	ActionRestart  = "restart"
	ActionDelete   = "delete"
	ActionRedeploy = "redeploy"

	WebsiteTypeStatic = "static"
	WebsiteTypePHP    = "php"
	WebsiteTypeProxy  = "proxy"
)

type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"not null" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Project struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"uniqueIndex;not null" json:"name"`
	TemplateID      string    `gorm:"not null" json:"template_id"`
	TemplateVersion string    `gorm:"not null" json:"template_version"`
	ConfigJSON      string    `gorm:"type:text;not null" json:"config_json"`
	ComposePath     string    `gorm:"not null" json:"compose_path"`
	Status          string    `gorm:"not null" json:"status"`
	LastError       string    `gorm:"type:text" json:"last_error"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Website struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Domain        string    `json:"domain"`
	DomainsJSON   string    `json:"domains_json"`
	SiteMode      string    `json:"site_mode"`
	RootPath      string    `json:"root_path"`
	IndexFiles    string    `json:"index_files"`
	ProxyPass     string    `json:"proxy_pass"`
	PHPProjectID  string    `json:"php_project_id"`
	PHPPort       int       `json:"php_port"`
	RewriteMode   string    `json:"rewrite_mode"`
	RewritePreset string    `json:"rewrite_preset"`
	RewriteRules  string    `json:"rewrite_rules"`
	ConfigPath    string    `json:"config_path"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Certificate struct {
	ID             string    `gorm:"primaryKey" json:"id"`
	Domain         string    `gorm:"uniqueIndex;not null" json:"domain"`
	Email          string    `gorm:"not null" json:"email"`
	Provider       string    `gorm:"not null" json:"provider"`
	Status         string    `gorm:"not null" json:"status"`
	FullchainPath  string    `gorm:"not null" json:"fullchain_path"`
	PrivateKeyPath string    `gorm:"not null" json:"private_key_path"`
	LastError      string    `gorm:"type:text" json:"last_error"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type AuditEvent struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	ActorID      string    `json:"actor_id"`
	Action       string    `gorm:"not null" json:"action"`
	TargetType   string    `gorm:"not null" json:"target_type"`
	TargetID     string    `gorm:"not null" json:"target_id"`
	MetadataJSON string    `gorm:"type:text" json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type AIProvider struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	Type      string    `gorm:"not null" json:"type"`
	BaseURL   string    `gorm:"not null" json:"base_url"`
	APIKey    string    `gorm:"type:text;not null" json:"-"`
	Enabled   bool      `gorm:"not null" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AIModel struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	ProviderID string    `gorm:"index;not null" json:"provider_id"`
	Name       string    `gorm:"not null" json:"name"`
	Enabled    bool      `gorm:"not null" json:"enabled"`
	IsDefault  bool      `gorm:"not null" json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
