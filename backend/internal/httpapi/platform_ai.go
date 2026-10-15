package httpapi

import (
	"strings"

	"camopanel/backend/internal/assistant"
	"github.com/gin-gonic/gin"
)

func (p *platform) mountAI(api *gin.RouterGroup) {
	api.GET("/ai/settings", func(c *gin.Context) {
		settings, err := assistant.Load(p.cfg.DataDir)
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, assistant.Public(settings))
	})
	api.PUT("/ai/settings", func(c *gin.Context) {
		var body struct {
			BaseURL string            `json:"base_url"`
			APIKey  string            `json:"api_key"`
			Model   string            `json:"model"`
			Headers map[string]string `json:"headers"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写模型设置")
			return
		}
		current, _ := assistant.Load(p.cfg.DataDir)
		if strings.TrimSpace(body.BaseURL) == "" {
			body.BaseURL = "https://api.openai.com/v1"
		}
		if body.APIKey == "" {
			body.APIKey = current.APIKey
		}
		if body.Headers == nil {
			body.Headers = current.Headers
		} else if current.Headers != nil {
			for name, value := range body.Headers {
				if value == "" || value == "***" {
					if kept, ok := current.Headers[name]; ok {
						body.Headers[name] = kept
					}
				}
			}
		}
		settings := assistant.Settings{BaseURL: body.BaseURL, APIKey: body.APIKey, Model: body.Model, Headers: body.Headers}
		if err := assistant.Save(p.cfg.DataDir, settings); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, assistant.Public(settings))
	})
	api.POST("/ai/chat", func(c *gin.Context) {
		var body struct {
			Message string `json:"message"`
			Confirm bool   `json:"confirm"`
			Stream  bool   `json:"stream"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Message) == "" {
			badRequest(c, "请填写问题")
			return
		}
		settings, err := assistant.Load(p.cfg.DataDir)
		if err != nil {
			failure(c, err)
			return
		}
		reply, err := actorTools(p, c).Chat(c.Request.Context(), settings, body.Message, body.Confirm)
		if err != nil {
			if strings.Contains(err.Error(), "只读") {
				c.JSON(403, gin.H{"error": gin.H{"message": err.Error()}})
				return
			}
			failure(c, err)
			return
		}
		_ = body.Stream
		c.JSON(200, reply)
	})
}
