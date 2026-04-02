package bootstrap

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	authapi "camopanel/server/internal/modules/auth/api"
	"camopanel/server/internal/webui"

	"github.com/gin-gonic/gin"
)

func NewRouter(deps ModuleSet) *gin.Engine {
	r := gin.New()
	r.Use(DefaultMiddlewares()...)

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	registerAuthRoutes(r, deps.Auth)
	api := r.Group("/api")
	if deps.Protected != nil {
		api.Use(deps.Protected)
	}
	registerProjectRoutes(api, deps.Projects)
	registerWebsiteRoutes(api, deps.Websites)
	registerDatabaseRoutes(api, deps.Databases)
	registerRuntimeRoutes(api, deps.Runtime)
	registerFileRoutes(api, deps.Files)
	registerSystemRoutes(api, deps.System)
	registerCopilotRoutes(api, deps.Copilot)

	r.NoRoute(handleSPA)
	r.NoMethod(handleSPA)
	return r
}

func registerAuthRoutes(r *gin.Engine, module authapi.Module)    { module.RegisterRoutes(r) }
func registerProjectRoutes(api *gin.RouterGroup, module Module)  { module.RegisterRoutes(api) }
func registerWebsiteRoutes(api *gin.RouterGroup, module Module)  { module.RegisterRoutes(api) }
func registerDatabaseRoutes(api *gin.RouterGroup, module Module) { module.RegisterRoutes(api) }
func registerRuntimeRoutes(api *gin.RouterGroup, module Module)  { module.RegisterRoutes(api) }
func registerFileRoutes(api *gin.RouterGroup, module Module)     { module.RegisterRoutes(api) }
func registerSystemRoutes(api *gin.RouterGroup, module Module)   { module.RegisterRoutes(api) }
func registerCopilotRoutes(api *gin.RouterGroup, module Module)  { module.RegisterRoutes(api) }

func handleSPA(c *gin.Context) {
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
