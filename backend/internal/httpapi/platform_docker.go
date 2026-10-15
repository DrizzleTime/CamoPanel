package httpapi

import (
	"fmt"
	"path/filepath"
	"strings"

	"camopanel/backend/internal/docker"
	"github.com/gin-gonic/gin"
)

func (p *platform) createContainer(c *gin.Context) {
	var req docker.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil || !validCreate(req) {
		badRequest(c, "请填写合法名称和镜像；端口须同时填写，范围 1–65535")
		return
	}
	if !validRestart(req.RestartPolicy) {
		badRequest(c, "重启策略无效")
		return
	}
	for _, bind := range req.Binds {
		if err := checkBind(p.cfg.DataDir, bind, req.ConfirmHostMount); err != nil {
			badRequest(c, err.Error())
			return
		}
	}
	if req.Pull {
		if _, err := p.docker.Pull(c.Request.Context(), req.Image); err != nil {
			failure(c, err)
			return
		}
	}
	id, err := p.docker.Create(c.Request.Context(), req)
	if err != nil {
		failure(c, err)
		return
	}
	c.JSON(201, gin.H{"id": id, "message": "容器已创建，可手动启动"})
}

func validCreate(req docker.CreateRequest) bool {
	if !namePattern.MatchString(req.Name) || strings.TrimSpace(req.Image) == "" {
		return false
	}
	if len(req.Ports) > 0 {
		for _, port := range req.Ports {
			if port.ContainerPort < 1 || port.ContainerPort > 65535 || port.HostPort < 0 || port.HostPort > 65535 {
				return false
			}
		}
		return true
	}
	return (req.HostPort == 0) == (req.ContainerPort == 0)
}

func checkBind(dataDir, bind string, confirm bool) error {
	host := bindHost(bind)
	if host == "" {
		return fmt.Errorf("挂载路径无效")
	}
	if !filepath.IsAbs(host) {
		host = filepath.Join(dataDir, host)
	}
	absData, err := filepath.Abs(dataDir)
	if err != nil {
		return err
	}
	absHost, err := filepath.Abs(host)
	if err != nil {
		return err
	}
	if absHost == absData || strings.HasPrefix(absHost, absData+string(filepath.Separator)) {
		return nil
	}
	if confirm {
		return nil
	}
	return fmt.Errorf("挂载位于数据目录之外，需要确认")
}

func bindHost(bind string) string {
	if strings.HasPrefix(bind, "/") {
		rest := bind[1:]
		if i := strings.Index(rest, ":"); i >= 0 {
			return "/" + rest[:i]
		}
		return bind
	}
	if i := strings.Index(bind, ":"); i >= 0 {
		return bind[:i]
	}
	return bind
}

func (p *platform) mountDockerExtra(api *gin.RouterGroup) {
	api.GET("/docker/containers/:id", func(c *gin.Context) {
		detail, err := p.docker.Inspect(c.Request.Context(), c.Param("id"))
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, detail)
	})
	api.GET("/docker/containers/:id/stats", func(c *gin.Context) {
		stats, err := p.docker.Stats(c.Request.Context(), c.Param("id"))
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, stats)
	})
	api.GET("/docker/containers/:id/logs/stream", func(c *gin.Context) {
		logs, err := p.docker.Logs(c.Request.Context(), c.Param("id"))
		if err != nil {
			failure(c, err)
			return
		}
		lines := strings.Split(strings.TrimRight(logs, "\n"), "\n")
		tail := 200
		if raw := c.Query("tail"); raw != "" {
			fmt.Sscanf(raw, "%d", &tail)
		}
		if tail < 1 {
			tail = 1
		}
		if tail > 500 {
			tail = 500
		}
		if len(lines) > tail {
			lines = lines[len(lines)-tail:]
		}
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-store")
		for _, line := range lines {
			_, _ = c.Writer.Write([]byte("data: " + line + "\n\n"))
		}
	})
	api.POST("/docker/containers/:id/force-remove", func(c *gin.Context) {
		var body struct {
			Confirm bool `json:"confirm"`
		}
		_ = c.ShouldBindJSON(&body)
		if !body.Confirm {
			badRequest(c, "强制删除需要确认，且默认不会删除数据卷")
			return
		}
		if err := p.docker.ForceRemove(c.Request.Context(), c.Param("id")); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"message": "容器已强制删除", "volumes_deleted": false})
	})
	api.POST("/docker/images/pull", func(c *gin.Context) {
		var body struct {
			Image string `json:"image"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Image) == "" {
			badRequest(c, "请填写镜像名")
			return
		}
		events, err := p.docker.Pull(c.Request.Context(), body.Image)
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"events": events})
	})
	api.DELETE("/docker/images/:id", func(c *gin.Context) {
		if err := p.docker.DeleteImage(c.Request.Context(), c.Param("id")); err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"message": "镜像已删除"})
	})
	api.POST("/docker/images/prune", func(c *gin.Context) {
		n, err := p.docker.PruneDangling(c.Request.Context())
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"removed": n})
	})
}
