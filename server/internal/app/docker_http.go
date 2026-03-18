package app

import (
	"errors"
	"net/http"
	"strconv"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type dockerContainerActionRequest struct {
	Action string `json:"action"`
}

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

func (a *App) handleDockerContainerAction(c *gin.Context) {
	var req dockerContainerActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	var err error
	switch req.Action {
	case model.ActionStart:
		err = a.docker.StartContainer(c.Request.Context(), c.Param("id"))
	case model.ActionStop:
		err = a.docker.StopContainer(c.Request.Context(), c.Param("id"))
	case model.ActionRestart:
		err = a.docker.RestartContainer(c.Request.Context(), c.Param("id"))
	case model.ActionDelete:
		err = a.docker.DeleteContainer(c.Request.Context(), c.Param("id"))
	default:
		writeError(c, http.StatusBadRequest, "不支持的动作")
		return
	}

	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
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

func (a *App) handleDockerImageDelete(c *gin.Context) {
	if err := a.docker.RemoveImage(c.Request.Context(), c.Param("id")); err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (a *App) handleDockerImagePrune(c *gin.Context) {
	result, err := a.docker.PruneUnusedImages(c.Request.Context())
	if err != nil {
		if writeDockerError(c, err) {
			return
		}
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, result)
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
	case errors.Is(err, services.ErrImageNotFound):
		writeError(c, http.StatusNotFound, "镜像不存在")
	default:
		return false
	}

	return true
}
