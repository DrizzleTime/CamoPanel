package domain

import (
	"errors"
	"time"
)

const (
	KindTemplate = "template"
	KindCustom   = "custom"

	ActionStart    = "start"
	ActionStop     = "stop"
	ActionRestart  = "restart"
	ActionDelete   = "delete"
	ActionRedeploy = "redeploy"
)

var ErrProjectNotFound = errors.New("project not found")

type Project struct {
	ID              string
	Name            string
	Kind            string
	TemplateID      string
	TemplateVersion string
	Config          map[string]any
	ComposePath     string
	Status          string
	LastError       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
