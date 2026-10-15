package site

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
)

type SSLManager struct {
	dataDir    string
	acmeEmail  string
	acmeDirURL string
}

func NewSSLManager(dataDir, acmeEmail string) *SSLManager {
	return &SSLManager{
		dataDir:    dataDir,
		acmeEmail:  acmeEmail,
		acmeDirURL: acme.LetsEncryptURL,
	}
}

// SetDirectory points HTTP-01 at an ACME directory. An empty value keeps Let's Encrypt.
func (m *SSLManager) SetDirectory(raw string) {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		m.acmeDirURL = raw
	}
}

func (m *SSLManager) siteSSLDir(siteID string) string {
	return filepath.Join(m.dataDir, "ssl", siteID)
}

func (m *SSLManager) acmeWebrootDir() string {
	return filepath.Join(m.dataDir, "acme", ".well-known", "acme-challenge")
}

// IssueSelfSignedCert generates a self-signed certificate for the given domains.
// This is ideal for local dev, intranet, or staging testing where public DNS is unavailable.
func (m *SSLManager) IssueSelfSignedCert(siteID string, domains []string) (string, error) {
	sslDir := m.siteSSLDir(siteID)
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return "", err
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", err
	}

	notBefore := time.Now().Add(-1 * time.Hour)
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	primary := "localhost"
	if len(domains) > 0 && domains[0] != "" {
		primary = domains[0]
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   primary,
			Organization: []string{"CamoPanel Self-Signed"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if ip := net.ParseIP(d); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, d)
		}
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", err
	}

	certPath := filepath.Join(sslDir, "fullchain.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return "", err
	}

	keyPath := filepath.Join(sslDir, "privkey.pem")
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer keyOut.Close()
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return "", err
	}

	return notAfter.Format(time.RFC3339), nil
}

// SaveCustomCert saves a user-provided PEM certificate and private key.
func (m *SSLManager) SaveCustomCert(siteID, certPEM, keyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil || block.Type != "CERTIFICATE" {
		return "", fmt.Errorf("无效的证书 PEM 内容")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("解析证书失败: %w", err)
	}

	keyBlock, _ := pem.Decode([]byte(keyPEM))
	if keyBlock == nil || !strings.Contains(keyBlock.Type, "PRIVATE KEY") {
		return "", fmt.Errorf("无效的私钥 PEM 内容")
	}

	sslDir := m.siteSSLDir(siteID)
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(filepath.Join(sslDir, "fullchain.pem"), []byte(strings.TrimSpace(certPEM)+"\n"), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(sslDir, "privkey.pem"), []byte(strings.TrimSpace(keyPEM)+"\n"), 0600); err != nil {
		return "", err
	}

	return cert.NotAfter.Format(time.RFC3339), nil
}

// IssueLetsEncryptCert requests a free SSL certificate from Let's Encrypt via ACME HTTP-01 challenge.
func (m *SSLManager) IssueLetsEncryptCert(ctx context.Context, siteID string, domains []string, email string) (string, error) {
	if len(domains) == 0 {
		return "", fmt.Errorf("必须至少指定一个有效域名")
	}
	if email == "" {
		email = m.acmeEmail
	}

	// Ensure challenge directory exists
	challengeDir := m.acmeWebrootDir()
	if err := os.MkdirAll(challengeDir, 0755); err != nil {
		return "", err
	}

	// 1. Generate ACME account key
	accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", err
	}

	client := &acme.Client{
		Key:          accountKey,
		DirectoryURL: m.acmeDirURL,
	}

	// 2. Register account
	account := &acme.Account{
		Contact: []string{"mailto:" + email},
	}
	_, err = client.Register(ctx, account, acme.AcceptTOS)
	if err != nil && !strings.Contains(err.Error(), "already registered") {
		// ACME may require account key registration
	}

	// 3. Create Order for all domains
	var authzIDs []acme.AuthzID
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d != "" {
			authzIDs = append(authzIDs, acme.AuthzID{Type: "dns", Value: d})
		}
	}

	order, err := client.AuthorizeOrder(ctx, authzIDs)
	if err != nil {
		return "", fmt.Errorf("Let's Encrypt 创建证书申请失败: %w", err)
	}

	// 4. Handle HTTP-01 challenge for each authorization URL
	for _, authzURL := range order.AuthzURLs {
		authz, err := client.GetAuthorization(ctx, authzURL)
		if err != nil {
			return "", fmt.Errorf("获取验证信息失败: %w", err)
		}
		if authz.Status == acme.StatusValid {
			continue
		}

		var challenge *acme.Challenge
		for _, c := range authz.Challenges {
			if c.Type == "http-01" {
				challenge = c
				break
			}
		}
		if challenge == nil {
			return "", fmt.Errorf("域名验证不支持 HTTP-01 验证方式")
		}

		keyAuth, err := client.HTTP01ChallengeResponse(challenge.Token)
		if err != nil {
			return "", fmt.Errorf("计算验证秘钥响应失败: %w", err)
		}

		// Write challenge token file to acme challenge directory
		tokenPath := filepath.Join(challengeDir, challenge.Token)
		if err := os.WriteFile(tokenPath, []byte(keyAuth), 0644); err != nil {
			return "", fmt.Errorf("写入验证文件失败: %w", err)
		}
		defer os.Remove(tokenPath)

		// Tell ACME server challenge is ready
		if _, err := client.Accept(ctx, challenge); err != nil {
			return "", fmt.Errorf("提交 ACME 验证失败: %w", err)
		}

		// Wait for authorization
		if _, err := client.WaitAuthorization(ctx, challenge.URI); err != nil {
			return "", fmt.Errorf("Let's Encrypt HTTP-01 验证未通过，请检查域名解析是否指向本机 80 端口: %w", err)
		}
	}

	// 5. Generate Cert Key and CSR
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	primary := domains[0]
	csrTemplate := x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: primary},
		DNSNames: domains,
	}
	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, certKey)
	if err != nil {
		return "", err
	}

	// 6. Finalize Order & Fetch Certificate
	derCerts, certURL, err := client.CreateOrderCert(ctx, order.FinalizeURL, csrBytes, true)
	if err != nil {
		return "", fmt.Errorf("下载证书签名失败: %w", err)
	}
	_ = certURL

	// 7. Save cert and key to site SSL directory
	sslDir := m.siteSSLDir(siteID)
	if err := os.MkdirAll(sslDir, 0755); err != nil {
		return "", err
	}

	certPath := filepath.Join(sslDir, "fullchain.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		return "", err
	}
	defer certFile.Close()

	var firstCert *x509.Certificate
	for _, b := range derCerts {
		if firstCert == nil {
			firstCert, _ = x509.ParseCertificate(b)
		}
		if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: b}); err != nil {
			return "", err
		}
	}

	keyPath := filepath.Join(sslDir, "privkey.pem")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", err
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)}); err != nil {
		return "", err
	}

	if firstCert != nil {
		return firstCert.NotAfter.Format(time.RFC3339), nil
	}
	return time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339), nil
}

// GetSSLInfo inspects the SSL files for a site and returns detailed status.
func (m *SSLManager) GetSSLInfo(siteID string) (*SSLInfo, error) {
	certPath := filepath.Join(m.siteSSLDir(siteID), "fullchain.pem")
	b, err := os.ReadFile(certPath)
	if err != nil {
		return &SSLInfo{Enabled: false}, nil
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return &SSLInfo{Enabled: false}, nil
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return &SSLInfo{Enabled: false}, nil
	}

	days := int(time.Until(cert.NotAfter).Hours() / 24)
	return &SSLInfo{
		Enabled:       true,
		Domains:       cert.DNSNames,
		Expiry:        cert.NotAfter.Format(time.RFC3339),
		DaysRemaining: days,
		Issuer:        cert.Issuer.CommonName,
	}, nil
}
