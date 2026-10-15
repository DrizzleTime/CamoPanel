package store

type EnvParam struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Default     string `json:"default"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	IsSecret    bool   `json:"is_secret"`
}

type StoreApp struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Description     string            `json:"description"`
	Version         string            `json:"version"`
	Icon            string            `json:"icon"`
	Website         string            `json:"website"`
	DefaultPorts    map[string]int    `json:"default_ports"`
	EnvParams       []EnvParam        `json:"env_params"`
	Ports           []PortSpec        `json:"ports"`
	Params          []ParamSpec       `json:"params"`
	Gateway         GatewaySpec       `json:"gateway"`
	ComposeTemplate string            `json:"compose_template"`
	DataDirs        []string          `json:"data_dirs"`
	Installed       bool              `json:"installed"`
}

type InstalledApp struct {
	ID             string            `json:"id"`
	AppID          string            `json:"app_id"`
	Name           string            `json:"name"`
	Category       string            `json:"category"`
	Status         string            `json:"status"` // "running", "stopped", "error"
	AppDir         string            `json:"app_dir"`
	ComposePath    string            `json:"compose_path"`
	Ports          map[string]int    `json:"ports"`
	PrimaryPort    int               `json:"primary_port"`
	Gateway        GatewaySpec       `json:"gateway"`
	Env            map[string]string `json:"env"`
	ContainerNames []string          `json:"container_names"`
	InstalledAt    string            `json:"installed_at"`
	UpdatedAt      string            `json:"updated_at"`
}

type InstallRequest struct {
	AppID         string            `json:"app_id" binding:"required"`
	Name          string            `json:"name" binding:"required,min=2,max=63"`
	Ports         map[string]int    `json:"ports"`
	Env           map[string]string `json:"env"`
	CustomCompose string            `json:"custom_compose"`
}

type ActionRequest struct {
	Action     string `json:"action" binding:"required"` // "start", "stop", "restart", "uninstall"
	DeleteData bool   `json:"delete_data"`
}
