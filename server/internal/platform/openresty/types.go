package openresty

import "time"

type Status struct {
	CertificateReady bool   `json:"certificate_ready"`
	Exists           bool   `json:"exists"`
	Ready            bool   `json:"ready"`
	ContainerName    string `json:"container_name"`
	ContainerStatus  string `json:"container_status"`
	HostConfigDir    string `json:"host_config_dir"`
	HostSiteDir      string `json:"host_site_dir"`
	Message          string `json:"message"`
}

type WebsiteSpec struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	Domain        string   `json:"domain"`
	Domains       []string `json:"domains,omitempty"`
	RootPath      string   `json:"root_path,omitempty"`
	IndexFiles    []string `json:"index_files,omitempty"`
	ProxyPass     string   `json:"proxy_pass,omitempty"`
	PHPPort       int      `json:"php_port,omitempty"`
	RewriteMode   string   `json:"rewrite_mode,omitempty"`
	RewritePreset string   `json:"rewrite_preset,omitempty"`
	RewriteRules  string   `json:"rewrite_rules,omitempty"`
}

type WebsiteMaterialized struct {
	RootPath   string
	ConfigPath string
}

type CertificateSpec struct {
	Domain             string `json:"domain"`
	Email              string `json:"email"`
	UseExistingWebsite bool   `json:"use_existing_website"`
}

type CertificateMaterialized struct {
	Provider       string
	FullchainPath  string
	PrivateKeyPath string
	ExpiresAt      time.Time
}
