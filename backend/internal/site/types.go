package site

type Redirect struct {
	From string `json:"from"`
	To   string `json:"to"`
	Code int    `json:"code"`
}

type Location struct {
	Path  string `json:"path"`
	Proxy string `json:"proxy"`
}

type Site struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	PrimaryDomain  string     `json:"primary_domain"`
	Domains        []string   `json:"domains"`
	Type           string     `json:"type"` // "static" | "proxy" | "runtime"
	Runtime        string     `json:"runtime,omitempty"`
	RootPath       string     `json:"root_path"`
	ProxyTarget    string     `json:"proxy_target"`
	SSLEnabled     bool       `json:"ssl_enabled"`
	SSLProvider    string     `json:"ssl_provider"` // "letsencrypt" | "self-signed" | "custom"
	CertID         string     `json:"cert_id,omitempty"`
	CertExpiry     string     `json:"cert_expiry"`
	DaysRemaining  int        `json:"days_remaining"`
	ForceHTTPS     bool       `json:"force_https"`
	Status         string     `json:"status"` // "running" | "stopped"
	CustomConfig   string     `json:"custom_config"`
	Redirects      []Redirect `json:"redirects,omitempty"`
	Locations      []Location `json:"locations,omitempty"`
	BasicAuthUser  string     `json:"basic_auth_user,omitempty"`
	BasicAuthFile  string     `json:"basic_auth_file,omitempty"`
	Hotlink        bool       `json:"hotlink,omitempty"`
	RateLimit      int        `json:"rate_limit,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

type CreateSiteRequest struct {
	Name              string     `json:"name" binding:"required,min=2,max=63"`
	PrimaryDomain     string     `json:"primary_domain" binding:"required"`
	Domains           []string   `json:"domains"`
	Type              string     `json:"type" binding:"required"` // "static" | "proxy" | "runtime"
	Runtime           string     `json:"runtime"`
	ProxyTarget       string     `json:"proxy_target"`
	ForceHTTPS        bool       `json:"force_https"`
	CustomConfig      string     `json:"custom_config"`
	Redirects         []Redirect `json:"redirects"`
	Locations         []Location `json:"locations"`
	BasicAuthUser     string     `json:"basic_auth_user"`
	BasicAuthPassword string     `json:"basic_auth_password"`
	Hotlink           bool       `json:"hotlink"`
	RateLimit         int        `json:"rate_limit"`
}

type UpdateSiteRequest struct {
	Name              string     `json:"name" binding:"required,min=2,max=63"`
	PrimaryDomain     string     `json:"primary_domain" binding:"required"`
	Domains           []string   `json:"domains"`
	Type              string     `json:"type" binding:"required"`
	Runtime           string     `json:"runtime"`
	ProxyTarget       string     `json:"proxy_target"`
	ForceHTTPS        bool       `json:"force_https"`
	CustomConfig      string     `json:"custom_config"`
	Redirects         []Redirect `json:"redirects"`
	Locations         []Location `json:"locations"`
	BasicAuthUser     string     `json:"basic_auth_user"`
	BasicAuthPassword string     `json:"basic_auth_password"`
	Hotlink           bool       `json:"hotlink"`
	RateLimit         int        `json:"rate_limit"`
}

type SSLApplyRequest struct {
	Provider   string `json:"provider" binding:"required"` // "letsencrypt" | "self-signed" | "custom"
	Email      string `json:"email"`
	CertPEM    string `json:"cert_pem"`
	KeyPEM     string `json:"key_pem"`
	ForceHTTPS bool   `json:"force_https"`
}

type SSLInfo struct {
	Enabled       bool     `json:"enabled"`
	Provider      string   `json:"provider"`
	Domains       []string `json:"domains"`
	Expiry        string   `json:"expiry"`
	DaysRemaining int      `json:"days_remaining"`
	Issuer        string   `json:"issuer"`
}

type FileContent struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}
