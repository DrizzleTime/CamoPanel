package api

import (
	"context"
	"errors"
	"strconv"

	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type DockerReader interface {
	ListContainers(ctx context.Context) ([]platformdocker.Container, error)
	ListImages(ctx context.Context) ([]platformdocker.Image, error)
	ListNetworks(ctx context.Context) ([]platformdocker.Network, error)
	ContainerLogs(ctx context.Context, containerID string, tail int) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RestartContainer(ctx context.Context, containerID string) error
	DeleteContainer(ctx context.Context, containerID string) error
	RemoveImage(ctx context.Context, imageID string) error
	PruneUnusedImages(ctx context.Context) (platformdocker.ImagePruneResult, error)
}

type Handler struct {
	docker DockerReader
}

func NewHandler(docker DockerReader) *Handler {
	return &Handler{docker: docker}
}

func (h *Handler) ListContainers(c *gin.Context) {
	items, err := h.docker.ListContainers(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) ContainerLogs(c *gin.Context) {
	tail, _ := strconv.Atoi(c.DefaultQuery("tail", "200"))
	if tail <= 0 {
		tail = 200
	}

	logs, err := h.docker.ContainerLogs(c.Request.Context(), c.Param("id"), tail)
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"logs": logs})
}

func (h *Handler) ContainerAction(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}

	var err error
	switch req.Action {
	case "start":
		err = h.docker.StartContainer(c.Request.Context(), c.Param("id"))
	case "stop":
		err = h.docker.StopContainer(c.Request.Context(), c.Param("id"))
	case "restart":
		err = h.docker.RestartContainer(c.Request.Context(), c.Param("id"))
	case "delete":
		err = h.docker.DeleteContainer(c.Request.Context(), c.Param("id"))
	default:
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "不支持的动作"))
		return
	}
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) ListImages(c *gin.Context) {
	items, err := h.docker.ListImages(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) DeleteImage(c *gin.Context) {
	if err := h.docker.RemoveImage(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) PruneImages(c *gin.Context) {
	result, err := h.docker.PruneUnusedImages(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) ListNetworks(c *gin.Context) {
	items, err := h.docker.ListNetworks(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapDockerError(err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func mapDockerError(err error) error {
	switch {
	case errors.Is(err, platformdocker.ErrUnavailable):
		return errs.E(errs.CodeUnavailable, "Docker 当前不可用")
	case errors.Is(err, platformdocker.ErrContainerNotFound):
		return errs.E(errs.CodeNotFound, "容器不存在")
	case errors.Is(err, platformdocker.ErrImageNotFound):
		return errs.E(errs.CodeNotFound, "镜像不存在")
	default:
		return errs.Wrap(errs.CodeInternal, "docker operation failed", err)
	}
}
