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
	api.GET("/docker/containers", m.handler.ListContainers)
	api.GET("/docker/containers/:id/logs", m.handler.ContainerLogs)
	api.POST("/docker/containers/:id/actions", m.handler.ContainerAction)
	api.GET("/docker/images", m.handler.ListImages)
	api.DELETE("/docker/images/:id", m.handler.DeleteImage)
	api.POST("/docker/images/prune", m.handler.PruneImages)
	api.GET("/docker/networks", m.handler.ListNetworks)
}
