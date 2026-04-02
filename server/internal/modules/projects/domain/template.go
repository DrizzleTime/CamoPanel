package domain

type TemplateParam struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

type Template struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Params      []TemplateParam `json:"params"`
	HealthHints []string        `json:"health_hints"`
}
