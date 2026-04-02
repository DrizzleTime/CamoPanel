package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	authdomain "camopanel/server/internal/modules/auth/domain"
	projectsusecase "camopanel/server/internal/modules/projects/usecase"
	systemusecase "camopanel/server/internal/modules/system/usecase"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
	platformauth "camopanel/server/internal/platform/auth"
	platformdocker "camopanel/server/internal/platform/docker"
	platformerrs "camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"
	platformsystem "camopanel/server/internal/platform/system"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

const dashboardStreamInterval = 1 * time.Second

type Handler struct {
	system    *platformsystem.Service
	dashboard *systemusecase.Dashboard
}

func NewHandler(system *platformsystem.Service, dashboard *systemusecase.Dashboard) *Handler {
	return &Handler{system: system, dashboard: dashboard}
}

func (h *Handler) HostSummary(c *gin.Context) {
	item, err := h.system.Summary(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, platformerrs.Wrap(platformerrs.CodeInternal, "load host summary", err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) HostMetrics(c *gin.Context) {
	item, err := h.system.Metrics(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, platformerrs.Wrap(platformerrs.CodeInternal, "load host metrics", err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) DashboardStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	push := func() {
		snapshot, err := h.dashboard.Build(c.Request.Context())
		if err != nil {
			writeSSE(c, "warning", gin.H{"error": err.Error()})
			return
		}
		writeSSE(c, "snapshot", snapshot)
	}

	push()
	ticker := time.NewTicker(dashboardStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			push()
		}
	}
}

func (h *Handler) DockerSystem(c *gin.Context) {
	item, err := h.system.DockerSystem(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) DockerSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	item, err := h.system.DockerSettings(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) UpdateDockerSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		RegistryMirrors []string `json:"registry_mirrors"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.system.UpdateDockerSettings(c.Request.Context(), req.RegistryMirrors)
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) RestartDocker(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	if err := h.system.RestartDocker(c.Request.Context()); err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) GetSystemConfig(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	cfg, err := h.system.GetSystemConfig(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, cfg)
}

func (h *Handler) UpdateHostname(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Hostname string `json:"hostname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.system.UpdateHostname(c.Request.Context(), req.Hostname); err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) UpdateDNS(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Servers []string `json:"servers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.system.UpdateDNS(c.Request.Context(), req.Servers); err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) UpdateTimezone(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Timezone string `json:"timezone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.system.UpdateTimezone(c.Request.Context(), req.Timezone); err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) CreateSwap(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		SizeMB int `json:"size_mb" binding:"required,min=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误，Swap 最小 64 MB"))
		return
	}
	swap, err := h.system.CreateSwap(c.Request.Context(), req.SizeMB)
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, swap)
}

func (h *Handler) RemoveSwap(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	swap, err := h.system.RemoveSwap(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, swap)
}

func (h *Handler) ScanCleanup(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	result, err := h.system.ScanCleanup(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, result)
}

func (h *Handler) ExecuteCleanup(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var req struct {
		Categories []string `json:"categories" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	result, err := h.system.ExecuteCleanup(c.Request.Context(), req.Categories)
	if err != nil {
		httpx.ErrorFrom(c, mapSystemError(err))
		return
	}
	httpx.OK(c, result)
}

func requireSuperAdmin(c *gin.Context) bool {
	value, ok := platformauth.CurrentSubject(c)
	if !ok {
		httpx.ErrorFrom(c, platformerrs.E(platformerrs.CodeUnauthenticated, "需要先登录"))
		return false
	}
	user, ok := value.(authdomain.User)
	if !ok || user.Role != authdomain.RoleSuperAdmin {
		httpx.Error(c, 403, "没有权限")
		return false
	}
	return true
}

func writeSSE(c *gin.Context, event string, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(raw) + "\n\n")
	c.Writer.Flush()
}

func mapSystemError(err error) error {
	switch {
	case errors.Is(err, platformdocker.ErrUnavailable):
		return platformerrs.E(platformerrs.CodeUnavailable, "Docker 当前不可用")
	case errors.Is(err, services.ErrHostControlUnavailable):
		return platformerrs.E(platformerrs.CodeUnavailable, "宿主机控制未启用")
	default:
		return platformerrs.E(platformerrs.CodeInvalidArgument, err.Error())
	}
}

type ProjectLister interface {
	Execute(ctx context.Context) ([]projectsusecase.ProjectView, error)
}

type WebsiteLister interface {
	Execute(ctx context.Context) ([]websitesdomain.Website, error)
}
