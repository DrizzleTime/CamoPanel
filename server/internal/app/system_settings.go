package app

import (
	"net/http"

	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type updateDockerSettingsRequest struct {
	RegistryMirrors []string `json:"registry_mirrors"`
}

func (a *App) handleDockerSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	settings, err := a.hostControl.GetDockerSettings(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusBadGateway, err.Error())
		return
	}

	c.JSON(http.StatusOK, settings)
}

func (a *App) handleUpdateDockerSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var req updateDockerSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	settings, err := a.hostControl.UpdateDockerSettings(c.Request.Context(), req.RegistryMirrors)
	if err != nil {
		if err == services.ErrHostControlUnavailable {
			writeError(c, http.StatusServiceUnavailable, "宿主机控制未启用")
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "docker_settings_update", "system", settings.ConfigPath, map[string]any{
		"registry_mirrors": settings.RegistryMirrors,
	})
	c.JSON(http.StatusOK, settings)
}

func (a *App) handleRestartDockerDaemon(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	if err := a.hostControl.RestartDocker(c.Request.Context()); err != nil {
		if err == services.ErrHostControlUnavailable {
			writeError(c, http.StatusServiceUnavailable, "宿主机控制未启用")
			return
		}
		writeError(c, http.StatusBadGateway, err.Error())
		return
	}

	_ = a.recordAudit(currentUser(c).ID, "docker_restart", "system", "docker", nil)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
