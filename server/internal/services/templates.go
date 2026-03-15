package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"gopkg.in/yaml.v3"
)

const (
	ParamString  = "string"
	ParamNumber  = "number"
	ParamBoolean = "boolean"
	ParamSecret  = "secret"
)

type TemplateParam struct {
	Name        string `json:"name" yaml:"name"`
	Label       string `json:"label" yaml:"label"`
	Description string `json:"description" yaml:"description"`
	Type        string `json:"type" yaml:"type"`
	Required    bool   `json:"required" yaml:"required"`
	Default     any    `json:"default,omitempty" yaml:"default"`
	Placeholder string `json:"placeholder,omitempty" yaml:"placeholder"`
}

type TemplateSpec struct {
	ID          string          `json:"id" yaml:"id"`
	Name        string          `json:"name" yaml:"name"`
	Version     string          `json:"version" yaml:"version"`
	Description string          `json:"description" yaml:"description"`
	Params      []TemplateParam `json:"params" yaml:"params"`
	HealthHints []string        `json:"health_hints" yaml:"health_hints"`
}

type TemplateRuntime struct {
	ProjectName          string
	OpenRestyContainer   string
	OpenRestyHostConfDir string
	OpenRestyHostSiteDir string
}

type LoadedTemplate struct {
	Spec            TemplateSpec
	Path            string
	ComposeTemplate string
}

type TemplateCatalog struct {
	root      string
	templates map[string]*LoadedTemplate
}

func NewTemplateCatalog(root string) (*TemplateCatalog, error) {
	catalog := &TemplateCatalog{
		root:      root,
		templates: map[string]*LoadedTemplate{},
	}
	if err := catalog.Reload(); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (c *TemplateCatalog) Reload() error {
	entries, err := os.ReadDir(c.root)
	if err != nil {
		return fmt.Errorf("read templates dir: %w", err)
	}

	loaded := map[string]*LoadedTemplate{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		templatePath := filepath.Join(c.root, entry.Name())
		specBytes, err := os.ReadFile(filepath.Join(templatePath, "template.yaml"))
		if err != nil {
			continue
		}

		var spec TemplateSpec
		if err := yaml.Unmarshal(specBytes, &spec); err != nil {
			continue
		}
		if spec.ID == "" {
			spec.ID = entry.Name()
		}
		composeBytes, err := os.ReadFile(filepath.Join(templatePath, "compose.yaml.tmpl"))
		if err != nil {
			continue
		}

		loaded[spec.ID] = &LoadedTemplate{
			Spec:            spec,
			Path:            templatePath,
			ComposeTemplate: string(composeBytes),
		}
	}

	c.templates = loaded
	return nil
}

func (c *TemplateCatalog) List() []TemplateSpec {
	templates := make([]TemplateSpec, 0, len(c.templates))
	for _, item := range c.templates {
		templates = append(templates, item.Spec)
	}
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})
	return templates
}

func (c *TemplateCatalog) Get(id string) (*LoadedTemplate, error) {
	item, ok := c.templates[id]
	if !ok {
		return nil, fmt.Errorf("template %s not found", id)
	}
	return item, nil
}

func (t *LoadedTemplate) ValidateAndNormalize(input map[string]any) (map[string]any, error) {
	normalized := map[string]any{}

	for _, param := range t.Spec.Params {
		value, exists := input[param.Name]
		if !exists || value == nil || value == "" {
			if param.Default != nil {
				normalized[param.Name] = param.Default
				continue
			}
			if param.Required {
				return nil, fmt.Errorf("param %s is required", param.Name)
			}
			continue
		}

		converted, err := convertParamValue(param.Type, value)
		if err != nil {
			return nil, fmt.Errorf("param %s: %w", param.Name, err)
		}
		normalized[param.Name] = converted
	}

	return normalized, nil
}

func (t *LoadedTemplate) Render(input map[string]any, runtime TemplateRuntime) (string, error) {
	tpl, err := template.New("compose").Option("missingkey=error").Parse(t.ComposeTemplate)
	if err != nil {
		return "", fmt.Errorf("parse compose template: %w", err)
	}

	payload := map[string]any{
		"Values":  input,
		"Runtime": runtime,
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, payload); err != nil {
		return "", fmt.Errorf("render compose template: %w", err)
	}

	return buf.String(), nil
}

func (t *LoadedTemplate) ConfigJSON(input map[string]any) (string, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func convertParamValue(paramType string, raw any) (any, error) {
	switch paramType {
	case ParamString, ParamSecret:
		switch value := raw.(type) {
		case string:
			if value == "" {
				return nil, fmt.Errorf("must not be empty")
			}
			return value, nil
		default:
			return fmt.Sprint(raw), nil
		}
	case ParamNumber:
		switch value := raw.(type) {
		case float64:
			if math.Mod(value, 1) == 0 {
				return int(value), nil
			}
			return value, nil
		case int, int32, int64, float32:
			return value, nil
		case string:
			var number float64
			if _, err := fmt.Sscanf(value, "%f", &number); err != nil {
				return nil, fmt.Errorf("must be a number")
			}
			if math.Mod(number, 1) == 0 {
				return int(number), nil
			}
			return number, nil
		default:
			return nil, fmt.Errorf("must be a number")
		}
	case ParamBoolean:
		switch value := raw.(type) {
		case bool:
			return value, nil
		case string:
			switch value {
			case "true":
				return true, nil
			case "false":
				return false, nil
			default:
				return nil, fmt.Errorf("must be true or false")
			}
		default:
			return nil, fmt.Errorf("must be boolean")
		}
	default:
		return nil, fmt.Errorf("unsupported param type %s", paramType)
	}
}
