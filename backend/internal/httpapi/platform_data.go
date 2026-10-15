package httpapi

import (
	"encoding/base64"
	"strconv"
	"strings"

	"camopanel/backend/internal/cron"
	"github.com/gin-gonic/gin"
)

func (p *platform) mountDB(api *gin.RouterGroup) {
	api.GET("/databases", func(c *gin.Context) {
		c.JSON(200, p.db.List())
	})
	api.GET("/databases/redis/keys", func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.Query("limit"))
		keys, err := p.db.RedisScan(c.Query("prefix"), limit)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"keys": keys})
	})
	api.POST("/databases/redis/keys", func(c *gin.Context) {
		var body struct {
			Key string `json:"key"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Key == "" {
			badRequest(c, "请填写键名")
			return
		}
		p.db.RedisAdd(body.Key)
		c.JSON(201, gin.H{"key": body.Key})
	})
	api.DELETE("/databases/redis/keys", func(c *gin.Context) {
		if err := p.db.RedisDel(c.Query("key")); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.JSON(200, gin.H{"message": "已删除"})
	})
	api.GET("/databases/:kind", func(c *gin.Context) {
		kind := c.Param("kind")
		for _, item := range p.db.List() {
			if item.Kind == kind {
				c.JSON(200, item)
				return
			}
		}
		c.JSON(200, p.db.Missing(kind))
	})
	api.POST("/databases/:kind/databases", func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			badRequest(c, "请填写数据库名")
			return
		}
		if err := p.db.CreateDatabase(c.Param("kind"), body.Name); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Set("audit_detail", "create database "+body.Name)
		c.JSON(201, gin.H{"name": body.Name})
	})
	api.POST("/databases/:kind/users", func(c *gin.Context) {
		var body struct {
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			badRequest(c, "请填写用户名")
			return
		}
		if err := p.db.CreateUser(c.Param("kind"), body.Name, body.Password); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Set("audit_detail", "create user "+body.Name)
		c.JSON(201, gin.H{"name": body.Name})
	})
	api.POST("/databases/:kind/export", func(c *gin.Context) {
		dump, err := p.db.Export(c.Param("kind"))
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Set("audit_detail", "export")
		c.JSON(200, gin.H{"sql": dump})
	})
	api.POST("/databases/:kind/import", func(c *gin.Context) {
		var body struct {
			SQL string `json:"sql"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			badRequest(c, "请提供 SQL")
			return
		}
		if err := p.db.Import(c.Param("kind"), body.SQL); err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Set("audit_detail", body.SQL)
		c.JSON(200, gin.H{"message": "已导入"})
	})
	api.POST("/databases/:kind/exec", func(c *gin.Context) {
		var body struct {
			SQL      string `json:"sql"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.SQL) == "" {
			badRequest(c, "请填写 SQL")
			return
		}
		out, err := p.db.Exec(c.Param("kind"), body.SQL)
		if err != nil {
			badRequest(c, err.Error())
			return
		}
		c.Set("audit_detail", body.SQL)
		c.JSON(200, gin.H{"result": out})
	})
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
