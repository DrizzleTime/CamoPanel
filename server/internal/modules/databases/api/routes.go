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
	api.GET("/databases", m.handler.ListInstances)
	api.GET("/databases/:id/overview", m.handler.Overview)
	api.POST("/databases/:id/databases", m.handler.CreateDatabase)
	api.DELETE("/databases/:id/databases/:name", m.handler.DeleteDatabase)
	api.POST("/databases/:id/accounts", m.handler.CreateAccount)
	api.DELETE("/databases/:id/accounts/:account", m.handler.DeleteAccount)
	api.POST("/databases/:id/accounts/:account/password", m.handler.UpdateAccountPassword)
	api.POST("/databases/:id/grants", m.handler.GrantAccount)
	api.POST("/databases/:id/redis/config", m.handler.UpdateRedisConfig)
}
