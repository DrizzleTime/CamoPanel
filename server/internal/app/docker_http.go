package app

import (
	"errors"
	"net/http"
	"strconv"

	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

func (a *App) handleDockerContainers(c *gin.Context) {
	items, err := a.docker.ListContainers(c.Request.Context())
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleDockerContainerLogs(c *gin.Context) {
	tail, _ := strconv.Atoi(c.DefaultQuery("tail", "200"))
	if tail <= 0 {
		tail = 200
	}

	logs, err := a.docker.ContainerLogs(c.Request.Context(), c.Param("id"), tail)
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (a *App) handleDockerImages(c *gin.Context) {
	items, err := a.docker.ListImages(c.Request.Context())
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleDockerNetworks(c *gin.Context) {
	items, err := a.docker.ListNetworks(c.Request.Context())
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleDockerSystem(c *gin.Context) {
	info, err := a.docker.GetSystemInfo(c.Request.Context())
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, info)
}

func writeDockerError(c *gin.Context, err error) bool {
	switch {
	case errors.Is(err, services.ErrDockerUnavailable):
		writeError(c, http.StatusBadGateway, "Docker 当前不可用")
	case errors.Is(err, services.ErrContainerNotFound):
		writeError(c, http.StatusNotFound, "容器不存在")
	default:
		return false
	}

	return true
}
