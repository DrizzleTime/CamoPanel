package api

import "github.com/gin-gonic/gin"

type Module struct {
	handler *Handler
}

func NewModule(handler *Handler) Module { return Module{handler: handler} }

func (m Module) RegisterRoutes(api *gin.RouterGroup) {
	if m.handler == nil {
		return
	}
	api.POST("/copilot/sessions", m.handler.CreateSession)
	api.POST("/copilot/sessions/:id/messages", m.handler.Message)
	api.GET("/copilot/config", m.handler.ConfigStatus)
	api.GET("/copilot/providers", m.handler.ListProviders)
	api.POST("/copilot/providers", m.handler.CreateProvider)
	api.PUT("/copilot/providers/:id", m.handler.UpdateProvider)
	api.DELETE("/copilot/providers/:id", m.handler.DeleteProvider)
	api.POST("/copilot/providers/:id/models", m.handler.CreateModel)
	api.PUT("/copilot/models/:id", m.handler.UpdateModel)
	api.DELETE("/copilot/models/:id", m.handler.DeleteModel)
}
