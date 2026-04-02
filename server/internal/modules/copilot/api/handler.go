package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	copilotdomain "camopanel/server/internal/modules/copilot/domain"
	copilotusecase "camopanel/server/internal/modules/copilot/usecase"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type ChatService interface {
	CreateSession() services.CopilotSession
	Reply(ctx context.Context, sessionID, userMessage string) (services.CopilotReply, error)
}

type ConfigService interface {
	ListProviders(ctx context.Context) ([]copilotusecase.ProviderView, error)
	CreateProvider(ctx context.Context, name, providerType, baseURL, apiKey string, enabled bool) (copilotusecase.ProviderView, error)
	UpdateProvider(ctx context.Context, providerID, name, providerType, baseURL, apiKey string, enabled bool) (copilotusecase.ProviderView, error)
	DeleteProvider(ctx context.Context, providerID string) error
	CreateModel(ctx context.Context, providerID, name string, enabled, isDefault bool) (copilotusecase.ModelView, error)
	UpdateModel(ctx context.Context, modelID, name string, enabled, isDefault bool) (copilotusecase.ModelView, error)
	DeleteModel(ctx context.Context, modelID string) error
	ConfigStatus(ctx context.Context) (copilotusecase.ConfigStatus, error)
}

type Handler struct {
	chat   ChatService
	config ConfigService
}

func NewHandler(chat ChatService, config ConfigService) *Handler {
	return &Handler{chat: chat, config: config}
}

func (h *Handler) CreateSession(c *gin.Context) {
	c.JSON(http.StatusCreated, h.chat.CreateSession())
}

func (h *Handler) Message(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	reply, err := h.chat.Reply(c.Request.Context(), c.Param("id"), req.Message)
	if err != nil {
		writeSSE(c, "error", gin.H{"error": err.Error()})
		return
	}
	writeSSE(c, "chunk", gin.H{"content": reply.Message})
	writeSSE(c, "done", gin.H{"message": reply.Message})
}

func (h *Handler) ConfigStatus(c *gin.Context) {
	item, err := h.config.ConfigStatus(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "load copilot config", err))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) ListProviders(c *gin.Context) {
	items, err := h.config.ListProviders(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "list providers", err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) CreateProvider(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.config.CreateProvider(c.Request.Context(), req.Name, req.Type, req.BaseURL, req.APIKey, req.Enabled)
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"provider": item})
}

func (h *Handler) UpdateProvider(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.config.UpdateProvider(c.Request.Context(), c.Param("id"), req.Name, req.Type, req.BaseURL, req.APIKey, req.Enabled)
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"provider": item})
}

func (h *Handler) DeleteProvider(c *gin.Context) {
	if err := h.config.DeleteProvider(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *Handler) CreateModel(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.config.CreateModel(c.Request.Context(), c.Param("id"), req.Name, req.Enabled, req.IsDefault)
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"model": item})
}

func (h *Handler) UpdateModel(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		Enabled   bool   `json:"enabled"`
		IsDefault bool   `json:"is_default"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.config.UpdateModel(c.Request.Context(), c.Param("id"), req.Name, req.Enabled, req.IsDefault)
	if err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"model": item})
}

func (h *Handler) DeleteModel(c *gin.Context) {
	if err := h.config.DeleteModel(c.Request.Context(), c.Param("id")); err != nil {
		httpx.ErrorFrom(c, mapError(err))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func writeSSE(c *gin.Context, event string, payload any) {
	raw, _ := json.Marshal(payload)
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(raw) + "\n\n")
	c.Writer.Flush()
}

func mapError(err error) error {
	switch {
	case errors.Is(err, copilotdomain.ErrProviderNotFound):
		return errs.E(errs.CodeNotFound, "模型服务不存在")
	case errors.Is(err, copilotdomain.ErrModelNotFound):
		return errs.E(errs.CodeNotFound, "模型不存在")
	default:
		return errs.E(errs.CodeInvalidArgument, err.Error())
	}
}
