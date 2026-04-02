package api

import (
	platformauth "camopanel/server/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

type Module struct {
	handler *Handler
}

func NewModule(handler *Handler) Module {
	return Module{handler: handler}
}

func (m Module) RegisterRoutes(r *gin.Engine) {
	if m.handler == nil {
		return
	}
	RegisterRoutes(r, m.handler)
}

func RegisterRoutes(r *gin.Engine, handler *Handler) {
	r.POST("/api/auth/login", handler.Login)
	r.POST("/api/auth/logout", handler.Logout)

	api := r.Group("/api")
	api.Use(platformauth.RequireAuth(handler.cookieName, handler.sessions, handler.loadSubject))
	api.GET("/auth/me", handler.Me)
}
