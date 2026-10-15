package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"camopanel/backend/internal/certs"
	"github.com/gin-gonic/gin"
)

func (p *platform) mountCerts(api *gin.RouterGroup) {
	api.GET("/certificates", func(c *gin.Context) {
		if p.certs == nil {
			c.JSON(200, []certs.Cert{})
			return
		}
		list, err := p.certs.List()
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, list)
	})
	api.POST("/certificates/self-signed", func(c *gin.Context) {
		var body struct {
			Domains []string `json:"domains"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Domains) == 0 {
			badRequest(c, "请填写域名")
			return
		}
		cert, err := p.certs.IssueSelfSigned(body.Domains)
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(201, cert)
	})
	api.POST("/certificates/upload", func(c *gin.Context) {
		var body struct {
			CertPEM string   `json:"cert_pem"`
			KeyPEM  string   `json:"key_pem"`
			Domains []string `json:"domains"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请提供证书和私钥")
			return
		}
		cert, err := p.certs.Upload(body.CertPEM, body.KeyPEM, body.Domains)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, cert)
	})
	api.POST("/certificates/issued", func(c *gin.Context) {
		var body struct {
			Provider string   `json:"provider"`
			Domains  []string `json:"domains"`
			Expiry   string   `json:"expiry"`
			CertPEM  string   `json:"cert_pem"`
			KeyPEM   string   `json:"key_pem"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Provider == "" {
			badRequest(c, "请填写证书来源")
			return
		}
		expiry := timeNow().Add(90 * 24 * time.Hour)
		if body.Expiry != "" {
			parsed, err := parseTime(body.Expiry)
			if err != nil {
				badRequest(c, "到期时间格式不正确")
				return
			}
			expiry = parsed
		}
		cert, err := p.certs.SaveIssued(body.Provider, body.Provider, body.Domains, expiry, body.CertPEM, body.KeyPEM)
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(201, cert)
	})
	api.GET("/certificates/dns", func(c *gin.Context) {
		pub, err := p.certs.DNSPublic()
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, pub)
	})
	api.PUT("/certificates/dns", func(c *gin.Context) {
		var body struct {
			Provider string `json:"provider"`
			APIBase  string `json:"api_base"`
			Token    string `json:"token"`
			Zone     string `json:"zone"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Provider == "" {
			badRequest(c, "请填写 DNS 服务商")
			return
		}
		if err := p.certs.SaveDNS(certs.DNSConfig{Provider: body.Provider, APIBase: body.APIBase, Token: body.Token, Zone: body.Zone}); err != nil {
			failure(c, err)
			return
		}
		pub, err := p.certs.DNSPublic()
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, pub)
	})
	api.POST("/certificates/dns/manual", func(c *gin.Context) {
		var body struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" || body.Value == "" {
			badRequest(c, "请填写 TXT 记录")
			return
		}
		c.JSON(200, p.certs.ManualTXT(body.Name, body.Value))
	})
	api.POST("/certificates/dns/issue", func(c *gin.Context) {
		cfg, err := p.certs.DNSConfig()
		if err != nil {
			failure(c, err)
			return
		}
		var body struct {
			Domains []string `json:"domains"`
		}
		_ = c.ShouldBindJSON(&body)
		if len(body.Domains) == 0 {
			body.Domains = []string{cfg.Zone}
		}
		status := "failed"
		message := "DNS 接口没有配置"
		if cfg.APIBase != "" && cfg.Zone != "" {
			payload, _ := json.Marshal(map[string]any{"name": "_acme-challenge." + cfg.Zone, "type": "TXT"})
			req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, strings.TrimRight(cfg.APIBase, "/")+"/zones/"+cfg.Zone+"/txt", bytes.NewReader(payload))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
				if cfg.Token != "" {
					req.Header.Set("Authorization", "Bearer "+cfg.Token)
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					message = stripSecret(err.Error(), cfg.Token)
				} else {
					raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
					resp.Body.Close()
					if resp.StatusCode < 300 {
						status = "submitted"
						message = "DNS-01 已提交"
					} else {
						message = stripSecret(string(raw), cfg.Token)
					}
				}
			}
		}
		cert, err := p.certs.SaveIssued("dns", cfg.Provider, body.Domains, timeNow().Add(90*24*time.Hour), "", "")
		if err != nil {
			failure(c, err)
			return
		}
		updated, err := p.certs.AddAttempt(cert.ID, certs.Attempt{At: timeNow().Format(time.RFC3339), Method: "dns-01", Status: status, Error: message})
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"status": status, "message": message, "certificate": updated})
	})
	api.POST("/certificates/:id/bind", func(c *gin.Context) {
		var body struct {
			Sites []string `json:"sites"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.Sites) == 0 {
			badRequest(c, "请指定要绑定的网站")
			return
		}
		cert, err := p.certs.Bind(c.Param("id"), body.Sites)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		for _, id := range body.Sites {
			if _, err := p.sites.AttachCert(c.Request.Context(), id, cert.ID, cert.Expiry, cert.Provider); err != nil {
				badRequest(c, err.Error())
				return
			}
		}
		c.JSON(200, cert)
	})
}

func stripSecret(message, secret string) string {
	if secret == "" {
		return message
	}
	return strings.ReplaceAll(message, secret, "[redacted]")
}
