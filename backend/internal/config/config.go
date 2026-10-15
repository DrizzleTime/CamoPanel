package config

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"camopanel/backend/internal/auth"
)

type Config struct {
	Address       string
	DockerHost    string
	Token         string
	AdminUsername string
	AdminPassword string
	WebDir        string
	DataDir       string
	FilesRoot     string
	NginxPort     int
	NginxSSLPort  int
	ACMEEmail     string
	ACMEDirectory string
	TLSCert       string
	TLSKey        string
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intValue(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 && n <= 65535 {
			return n
		}
	}
	return fallback
}

func Load() (Config, error) {
	c := Config{
		Address:       value("CAMO_ADDR", "127.0.0.1:8080"),
		DockerHost:    value("DOCKER_HOST", "unix:///var/run/docker.sock"),
		Token:         os.Getenv("CAMO_API_TOKEN"),
		AdminUsername: strings.TrimSpace(value("CAMO_ADMIN_USERNAME", "admin")),
		AdminPassword: os.Getenv("CAMO_ADMIN_PASSWORD"),
		WebDir:        os.Getenv("CAMO_WEB_DIR"),
		DataDir:       value("CAMO_DATA_DIR", "./data"),
		FilesRoot:     value("CAMO_FILES_ROOT", "/"),
		NginxPort:     intValue("CAMO_NGINX_PORT", 80),
		NginxSSLPort:  intValue("CAMO_NGINX_SSL_PORT", 443),
		ACMEEmail:     value("CAMO_ACME_EMAIL", "admin@example.com"),
		ACMEDirectory: os.Getenv("CAMO_ACME_DIRECTORY"),
		TLSCert:       os.Getenv("CAMO_TLS_CERT"),
		TLSKey:        os.Getenv("CAMO_TLS_KEY"),
	}
	if c.AdminPassword != "" {
		if c.AdminUsername == "" || len(c.AdminUsername) > 64 {
			return c, fmt.Errorf("CAMO_ADMIN_USERNAME must contain 1–64 bytes")
		}
		if len(c.AdminPassword) < 8 || len(c.AdminPassword) > 72 {
			return c, fmt.Errorf("CAMO_ADMIN_PASSWORD must contain 8–72 bytes")
		}
	}
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return c, fmt.Errorf("invalid CAMO_ADDR: %w", err)
	}
	if c.AdminPassword == "" && c.Token == "" && !auth.PasswordConfigured(c.DataDir) {
		return c, fmt.Errorf("set CAMO_ADMIN_USERNAME and CAMO_ADMIN_PASSWORD in .env before starting the panel")
	}
	if c.WebDir == "" {
		for _, path := range []string{"frontend/dist", "../frontend/dist"} {
			if _, err := os.Stat(path + "/index.html"); err == nil {
				c.WebDir = path
				break
			}
		}
	}
	return c, nil
}

func (c Config) PanelPort() int {
	_, port, err := net.SplitHostPort(c.Address)
	if err != nil {
		return 8080
	}
	n, err := strconv.Atoi(port)
	if err != nil || n <= 0 || n > 65535 {
		return 8080
	}
	return n
}

// LoadEnv supports KEY=value, comments and quoted values. Process env takes priority.
func LoadEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("invalid .env line")
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if _, set := os.LookupEnv(key); !set {
			if err := os.Setenv(key, val); err != nil {
				return err
			}
		}
	}
	return s.Err()
}
