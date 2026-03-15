package app

import (
	"net/http"

	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

func (a *App) handleTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": a.templates.List()})
}

func (a *App) handleTemplate(c *gin.Context) {
	templateItem, err := a.templates.Get(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, templateItem.Spec)
}

func (a *App) ListTemplates() []services.TemplateSpec {
	return a.templates.List()
}

func (a *App) GetTemplate(templateID string) (*services.LoadedTemplate, error) {
	return a.templates.Get(templateID)
}
