package store

import (
	"fmt"
	"regexp"

	"github.com/goccy/go-yaml"
)

var validManifestIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{1,62}$`)

type GatewaySpec struct {
	IsWebApp       bool   `json:"is_web_app" yaml:"is_web_app"`
	PrimaryPortKey string `json:"primary_port_key" yaml:"primary_port_key"`
	DefaultSubpath string `json:"default_subpath,omitempty" yaml:"default_subpath,omitempty"`
}

type PortSpec struct {
	Key           string `json:"key" yaml:"key"`
	Label         string `json:"label" yaml:"label"`
	ContainerPort int    `json:"container_port" yaml:"container_port"`
	Protocol      string `json:"protocol" yaml:"protocol"` // "tcp" or "udp", defaults to "tcp"
	Default       int    `json:"default" yaml:"default"`
	Description   string `json:"description" yaml:"description"`
	Required      bool   `json:"required" yaml:"required"`
	Proxyable     bool   `json:"proxyable" yaml:"proxyable"`
}

type ShowIfCondition struct {
	Field    string      `json:"field" yaml:"field"`
	Operator string      `json:"operator" yaml:"operator"` // "equals", "not_equals"
	Value    interface{} `json:"value" yaml:"value"`
}

type ParamOption struct {
	Label string `json:"label" yaml:"label"`
	Value string `json:"value" yaml:"value"`
}

type ParamSpec struct {
	Key          string           `json:"key" yaml:"key"`
	Label        string           `json:"label" yaml:"label"`
	Type         string           `json:"type" yaml:"type"` // "string", "password", "number", "boolean", "select"
	Default      interface{}      `json:"default" yaml:"default"`
	Description  string           `json:"description" yaml:"description"`
	Required     bool             `json:"required" yaml:"required"`
	IsSecret     bool             `json:"is_secret" yaml:"is_secret"`
	AutoGenerate bool             `json:"auto_generate" yaml:"auto_generate"`
	Options      []ParamOption    `json:"options,omitempty" yaml:"options,omitempty"`
	ShowIf       *ShowIfCondition `json:"show_if,omitempty" yaml:"show_if,omitempty"`
}

type AppManifest struct {
	SchemaVersion string      `json:"schema_version" yaml:"schema_version"`
	ID            string      `json:"id" yaml:"id"`
	Name          string      `json:"name" yaml:"name"`
	Category      string      `json:"category" yaml:"category"`
	Version       string      `json:"version" yaml:"version"`
	Icon          string      `json:"icon" yaml:"icon"`
	Website       string      `json:"website" yaml:"website"`
	Description   string      `json:"description" yaml:"description"`
	Gateway       GatewaySpec `json:"gateway" yaml:"gateway"`
	DataDirs      []string    `json:"data_dirs" yaml:"data_dirs"`
	Ports         []PortSpec  `json:"ports" yaml:"ports"`
	Params        []ParamSpec `json:"params" yaml:"params"`
}

// ParseAppManifest parses and validates an app.yaml byte slice.
func ParseAppManifest(data []byte) (*AppManifest, error) {
	var manifest AppManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("解析 app.yaml 失败: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Validate checks semantic correctness of the manifest.
func (m *AppManifest) Validate() error {
	if m.ID == "" || !validManifestIDPattern.MatchString(m.ID) {
		return fmt.Errorf("非法应用 ID [%s]: 必须由 2-63 位小写字母、数字、点、下划线或横线组成", m.ID)
	}
	if m.Name == "" {
		return fmt.Errorf("应用名称 (name) 不能为空")
	}
	if m.Category == "" {
		return fmt.Errorf("应用分类 (category) 不能为空")
	}

	portKeys := make(map[string]bool)
	for i, p := range m.Ports {
		if p.Key == "" {
			return fmt.Errorf("端口配置第 %d 项缺少 key", i+1)
		}
		if portKeys[p.Key] {
			return fmt.Errorf("端口配置 key 重复: %s", p.Key)
		}
		portKeys[p.Key] = true
		if p.ContainerPort <= 0 || p.ContainerPort > 65535 {
			return fmt.Errorf("端口 %s 容器目标端口无效: %d", p.Key, p.ContainerPort)
		}
		if p.Default < 0 || p.Default > 65535 {
			return fmt.Errorf("端口 %s 默认宿主机端口无效: %d", p.Key, p.Default)
		}
		if m.Ports[i].Protocol == "" {
			m.Ports[i].Protocol = "tcp"
		}
	}

	paramKeys := make(map[string]bool)
	for i, param := range m.Params {
		if param.Key == "" {
			return fmt.Errorf("参数配置第 %d 项缺少 key", i+1)
		}
		if paramKeys[param.Key] {
			return fmt.Errorf("参数配置 key 重复: %s", param.Key)
		}
		paramKeys[param.Key] = true

		if param.Type == "" {
			m.Params[i].Type = "string"
		}
	}

	if m.Gateway.IsWebApp && m.Gateway.PrimaryPortKey != "" {
		if !portKeys[m.Gateway.PrimaryPortKey] {
			return fmt.Errorf("网关指定的主端口 key [%s] 未在 ports 中声明", m.Gateway.PrimaryPortKey)
		}
	}

	return nil
}

// ToStoreApp converts a validated manifest and its compose content to StoreApp.
func (m *AppManifest) ToStoreApp(composeContent string) StoreApp {
	defaultPorts := make(map[string]int)
	for _, p := range m.Ports {
		defaultPorts[p.Key] = p.Default
	}

	envParams := make([]EnvParam, 0, len(m.Params))
	for _, p := range m.Params {
		defaultStr := ""
		if p.Default != nil {
			defaultStr = fmt.Sprintf("%v", p.Default)
		}
		envParams = append(envParams, EnvParam{
			Key:         p.Key,
			Label:       p.Label,
			Default:     defaultStr,
			Description: p.Description,
			Required:    p.Required,
			IsSecret:    p.IsSecret,
		})
	}

	gateway := m.Gateway
	if gateway.DefaultSubpath == "" {
		gateway.DefaultSubpath = "/"
	}

	return StoreApp{
		ID:              m.ID,
		Name:            m.Name,
		Category:        m.Category,
		Description:     m.Description,
		Version:         m.Version,
		Icon:            m.Icon,
		Website:         m.Website,
		DefaultPorts:    defaultPorts,
		EnvParams:       envParams,
		Ports:           m.Ports,
		Params:          m.Params,
		Gateway:         gateway,
		ComposeTemplate: composeContent,
		DataDirs:        m.DataDirs,
	}
}
