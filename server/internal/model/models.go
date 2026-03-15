package model

import "time"

const (
	RoleSuperAdmin = "super_admin"

	ApprovalStatusPending   = "pending"
	ApprovalStatusExecuting = "executing"
	ApprovalStatusApproved  = "approved"
	ApprovalStatusRejected  = "rejected"
	ApprovalStatusFailed    = "failed"

	ApprovalActionDeploy        = "deploy"
	ApprovalActionStart         = "start"
	ApprovalActionStop          = "stop"
	ApprovalActionRestart       = "restart"
	ApprovalActionDelete        = "delete"
	ApprovalActionRedeploy      = "redeploy"
	ApprovalActionCreateWebsite = "create_website"

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

type ApprovalRequest struct {
	ID           string     `gorm:"primaryKey" json:"id"`
	Source       string     `gorm:"not null" json:"source"`
	Action       string     `gorm:"not null" json:"action"`
	TargetType   string     `gorm:"not null" json:"target_type"`
	TargetID     string     `gorm:"not null" json:"target_id"`
	PayloadJSON  string     `gorm:"type:text;not null" json:"payload_json"`
	Summary      string     `gorm:"type:text;not null" json:"summary"`
	Status       string     `gorm:"not null" json:"status"`
	CreatedBy    string     `gorm:"not null" json:"created_by"`
	ApprovedBy   string     `json:"approved_by"`
	ErrorMessage string     `gorm:"type:text" json:"error_message"`
	ExecutedAt   *time.Time `json:"executed_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
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
