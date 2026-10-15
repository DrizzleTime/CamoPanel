package httpapi

import (
	"encoding/base64"
	"net/http"
	"strings"

	"camopanel/backend/internal/cron"
	"github.com/gin-gonic/gin"
)

// Database operations stay unavailable until a real database backend is configured.
func (p *platform) mountDB(api *gin.RouterGroup) {
	info := func(kind string) gin.H {
		return gin.H{"kind": kind, "installed": false, "store_path": "/store?app=" + kind}
	}
	api.GET("/databases", func(c *gin.Context) {
		c.JSON(200, []gin.H{info("mysql"), info("postgres"), info("redis")})
	})
	unavailable := func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": gin.H{"message": "数据库控制台尚未接入真实数据库"}})
	}
	api.GET("/databases/redis/keys", unavailable)
	api.POST("/databases/redis/keys", unavailable)
	api.DELETE("/databases/redis/keys", unavailable)
	api.GET("/databases/:kind", func(c *gin.Context) {
		c.JSON(200, info(c.Param("kind")))
	})
	for _, operation := range []string{"databases", "users", "export", "import", "exec"} {
		api.POST("/databases/:kind/"+operation, unavailable)
	}
}

func (p *platform) mountBackups(api *gin.RouterGroup) {
	api.GET("/backups", func(c *gin.Context) {
		c.JSON(200, p.backups.List())
	})
	api.POST("/backups/sites/:id", func(c *gin.Context) {
		item, err := p.backups.CreateSite(c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, item)
	})
	api.POST("/backups/directories", func(c *gin.Context) {
		var body struct {
			Path string `json:"path"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请填写目录")
			return
		}
		item, err := p.backups.CreateDir(body.Path)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, item)
	})
	api.POST("/backups/import", func(c *gin.Context) {
		var body struct {
			Kind   string `json:"kind"`
			Source string `json:"source"`
			Zip    string `json:"zip"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Zip == "" {
			badRequest(c, "请提供备份包")
			return
		}
		raw, err := base64.StdEncoding.DecodeString(body.Zip)
		if err != nil {
			badRequest(c, "备份包不是有效的 base64")
			return
		}
		item, err := p.backups.CreateBytes(body.Kind, body.Source, body.Source, raw)
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(201, item)
	})
	api.POST("/backups/:id/restore", func(c *gin.Context) {
		var body struct {
			Confirm bool `json:"confirm"`
		}
		_ = c.ShouldBindJSON(&body)
		if err := p.backups.Restore(c.Param("id"), body.Confirm); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "已恢复"})
	})
	api.POST("/backups/:id/publish", func(c *gin.Context) {
		var body struct {
			Endpoint string `json:"endpoint"`
			Bucket   string `json:"bucket"`
			Access   string `json:"access_key"`
			Secret   string `json:"secret"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Endpoint == "" || body.Bucket == "" {
			badRequest(c, "请填写对象存储地址和桶")
			return
		}
		url, err := p.backups.Publish(c.Param("id"), body.Endpoint, body.Bucket, body.Access, body.Secret)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"remote": url})
	})
}

func (p *platform) mountCron(api *gin.RouterGroup) {
	api.GET("/schedules", func(c *gin.Context) {
		c.JSON(200, p.cron.List())
	})
	api.POST("/schedules", func(c *gin.Context) {
		var job cron.Job
		if err := c.ShouldBindJSON(&job); err != nil || job.Type == "" {
			badRequest(c, "请填写任务类型")
			return
		}
		created, err := p.cron.Create(job)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(201, created)
	})
	api.POST("/schedules/reload", func(c *gin.Context) {
		if err := p.cron.Reload(); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, p.cron.List())
	})
	api.POST("/schedules/scripts", func(c *gin.Context) {
		var body struct {
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			badRequest(c, "请填写脚本名")
			return
		}
		if strings.Contains(body.Name, "/") || strings.Contains(body.Name, "..") {
			badRequest(c, "脚本必须放在面板脚本目录")
			return
		}
		if err := writeScript(p.cfg.DataDir, body.Name, body.Content); err != nil {
			failure(c, err)
			return
		}
		c.JSON(201, gin.H{"name": body.Name})
	})
	api.POST("/schedules/:id/run", func(c *gin.Context) {
		run, err := p.cron.RunNow(c.Request.Context(), c.Param("id"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, run)
	})
	api.POST("/schedules/:id/enabled", func(c *gin.Context) {
		var body struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请指定是否启用")
			return
		}
		if err := p.cron.SetEnabled(c.Param("id"), body.Enabled); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"enabled": body.Enabled})
	})
}
