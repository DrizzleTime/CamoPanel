package certs

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Cert struct {
	ID        string    `json:"id"`
	Domains   []string  `json:"domains"`
	Provider  string    `json:"provider"`
	Expiry    string    `json:"expiry"`
	Issuer    string    `json:"issuer"`
	Sites     []string  `json:"sites"`
	Attempts  []Attempt `json:"attempts"`
	CreatedAt string    `json:"created_at"`
}

type Attempt struct {
	At     string `json:"at"`
	Method string `json:"method"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type DNSConfig struct {
	Provider string `json:"provider"`
	APIBase  string `json:"api_base,omitempty"`
	Token    string `json:"token,omitempty"`
	Zone     string `json:"zone,omitempty"`
}

// HTTP01Func asks an ACME directory to renew one certificate with HTTP-01.
// It returns the new expiry on success.
type HTTP01Func func(ctx context.Context, id string, domains []string) (string, error)

type Store struct {
	dir    string
	mu     sync.Mutex
	http01 HTTP01Func
}

func Open(dataDir string) (*Store, error) {
	dir := filepath.Join(dataDir, "ssl", "certs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) List() ([]Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := []Cert{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cert, err := s.read(entry.Name())
		if err != nil {
			continue
		}
		out = append(out, publicCert(cert))
	}
	return out, nil
}

func (s *Store) Get(id string) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, err := s.read(id)
	if err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

func (s *Store) IssueSelfSigned(domains []string) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := newID()
	if err != nil {
		return Cert{}, err
	}
	expiry := time.Now().UTC().Add(365 * 24 * time.Hour)
	if err := writeSelfSigned(s.path(id), domains, expiry); err != nil {
		return Cert{}, err
	}
	cert := Cert{
		ID: id, Domains: domains, Provider: "self-signed", Issuer: "CamoPanel Self-Signed",
		Expiry: expiry.Format(time.RFC3339), CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Sites: []string{}, Attempts: []Attempt{},
	}
	if err := s.write(cert); err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

func (s *Store) Upload(certPEM, keyPEM string, domains []string) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return Cert{}, fmt.Errorf("无效的证书 PEM")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Cert{}, err
	}
	if !strings.Contains(keyPEM, "PRIVATE KEY") {
		return Cert{}, fmt.Errorf("无效的私钥 PEM")
	}
	id, err := newID()
	if err != nil {
		return Cert{}, err
	}
	if err := os.MkdirAll(s.path(id), 0o755); err != nil {
		return Cert{}, err
	}
	if err := os.WriteFile(filepath.Join(s.path(id), "fullchain.pem"), []byte(certPEM), 0o644); err != nil {
		return Cert{}, err
	}
	if err := os.WriteFile(filepath.Join(s.path(id), "privkey.pem"), []byte(keyPEM), 0o600); err != nil {
		return Cert{}, err
	}
	if len(domains) == 0 {
		domains = parsed.DNSNames
	}
	cert := Cert{
		ID: id, Domains: domains, Provider: "custom", Issuer: parsed.Issuer.CommonName,
		Expiry: parsed.NotAfter.UTC().Format(time.RFC3339), CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Sites: []string{}, Attempts: []Attempt{},
	}
	if err := s.write(cert); err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

// PutTestCert writes one certificate record and a single key file. Tests use it for expiry and binding.
func (s *Store) PutTestCert(cert Cert, keyPEM string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.path(cert.ID), 0o755); err != nil {
		return err
	}
	if keyPEM == "" {
		keyPEM = "-----BEGIN PRIVATE KEY-----\nTEST\n-----END PRIVATE KEY-----\n"
	}
	if err := os.WriteFile(filepath.Join(s.path(cert.ID), "privkey.pem"), []byte(keyPEM), 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(s.path(cert.ID), "fullchain.pem"), []byte("-----BEGIN CERTIFICATE-----\nTEST\n-----END CERTIFICATE-----\n"), 0o644); err != nil {
		return err
	}
	if cert.Sites == nil {
		cert.Sites = []string{}
	}
	if cert.Attempts == nil {
		cert.Attempts = []Attempt{}
	}
	return s.write(cert)
}

func (s *Store) Bind(id string, sites []string) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, err := s.read(id)
	if err != nil {
		return Cert{}, err
	}
	seen := map[string]bool{}
	for _, siteID := range cert.Sites {
		seen[siteID] = true
	}
	for _, siteID := range sites {
		if !seen[siteID] {
			cert.Sites = append(cert.Sites, siteID)
			seen[siteID] = true
		}
	}
	if err := s.write(cert); err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

func (s *Store) KeyPath(id string) string {
	return filepath.Join(s.path(id), "privkey.pem")
}

func (s *Store) NginxDir(id string) string {
	return "certs/" + id
}

func (s *Store) SetHTTP01(fn HTTP01Func) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.http01 = fn
}

func (s *Store) RenewDue(ctx context.Context, now time.Time) ([]Attempt, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	var due []Cert
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cert, err := s.read(entry.Name())
		if err != nil || cert.Provider != "letsencrypt" {
			continue
		}
		expiry, err := time.Parse(time.RFC3339, cert.Expiry)
		if err != nil || expiry.After(now.Add(30*24*time.Hour)) {
			continue
		}
		due = append(due, cert)
	}
	renew := s.http01
	s.mu.Unlock()

	var attempts []Attempt
	for _, cert := range due {
		attempt := Attempt{At: now.UTC().Format(time.RFC3339), Method: "http-01", Status: "submitted"}
		if renew == nil {
			attempt.Status = "failed"
			attempt.Error = "HTTP-01 续签未配置"
		} else {
			expiry, err := renew(ctx, cert.ID, cert.Domains)
			if err != nil {
				attempt.Status = "failed"
				attempt.Error = err.Error()
			} else {
				attempt.Status = "success"
				cert.Expiry = expiry
				s.importIssued(cert.ID)
			}
		}
		s.mu.Lock()
		current, err := s.read(cert.ID)
		if err != nil {
			s.mu.Unlock()
			return attempts, err
		}
		if attempt.Status == "success" {
			current.Expiry = cert.Expiry
		}
		current.Attempts = append(current.Attempts, attempt)
		if err := s.write(current); err != nil {
			s.mu.Unlock()
			return attempts, err
		}
		s.mu.Unlock()
		attempts = append(attempts, attempt)
	}
	return attempts, nil
}

func (s *Store) importIssued(id string) {
	src := filepath.Join(filepath.Dir(s.dir), id)
	dst := s.path(id)
	for _, name := range []string{"fullchain.pem", "privkey.pem"} {
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil || len(b) == 0 {
			continue
		}
		mode := os.FileMode(0o644)
		if name == "privkey.pem" {
			mode = 0o600
		}
		_ = os.WriteFile(filepath.Join(dst, name), b, mode)
	}
}

func (s *Store) SaveDNS(cfg DNSConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(filepath.Dir(s.dir), "dns.json"), b, 0o600)
}

func (s *Store) DNSPublic() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(s.dir), "dns.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{"provider": "", "token_set": false}, nil
		}
		return nil, err
	}
	var cfg DNSConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return map[string]any{"provider": cfg.Provider, "zone": cfg.Zone, "api_base": cfg.APIBase, "token_set": cfg.Token != ""}, nil
}

func (s *Store) ManualTXT(name, value string) map[string]string {
	return map[string]string{"name": name, "type": "TXT", "value": value}
}

func (s *Store) DNSConfig() (DNSConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := os.ReadFile(filepath.Join(filepath.Dir(s.dir), "dns.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return DNSConfig{}, nil
		}
		return DNSConfig{}, err
	}
	var cfg DNSConfig
	err = json.Unmarshal(b, &cfg)
	return cfg, err
}

func (s *Store) SaveIssued(provider, issuer string, domains []string, expiry time.Time, certPEM, keyPEM string) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, err := newID()
	if err != nil {
		return Cert{}, err
	}
	if err := os.MkdirAll(s.path(id), 0o755); err != nil {
		return Cert{}, err
	}
	if certPEM == "" {
		certPEM = "-----BEGIN CERTIFICATE-----\nISSUED\n-----END CERTIFICATE-----\n"
	}
	if keyPEM == "" {
		keyPEM = "-----BEGIN PRIVATE KEY-----\nISSUED\n-----END PRIVATE KEY-----\n"
	}
	if err := os.WriteFile(filepath.Join(s.path(id), "fullchain.pem"), []byte(certPEM), 0o644); err != nil {
		return Cert{}, err
	}
	if err := os.WriteFile(filepath.Join(s.path(id), "privkey.pem"), []byte(keyPEM), 0o600); err != nil {
		return Cert{}, err
	}
	if domains == nil {
		domains = []string{}
	}
	if issuer == "" {
		issuer = provider
	}
	cert := Cert{
		ID: id, Domains: domains, Provider: provider, Issuer: issuer,
		Expiry: expiry.UTC().Format(time.RFC3339), CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Sites: []string{}, Attempts: []Attempt{},
	}
	if err := s.write(cert); err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

func (s *Store) AddAttempt(id string, attempt Attempt) (Cert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cert, err := s.read(id)
	if err != nil {
		return Cert{}, err
	}
	cert.Attempts = append(cert.Attempts, attempt)
	if err := s.write(cert); err != nil {
		return Cert{}, err
	}
	return publicCert(cert), nil
}

func (s *Store) path(id string) string { return filepath.Join(s.dir, id) }

func (s *Store) read(id string) (Cert, error) {
	if strings.Contains(id, "..") || strings.Contains(id, "/") {
		return Cert{}, fmt.Errorf("证书不存在")
	}
	b, err := os.ReadFile(filepath.Join(s.path(id), "meta.json"))
	if err != nil {
		return Cert{}, err
	}
	var cert Cert
	err = json.Unmarshal(b, &cert)
	return cert, err
}

func (s *Store) write(cert Cert) error {
	b, err := json.MarshalIndent(cert, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.path(cert.ID), "meta.json"), b, 0o644)
}

func publicCert(cert Cert) Cert {
	if cert.Sites == nil {
		cert.Sites = []string{}
	}
	if cert.Attempts == nil {
		cert.Attempts = []Attempt{}
	}
	return cert
}

func newID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("cert-%x", b), nil
}

func writeSelfSigned(dir string, domains []string, expiry time.Time) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	tmpl := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: domains[0], Organization: []string{"CamoPanel"}},
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: expiry,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    domains, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), certPEM, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "privkey.pem"), keyPEM, 0o600)
}
