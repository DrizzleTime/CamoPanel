package app

import (
	"fmt"
	"net/http"
	"os"

	"camopanel/server/internal/config"
	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type App struct {
	cfg         config.Config
	db          *gorm.DB
	auth        *services.AuthService
	templates   *services.TemplateCatalog
	executor    services.Executor
	docker      services.DockerReader
	containers  services.ContainerOperator
	host        *services.HostService
	hostControl *services.HostControlService
	openresty   services.OpenRestyManager
	copilot     *services.CopilotService
}

func New(cfg config.Config) (*App, error) {
	if err := os.MkdirAll(cfg.ProjectsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create projects dir: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(cfg.DatabasePath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Project{}, &model.Certificate{}, &model.AuditEvent{}, &model.AIProvider{}, &model.AIModel{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	if err := cleanupLegacyApprovalData(db); err != nil {
		return nil, fmt.Errorf("cleanup legacy approval data: %w", err)
	}

	auth := services.NewAuthService(cfg.SessionSecret)
	if err := seedAdmin(db, cfg, auth); err != nil {
		return nil, err
	}

	catalog, err := services.NewTemplateCatalog(cfg.TemplatesDir)
	if err != nil {
		return nil, fmt.Errorf("load templates: %w", err)
	}

	dockerService := services.NewDockerService()

	instance := &App{
		cfg:         cfg,
		db:          db,
		auth:        auth,
		templates:   catalog,
		executor:    dockerService,
		docker:      dockerService,
		containers:  dockerService,
		host:        services.NewHostService(cfg.DataDir),
		hostControl: services.NewHostControlService(cfg.HostControlHelper),
		openresty:   services.NewOpenRestyService(dockerService, cfg.OpenRestyContainer, cfg.OpenRestyDataDir),
	}
	instance.copilot = services.NewCopilotService(cfg.AI, instance, instance)

	return instance, nil
}

func (a *App) Run() error {
	return a.router().Run(a.cfg.HTTPAddr)
}

func (a *App) router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.POST("/api/auth/login", a.handleLogin)
	router.POST("/api/auth/logout", a.handleLogout)

	api := router.Group("/api")
	api.Use(a.authMiddleware())
	{
		api.GET("/auth/me", a.handleMe)
		api.GET("/templates", a.handleTemplates)
		api.GET("/templates/:id", a.handleTemplate)
		api.GET("/openresty/status", a.handleOpenRestyStatus)
		api.GET("/projects", a.handleProjects)
		api.POST("/projects", a.handleCreateProject)
		api.POST("/projects/custom", a.handleCreateCustomProject)
		api.GET("/docker/containers", a.handleDockerContainers)
		api.GET("/docker/containers/:id/logs", a.handleDockerContainerLogs)
		api.POST("/docker/containers/:id/actions", a.handleDockerContainerAction)
		api.GET("/docker/images", a.handleDockerImages)
		api.DELETE("/docker/images/:id", a.handleDockerImageDelete)
		api.POST("/docker/images/prune", a.handleDockerImagePrune)
		api.GET("/docker/networks", a.handleDockerNetworks)
		api.GET("/docker/system", a.handleDockerSystem)
		api.GET("/docker/settings", a.handleDockerSettings)
		api.PUT("/docker/settings", a.handleUpdateDockerSettings)
		api.POST("/docker/restart", a.handleRestartDockerDaemon)
		api.GET("/websites", a.handleWebsites)
		api.POST("/websites", a.handleCreateWebsite)
		api.GET("/certificates", a.handleCertificates)
		api.POST("/certificates", a.handleCreateCertificate)
		api.DELETE("/certificates/:id", a.handleDeleteCertificate)
		api.PUT("/websites/:id", a.handleUpdateWebsite)
		api.DELETE("/websites/:id", a.handleDeleteWebsite)
		api.GET("/websites/:id/config-preview", a.handleWebsiteConfigPreview)
		api.GET("/projects/:id", a.handleProject)
		api.POST("/projects/:id/actions", a.handleProjectAction)
		api.GET("/projects/:id/logs", a.handleProjectLogs)
		api.GET("/databases", a.handleDatabaseInstances)
		api.GET("/databases/:id/overview", a.handleDatabaseOverview)
		api.POST("/databases/:id/databases", a.handleCreateManagedDatabase)
		api.DELETE("/databases/:id/databases/:name", a.handleDeleteManagedDatabase)
		api.POST("/databases/:id/accounts", a.handleCreateDatabaseAccount)
		api.DELETE("/databases/:id/accounts/:account", a.handleDeleteDatabaseAccount)
		api.POST("/databases/:id/accounts/:account/password", a.handleUpdateDatabaseAccountPassword)
		api.POST("/databases/:id/grants", a.handleGrantDatabaseAccount)
		api.POST("/databases/:id/redis/config", a.handleUpdateRedisConfig)
		api.GET("/host/summary", a.handleHostSummary)
		api.GET("/host/metrics", a.handleHostMetrics)
		api.GET("/dashboard/stream", a.handleDashboardStream)
		api.POST("/copilot/sessions", a.handleCreateCopilotSession)
		api.POST("/copilot/sessions/:id/messages", a.handleCopilotMessage)
		api.GET("/copilot/config", a.handleCopilotConfig)
		api.GET("/copilot/providers", a.handleCopilotProviders)
		api.POST("/copilot/providers", a.handleCreateCopilotProvider)
		api.PUT("/copilot/providers/:id", a.handleUpdateCopilotProvider)
		api.DELETE("/copilot/providers/:id", a.handleDeleteCopilotProvider)
		api.POST("/copilot/providers/:id/models", a.handleCreateCopilotModel)
		api.PUT("/copilot/models/:id", a.handleUpdateCopilotModel)
		api.DELETE("/copilot/models/:id", a.handleDeleteCopilotModel)
		api.GET("/files/list", a.handleFileList)
		api.GET("/files/read", a.handleFileRead)
		api.POST("/files/write", a.handleFileWrite)
		api.POST("/files/create", a.handleFileCreate)
		api.POST("/files/mkdir", a.handleFileMkdir)
		api.POST("/files/move", a.handleFileMove)
		api.POST("/files/delete", a.handleFileDelete)
		api.POST("/files/upload", a.handleFileUpload)
		api.GET("/files/download", a.handleFileDownload)
	}

	router.NoRoute(a.handleSPA)

	return router
}
