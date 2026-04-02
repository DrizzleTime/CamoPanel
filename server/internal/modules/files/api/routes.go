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
	api.GET("/files/list", m.handler.List)
	api.GET("/files/read", m.handler.Read)
	api.POST("/files/write", m.handler.Write)
	api.POST("/files/create", m.handler.Create)
	api.POST("/files/mkdir", m.handler.Mkdir)
	api.POST("/files/move", m.handler.Move)
	api.POST("/files/delete", m.handler.Delete)
	api.POST("/files/upload", m.handler.Upload)
	api.GET("/files/download", m.handler.Download)
}
