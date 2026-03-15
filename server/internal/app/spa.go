package app

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"camopanel/server/internal/webui"

	"github.com/gin-gonic/gin"
)

func (a *App) handleSPA(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	filePath := strings.TrimPrefix(path.Clean(c.Request.URL.Path), "/")
	if filePath == "" || filePath == "." {
		filePath = "index.html"
	}

	distFS, err := fs.Sub(webui.DistFS, "dist")
	if err != nil {
		c.String(http.StatusInternalServerError, "web ui is unavailable")
		return
	}

	if _, err := fs.Stat(distFS, filePath); err != nil {
		filePath = "index.html"
	}
	c.FileFromFS(filePath, http.FS(distFS))
}
