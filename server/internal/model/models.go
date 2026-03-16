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
	ID         string    `gorm:"primaryKey" json:"id"`
	Name       string    `gorm:"uniqueIndex;not null" json:"name"`
	Type       string    `gorm:"not null" json:"type"`
	Domain     string    `gorm:"uniqueIndex;not null" json:"domain"`
	RootPath   string    `gorm:"not null" json:"root_path"`
	ProxyPass  string    `gorm:"not null" json:"proxy_pass"`
	ConfigPath string    `gorm:"not null" json:"config_path"`
	Status     string    `gorm:"not null" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
