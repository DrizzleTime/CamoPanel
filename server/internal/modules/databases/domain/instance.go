package domain

import (
	"errors"
	"time"

	platformdocker "camopanel/server/internal/platform/docker"
)

const (
	EngineMySQL    = "mysql"
	EnginePostgres = "postgres"
	EngineRedis    = "redis"
)

var (
	ErrInstanceNotFound = errors.New("database instance not found")
	ManagedEngines      = []string{EngineMySQL, EnginePostgres, EngineRedis}
)

type ConnectionInfo struct {
	Host            string `json:"host"`
	Port            int    `json:"port"`
	AdminUsername   string `json:"admin_username,omitempty"`
	AppUsername     string `json:"app_username,omitempty"`
	DefaultDatabase string `json:"default_database,omitempty"`
	PasswordManaged bool   `json:"password_managed"`
}

type Instance struct {
	ID              string
	Name            string
	Engine          string
	TemplateVersion string
	Config          map[string]any
	ComposePath     string
	Status          string
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type InstanceView struct {
	ID         string                        `json:"id"`
	Name       string                        `json:"name"`
	Engine     string                        `json:"engine"`
	Status     string                        `json:"status"`
	LastError  string                        `json:"last_error"`
	Runtime    platformdocker.ProjectRuntime `json:"runtime"`
	Connection ConnectionInfo                `json:"connection"`
	CreatedAt  string                        `json:"created_at"`
	UpdatedAt  string                        `json:"updated_at"`
}

type DatabaseNameItem struct {
	Name string `json:"name"`
}

type AccountItem struct {
	Name      string `json:"name"`
	Host      string `json:"host,omitempty"`
	Superuser bool   `json:"superuser,omitempty"`
}

type RedisKeyspaceItem struct {
	Name string `json:"name"`
	Keys int    `json:"keys"`
}

type RedisConfigItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Overview struct {
	Instance       InstanceView        `json:"instance"`
	Notice         string              `json:"notice,omitempty"`
	Databases      []DatabaseNameItem  `json:"databases,omitempty"`
	Accounts       []AccountItem       `json:"accounts,omitempty"`
	RedisKeyspaces []RedisKeyspaceItem `json:"redis_keyspaces,omitempty"`
	RedisConfig    []RedisConfigItem   `json:"redis_config,omitempty"`
	Summary        map[string]string   `json:"summary,omitempty"`
}
