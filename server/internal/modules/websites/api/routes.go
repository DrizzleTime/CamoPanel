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
	api.GET("/openresty/status", m.handler.OpenRestyStatus)
	api.GET("/websites", m.handler.ListWebsites)
	api.POST("/websites", m.handler.CreateWebsite)
	api.PUT("/websites/:id", m.handler.UpdateWebsite)
	api.DELETE("/websites/:id", m.handler.DeleteWebsite)
	api.GET("/websites/:id/config-preview", m.handler.PreviewConfig)
	api.GET("/certificates", m.handler.ListCertificates)
	api.POST("/certificates", m.handler.CreateCertificate)
	api.DELETE("/certificates/:id", m.handler.DeleteCertificate)
}
