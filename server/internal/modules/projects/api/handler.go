package api

import (
	"context"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	"camopanel/server/internal/modules/projects/usecase"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

type TemplateCatalog interface {
	List() []projectsdomain.Template
	Get(id string) (projectsdomain.Template, error)
}

type CreateProjectUsecase interface {
	Execute(ctx context.Context, input usecase.CreateProjectInput) (usecase.CreateProjectOutput, error)
}

type CreateCustomProjectUsecase interface {
	Execute(ctx context.Context, input usecase.CreateCustomProjectInput) (usecase.CreateProjectOutput, error)
}

type ListProjectsUsecase interface {
	Execute(ctx context.Context) ([]usecase.ProjectView, error)
}

type GetProjectUsecase interface {
	Execute(ctx context.Context, projectID string) (usecase.ProjectView, error)
}

type RunActionUsecase interface {
	Execute(ctx context.Context, input usecase.RunActionInput) (usecase.RunActionOutput, error)
	Logs(ctx context.Context, projectID string, tail int) (string, error)
}

type Handler struct {
	templates     TemplateCatalog
	createProject CreateProjectUsecase
	createCustom  CreateCustomProjectUsecase
	listProjects  ListProjectsUsecase
	getProject    GetProjectUsecase
	runAction     RunActionUsecase
}

func NewHandler(templates TemplateCatalog, createProject CreateProjectUsecase, createCustom CreateCustomProjectUsecase, listProjects ListProjectsUsecase, getProject GetProjectUsecase, runAction RunActionUsecase) *Handler {
	return &Handler{
		templates:     templates,
		createProject: createProject,
		createCustom:  createCustom,
		listProjects:  listProjects,
		getProject:    getProject,
		runAction:     runAction,
	}
}

func (h *Handler) ListTemplates(c *gin.Context) {
	httpx.OK(c, gin.H{"items": h.templates.List()})
}

func (h *Handler) GetTemplate(c *gin.Context) {
	item, err := h.templates.Get(c.Param("id"))
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeNotFound, err.Error()))
		return
	}
	httpx.OK(c, item)
}

func (h *Handler) ListProjects(c *gin.Context) {
	items, err := h.listProjects.Execute(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "list projects", err))
		return
	}
	httpx.OK(c, gin.H{"items": items})
}

func (h *Handler) CreateProject(c *gin.Context) {
	var req struct {
		Name       string         `json:"name"`
		TemplateID string         `json:"template_id"`
		Parameters map[string]any `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.createProject.Execute(c.Request.Context(), usecase.CreateProjectInput{
		Name:       req.Name,
		TemplateID: req.TemplateID,
		Parameters: req.Parameters,
	})
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	c.JSON(201, gin.H{"project": toProjectResponse(usecase.ProjectView{
		ID:              item.Project.ID,
		Name:            item.Project.Name,
		Kind:            item.Project.Kind,
		TemplateID:      item.Project.TemplateID,
		TemplateVersion: item.Project.TemplateVersion,
		Config:          item.Project.Config,
		ComposePath:     item.Project.ComposePath,
		Status:          item.Project.Status,
		LastError:       item.Project.LastError,
		Runtime:         item.Runtime,
		CreatedAt:       item.Project.CreatedAt,
		UpdatedAt:       item.Project.UpdatedAt,
	})})
}

func (h *Handler) CreateCustomProject(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Compose string `json:"compose"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.createCustom.Execute(c.Request.Context(), usecase.CreateCustomProjectInput{
		Name:    req.Name,
		Compose: req.Compose,
	})
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	c.JSON(201, gin.H{"project": toProjectResponse(usecase.ProjectView{
		ID:              item.Project.ID,
		Name:            item.Project.Name,
		Kind:            item.Project.Kind,
		TemplateID:      item.Project.TemplateID,
		TemplateVersion: item.Project.TemplateVersion,
		Config:          item.Project.Config,
		ComposePath:     item.Project.ComposePath,
		Status:          item.Project.Status,
		LastError:       item.Project.LastError,
		Runtime:         item.Runtime,
		CreatedAt:       item.Project.CreatedAt,
		UpdatedAt:       item.Project.UpdatedAt,
	})})
}

func (h *Handler) GetProject(c *gin.Context) {
	item, err := h.getProject.Execute(c.Request.Context(), c.Param("id"))
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeNotFound, err.Error()))
		return
	}
	httpx.OK(c, toProjectResponse(item))
}

func (h *Handler) RunAction(c *gin.Context) {
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	result, err := h.runAction.Execute(c.Request.Context(), usecase.RunActionInput{
		ProjectID: c.Param("id"),
		Action:    req.Action,
	})
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	if result.Deleted {
		httpx.OK(c, gin.H{"deleted": true})
		return
	}
	httpx.OK(c, gin.H{"project": result.Project})
}

func (h *Handler) Logs(c *gin.Context) {
	logs, err := h.runAction.Logs(c.Request.Context(), c.Param("id"), 200)
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	httpx.OK(c, gin.H{"logs": logs})
}
