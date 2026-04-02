package domain

import (
	"errors"
	"time"
)

const (
	TypeStatic = "static"
	TypePHP    = "php"
	TypeProxy  = "proxy"
)

var ErrWebsiteNotFound = errors.New("website not found")

type Website struct {
	ID            string
	Name          string
	Type          string
	Domain        string
	Domains       []string
	RootPath      string
	IndexFiles    []string
	ProxyPass     string
	PHPProjectID  string
	PHPPort       int
	RewriteMode   string
	RewritePreset string
	RewriteRules  string
	ConfigPath    string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
