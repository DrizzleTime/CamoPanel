package bootstrap

import (
	"camopanel/server/internal/config"
	authapi "camopanel/server/internal/modules/auth/api"
	"camopanel/server/internal/modules/auth/domain"
	authrepo "camopanel/server/internal/modules/auth/repo"
	authusecase "camopanel/server/internal/modules/auth/usecase"
	copilotapi "camopanel/server/internal/modules/copilot/api"
	copilotrepo "camopanel/server/internal/modules/copilot/repo"
	copilotusecase "camopanel/server/internal/modules/copilot/usecase"
	databasesapi "camopanel/server/internal/modules/databases/api"
	databasesrepo "camopanel/server/internal/modules/databases/repo"
	databasesusecase "camopanel/server/internal/modules/databases/usecase"
	filesapi "camopanel/server/internal/modules/files/api"
	filesusecase "camopanel/server/internal/modules/files/usecase"
	projectsapi "camopanel/server/internal/modules/projects/api"
	projectsdomain "camopanel/server/internal/modules/projects/domain"
	projectsrepo "camopanel/server/internal/modules/projects/repo"
	projectsusecase "camopanel/server/internal/modules/projects/usecase"
	runtimeapi "camopanel/server/internal/modules/runtime/api"
	systemapi "camopanel/server/internal/modules/system/api"
	systemusecase "camopanel/server/internal/modules/system/usecase"
	websitesapi "camopanel/server/internal/modules/websites/api"
	websitesrepo "camopanel/server/internal/modules/websites/repo"
	websitesusecase "camopanel/server/internal/modules/websites/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformauth "camopanel/server/internal/platform/auth"
	platformdocker "camopanel/server/internal/platform/docker"
	platformfilesystem "camopanel/server/internal/platform/filesystem"
	platformopenresty "camopanel/server/internal/platform/openresty"
	platformsystem "camopanel/server/internal/platform/system"
	"camopanel/server/internal/services"
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	Name     string
	Register func(api *gin.RouterGroup)
}

func (m Module) RegisterRoutes(api *gin.RouterGroup) {
	if m.Register != nil {
		m.Register(api)
	}
}

type ModuleSet struct {
	Auth      authapi.Module
	Protected gin.HandlerFunc
	Projects  Module
	Websites  Module
	Databases Module
	Runtime   Module
	Files     Module
	System    Module
	Copilot   Module
}

type projectTemplateCatalog struct {
	catalog *services.TemplateCatalog
}

func (c projectTemplateCatalog) List() []projectsdomain.Template {
	specs := c.catalog.List()
	items := make([]projectsdomain.Template, 0, len(specs))
	for _, item := range specs {
		params := make([]projectsdomain.TemplateParam, 0, len(item.Params))
		for _, param := range item.Params {
			params = append(params, projectsdomain.TemplateParam{
				Name:        param.Name,
				Label:       param.Label,
				Description: param.Description,
				Type:        param.Type,
				Required:    param.Required,
				Default:     param.Default,
				Placeholder: param.Placeholder,
			})
		}
		items = append(items, projectsdomain.Template{
			ID:          item.ID,
			Name:        item.Name,
			Version:     item.Version,
			Description: item.Description,
			Params:      params,
			HealthHints: item.HealthHints,
		})
	}
	return items
}

func (c projectTemplateCatalog) Get(id string) (projectsdomain.Template, error) {
	item, err := c.catalog.Get(id)
	if err != nil {
		return projectsdomain.Template{}, err
	}

	params := make([]projectsdomain.TemplateParam, 0, len(item.Spec.Params))
	for _, param := range item.Spec.Params {
		params = append(params, projectsdomain.TemplateParam{
			Name:        param.Name,
			Label:       param.Label,
			Description: param.Description,
			Type:        param.Type,
			Required:    param.Required,
			Default:     param.Default,
			Placeholder: param.Placeholder,
		})
	}

	return projectsdomain.Template{
		ID:          item.Spec.ID,
		Name:        item.Spec.Name,
		Version:     item.Spec.Version,
		Description: item.Spec.Description,
		Params:      params,
		HealthHints: item.Spec.HealthHints,
	}, nil
}

func NewModules(cfg config.Config, db *gorm.DB) (ModuleSet, error) {
	userRepo := authrepo.NewUserRepository(db)
	sessions := platformauth.NewSessionManager(cfg.SessionSecret)
	auditService := platformaudit.NewService(platformaudit.NewGormRepository(db))
	loginUsecase := authusecase.NewLogin(userRepo, sessions, auditService)
	meUsecase := authusecase.NewMe(userRepo)
	authModule := authapi.NewModule(authapi.NewHandler(cfg.CookieName, sessions, loginUsecase, meUsecase))
	dockerService := platformdocker.NewService()
	runtimeModule := runtimeapi.NewModule(runtimeapi.NewHandler(dockerService))
	if err := db.AutoMigrate(&projectsrepo.ProjectRecord{}); err != nil {
		return ModuleSet{}, err
	}
	if err := db.AutoMigrate(&websitesrepo.CertificateRecord{}); err != nil {
		return ModuleSet{}, err
	}
	if err := db.AutoMigrate(&copilotrepo.ProviderRecord{}, &copilotrepo.ModelRecord{}); err != nil {
		return ModuleSet{}, err
	}
	templateCatalog, err := services.NewTemplateCatalog(cfg.TemplatesDir)
	if err != nil {
		return ModuleSet{}, err
	}
	projectRepo := projectsrepo.NewProjectRepository(db)
	openrestyService := platformopenresty.NewService(dockerService, cfg.OpenRestyContainer, cfg.OpenRestyDataDir)
	websiteRepo := websitesrepo.NewWebsiteRepository(cfg.OpenRestyDataDir)
	certificateRepo := websitesrepo.NewCertificateRepository(db)
	projectConfig := projectsusecase.ProjectConfig{
		ProjectsDir:        cfg.ProjectsDir,
		BridgeNetworkName:  cfg.BridgeNetworkName,
		OpenRestyContainer: cfg.OpenRestyContainer,
		OpenRestyDataDir:   cfg.OpenRestyDataDir,
	}
	projectsModule := projectsapi.NewModule(projectsapi.NewHandler(
		projectTemplateCatalog{catalog: templateCatalog},
		projectsusecase.NewCreateProject(projectRepo, templateCatalog, dockerService, auditService, projectConfig),
		projectsusecase.NewCreateCustomProject(projectRepo, templateCatalog, dockerService, auditService, projectConfig),
		projectsusecase.NewListProjects(projectRepo, dockerService),
		projectsusecase.NewGetProject(projectRepo, dockerService),
		projectsusecase.NewRunAction(projectRepo, projectsusecase.NewRuntimeActionRunner(projectRepo, dockerService, auditService, projectConfig)),
	))
	listProjects := projectsusecase.NewListProjects(projectRepo, dockerService)
	getProject := projectsusecase.NewGetProject(projectRepo, dockerService)
	runProjectAction := projectsusecase.NewRunAction(projectRepo, projectsusecase.NewRuntimeActionRunner(projectRepo, dockerService, auditService, projectConfig))
	websiteConfig := websitesusecase.WebsiteConfig{OpenRestyDataDir: cfg.OpenRestyDataDir}
	listWebsites := websitesusecase.NewListWebsites(websiteRepo)
	websitesModule := websitesapi.NewModule(websitesapi.NewHandler(
		openrestyService,
		listWebsites,
		websitesusecase.NewCreateWebsite(websiteRepo, certificateRepo, projectRepo, openrestyService, auditService, websiteConfig),
		websitesusecase.NewUpdateWebsite(websiteRepo, certificateRepo, projectRepo, openrestyService, auditService, websiteConfig),
		websitesusecase.NewDeleteWebsite(websiteRepo, openrestyService, auditService),
		websitesusecase.NewPreviewConfig(websiteRepo, openrestyService),
		websitesusecase.NewListCertificates(websiteRepo, certificateRepo),
		websitesusecase.NewIssueCertificate(websiteRepo, certificateRepo, openrestyService, auditService),
		websitesusecase.NewDeleteCertificate(websiteRepo, certificateRepo, openrestyService, auditService),
	))
	databaseService := databasesusecase.NewService(
		databasesrepo.NewRepository(projectRepo),
		dockerService,
		dockerService,
		templateCatalog,
		auditService,
		databasesusecase.Config{
			BridgeNetworkName:  cfg.BridgeNetworkName,
			OpenRestyContainer: cfg.OpenRestyContainer,
			OpenRestyDataDir:   cfg.OpenRestyDataDir,
		},
	)
	databasesModule := databasesapi.NewModule(databasesapi.NewHandler(databaseService))
	fileSystemService := platformfilesystem.NewService()
	filesModule := filesapi.NewModule(filesapi.NewHandler(filesusecase.NewService(fileSystemService)))
	hostService := services.NewHostService(cfg.DataDir)
	hostControlService := services.NewHostControlService(cfg.HostControlHelper)
	systemPlatformService := platformsystem.NewService(hostService, hostControlService, dockerService)
	systemModule := systemapi.NewModule(systemapi.NewHandler(systemPlatformService, systemusecase.NewDashboard(systemPlatformService, listProjects, listWebsites)))
	copilotRepo := copilotrepo.NewRepository(db)
	fallbackCopilot := services.CopilotRuntimeConfig{
		Source:       "env",
		ProviderName: "环境变量",
		Model:        cfg.AI.Model,
		BaseURL:      cfg.AI.BaseURL,
		APIKey:       cfg.AI.APIKey,
	}
	copilotConfigService := copilotusecase.NewService(copilotRepo, fallbackCopilot)
	copilotChatService := services.NewCopilotService(cfg.AI, copilotToolbox{
		templates:    templateCatalog,
		listProjects: listProjects,
		getProject:   getProject,
		runAction:    runProjectAction,
		host:         systemPlatformService,
	}, copilotConfigService)
	copilotModule := copilotapi.NewModule(copilotapi.NewHandler(copilotChatService, copilotConfigService))
	loadSubject := func(c *gin.Context, userID string) (any, error) {
		user, err := meUsecase.Execute(c.Request.Context(), userID)
		if err != nil {
			if err == domain.ErrUserNotFound {
				return nil, platformauth.ErrSubjectNotFound
			}
			return nil, err
		}
		return user, nil
	}

	return ModuleSet{
		Auth:      authModule,
		Protected: platformauth.RequireAuth(cfg.CookieName, sessions, loadSubject),
		Projects: Module{
			Name:     "projects",
			Register: projectsModule.RegisterRoutes,
		},
		Websites: Module{
			Name:     "websites",
			Register: websitesModule.RegisterRoutes,
		},
		Databases: Module{
			Name:     "databases",
			Register: databasesModule.RegisterRoutes,
		},
		Runtime: Module{
			Name:     "runtime",
			Register: runtimeModule.RegisterRoutes,
		},
		Files: Module{
			Name:     "files",
			Register: filesModule.RegisterRoutes,
		},
		System: Module{
			Name:     "system",
			Register: systemModule.RegisterRoutes,
		},
		Copilot: Module{
			Name:     "copilot",
			Register: copilotModule.RegisterRoutes,
		},
	}, nil
}

type copilotToolbox struct {
	templates    *services.TemplateCatalog
	listProjects *projectsusecase.ListProjects
	getProject   *projectsusecase.GetProject
	runAction    *projectsusecase.RunAction
	host         *platformsystem.Service
}

func (c copilotToolbox) ListTemplates() []services.TemplateSpec {
	return c.templates.List()
}

func (c copilotToolbox) GetTemplate(templateID string) (*services.LoadedTemplate, error) {
	return c.templates.Get(templateID)
}

func (c copilotToolbox) ListProjects(ctx context.Context) ([]services.ProjectToolData, error) {
	items, err := c.listProjects.Execute(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]services.ProjectToolData, 0, len(items))
	for _, item := range items {
		result = append(result, services.ProjectToolData{
			ID:              item.ID,
			Name:            item.Name,
			TemplateID:      item.TemplateID,
			TemplateVersion: item.TemplateVersion,
			Status:          item.Status,
			LastError:       item.LastError,
			Containers:      toServiceContainers(item.Runtime.Containers),
		})
	}
	return result, nil
}

func (c copilotToolbox) GetProject(ctx context.Context, projectID string) (services.ProjectToolData, error) {
	item, err := c.getProject.Execute(ctx, projectID)
	if err != nil {
		return services.ProjectToolData{}, err
	}
	return services.ProjectToolData{
		ID:              item.ID,
		Name:            item.Name,
		TemplateID:      item.TemplateID,
		TemplateVersion: item.TemplateVersion,
		Status:          item.Status,
		LastError:       item.LastError,
		Containers:      toServiceContainers(item.Runtime.Containers),
	}, nil
}

func (c copilotToolbox) GetProjectLogs(ctx context.Context, projectID string, tail int) (string, error) {
	return c.runAction.Logs(ctx, projectID, tail)
}

func (c copilotToolbox) GetHostSummary(ctx context.Context) (services.HostSummary, error) {
	return c.host.Summary(ctx)
}

func toServiceContainers(items []platformdocker.ProjectContainer) []services.ProjectContainer {
	result := make([]services.ProjectContainer, 0, len(items))
	for _, item := range items {
		result = append(result, services.ProjectContainer{
			ID:     item.ID,
			Name:   item.Name,
			Image:  item.Image,
			State:  item.State,
			Status: item.Status,
			Ports:  append([]string(nil), item.Ports...),
		})
	}
	return result
}
