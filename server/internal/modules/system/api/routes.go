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
	api.GET("/host/summary", m.handler.HostSummary)
	api.GET("/host/metrics", m.handler.HostMetrics)
	api.GET("/dashboard/stream", m.handler.DashboardStream)
	api.GET("/docker/system", m.handler.DockerSystem)
	api.GET("/docker/settings", m.handler.DockerSettings)
	api.PUT("/docker/settings", m.handler.UpdateDockerSettings)
	api.POST("/docker/restart", m.handler.RestartDocker)
}
