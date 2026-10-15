package httpapi

import (
	"encoding/json"
	"os"
	"time"

	"camopanel/backend/internal/alert"
	"github.com/gin-gonic/gin"
)

func (p *platform) mountSecurity(api *gin.RouterGroup) {
	api.GET("/firewall", func(c *gin.Context) {
		c.JSON(200, p.firewall.Status())
	})
	api.POST("/firewall/rules", func(c *gin.Context) {
		var body struct {
			Port    int    `json:"port"`
			Policy  string `json:"policy"`
			Confirm bool   `json:"confirm"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Port < 1 || body.Port > 65535 {
			badRequest(c, "请填写端口和策略")
			return
		}
		status, err := p.firewall.Add(body.Port, body.Policy, body.Confirm)
		if err != nil {
			c.JSON(400, gin.H{"error": gin.H{"message": err.Error()}, "firewall": status})
			return
		}
		c.JSON(201, status)
	})
	api.GET("/alerts", func(c *gin.Context) {
		c.JSON(200, p.alerts.List())
	})
	api.POST("/alerts", func(c *gin.Context) {
		var rule alert.Rule
		if err := c.ShouldBindJSON(&rule); err != nil || rule.Kind == "" {
			badRequest(c, "请填写告警类型")
			return
		}
		switch rule.Kind {
		case "disk", "cert_expiry", "container_exit", "site_down":
		default:
			badRequest(c, "不支持的告警类型")
			return
		}
		c.JSON(201, p.alerts.Add(rule))
	})
	api.POST("/alerts/evaluate", func(c *gin.Context) {
		var sample alert.Sample
		if err := c.ShouldBindJSON(&sample); err != nil {
			badRequest(c, "请提供采样")
			return
		}
		sent, err := p.alerts.Evaluate(sample)
		if err != nil && sent == 0 {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"sent": sent})
	})
	api.POST("/terminals/host", func(c *gin.Context) {
		id := p.recordTerminal(c.GetString("actor"), "host", "")
		c.JSON(200, gin.H{"id": id, "kind": "host", "account": c.GetString("actor")})
	})
	api.POST("/terminals/containers/:id", func(c *gin.Context) {
		if _, err := p.docker.Inspect(c.Request.Context(), c.Param("id")); err != nil {
			failure(c, err)
			return
		}
		id := p.recordTerminal(c.GetString("actor"), "container", c.Param("id"))
		c.JSON(200, gin.H{"id": id, "kind": "container", "account": c.GetString("actor"), "container": c.Param("id")})
	})
}

func (p *platform) recordTerminal(account, kind, target string) string {
	id := "term-" + timeNow().Format("20060102150405.000")
	if p.terms == "" {
		return id
	}
	for _, event := range []string{"start", "end"} {
		rec := map[string]string{"time": time.Now().UTC().Format(time.RFC3339), "account": account, "kind": kind, "event": event, "id": id, "target": target}
		b, _ := json.Marshal(rec)
		f, err := os.OpenFile(p.terms, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return id
		}
		_, _ = f.Write(append(b, '\n'))
		f.Close()
	}
	return id
}
