package api

import "github.com/gin-gonic/gin"

type Module struct {
	handler *Handler
}

func NewModule(handler *Handler) Module {
	return Module{handler: handler}
}

func (m Module) RegisterRoutes(api *gin.RouterGroup) {
	if m.handler == nil {
		return
	}
	api.GET("/templates", m.handler.ListTemplates)
	api.GET("/templates/:id", m.handler.GetTemplate)
	api.GET("/projects", m.handler.ListProjects)
	api.POST("/projects", m.handler.CreateProject)
	api.POST("/projects/custom", m.handler.CreateCustomProject)
	api.GET("/projects/:id", m.handler.GetProject)
	api.POST("/projects/:id/actions", m.handler.RunAction)
	api.GET("/projects/:id/logs", m.handler.Logs)
}
