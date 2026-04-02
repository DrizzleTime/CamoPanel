package api

import (
	"context"
	"errors"

	databasesdomain "camopanel/server/internal/modules/databases/domain"
	"camopanel/server/internal/platform/docker"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type Service interface {
	ListInstances(ctx context.Context, engine string) ([]databasesdomain.InstanceView, error)
	GetOverview(ctx context.Context, instanceID string) (databasesdomain.Overview, error)
	CreateDatabase(ctx context.Context, actorID, instanceID, name string) error
	DeleteDatabase(ctx context.Context, actorID, instanceID, name string) error
	CreateAccount(ctx context.Context, actorID, instanceID, name, password, databaseName string) error
	DeleteAccount(ctx context.Context, actorID, instanceID, accountName string) error
	UpdateAccountPassword(ctx context.Context, actorID, instanceID, accountName, password string) error
	GrantAccount(ctx context.Context, actorID, instanceID, accountName, databaseName string) error
	UpdateRedisConfig(ctx context.Context, actorID, instanceID, key string, value any) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) ListInstances(c *gin.Context) {
	items, err := h.service.ListInstances(c.Request.Context(), c.Query("engine"))
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) Overview(c *gin.Context) {
	item, err := h.service.GetOverview(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) CreateDatabase(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.service.CreateDatabase(c.Request.Context(), "", c.Param("id"), req.Name); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) DeleteDatabase(c *gin.Context) {
	if err := h.service.DeleteDatabase(c.Request.Context(), "", c.Param("id"), c.Param("name")); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) CreateAccount(c *gin.Context) {
	var req struct {
		Name         string `json:"name"`
		Password     string `json:"password"`
		DatabaseName string `json:"database_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.service.CreateAccount(c.Request.Context(), "", c.Param("id"), req.Name, req.Password, req.DatabaseName); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	if err := h.service.DeleteAccount(c.Request.Context(), "", c.Param("id"), c.Param("account")); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) UpdateAccountPassword(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.service.UpdateAccountPassword(c.Request.Context(), "", c.Param("id"), c.Param("account"), req.Password); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) GrantAccount(c *gin.Context) {
	var req struct {
		AccountName  string `json:"account_name"`
		DatabaseName string `json:"database_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.service.GrantAccount(c.Request.Context(), "", c.Param("id"), req.AccountName, req.DatabaseName); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func (h *Handler) UpdateRedisConfig(c *gin.Context) {
	var req struct {
		Key   string `json:"key"`
		Value any    `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	if err := h.service.UpdateRedisConfig(c.Request.Context(), "", c.Param("id"), req.Key, req.Value); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"ok": true})
}

func mapError(err error) error {
	switch {
	case errors.Is(err, databasesdomain.ErrInstanceNotFound):
		return errs.E(errs.CodeNotFound, "数据库实例不存在")
	case errors.Is(err, docker.ErrUnavailable):
		return errs.E(errs.CodeUnavailable, "Docker 当前不可用")
	default:
		return errs.E(errs.CodeInvalidArgument, err.Error())
	}
}
