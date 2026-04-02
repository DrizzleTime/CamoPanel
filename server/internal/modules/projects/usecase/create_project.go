package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/services"

	"github.com/google/uuid"
)

const (
	ManagedOpenRestyTemplateID = "openresty"
	ManagedOpenRestyProjectID  = "openresty"
)

var projectNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

type ProjectRepository interface {
	List(ctx context.Context) ([]projectsdomain.Project, error)
	FindByID(ctx context.Context, projectID string) (projectsdomain.Project, error)
	FindByName(ctx context.Context, name string) (projectsdomain.Project, error)
	CountByTemplateID(ctx context.Context, templateID string) (int64, error)
	Create(ctx context.Context, project projectsdomain.Project) error
	Save(ctx context.Context, project projectsdomain.Project) error
	Delete(ctx context.Context, projectID string) error
}

type Runtime interface {
	platformdocker.Runtime
}

type TemplateCatalog interface {
	Get(id string) (*services.LoadedTemplate, error)
}

type AuditRecorder interface {
	Record(ctx context.Context, entry platformaudit.Entry) error
}

type ProjectConfig struct {
	ProjectsDir        string
	BridgeNetworkName  string
	OpenRestyContainer string
	OpenRestyDataDir   string
}

type CreateProjectInput struct {
	ActorID    string
	Name       string
	TemplateID string
	Parameters map[string]any
}

type CreateProjectOutput struct {
	Project projectsdomain.Project
	Runtime platformdocker.ProjectRuntime
}

type CreateProject struct {
	projects  ProjectRepository
	templates TemplateCatalog
	runtime   Runtime
	audit     AuditRecorder
	cfg       ProjectConfig
}

func NewCreateProject(projects ProjectRepository, templates TemplateCatalog, runtime Runtime, audit AuditRecorder, cfg ProjectConfig) *CreateProject {
	return &CreateProject{
		projects:  projects,
		templates: templates,
		runtime:   runtime,
		audit:     audit,
		cfg:       cfg,
	}
}

func (u *CreateProject) Execute(ctx context.Context, input CreateProjectInput) (CreateProjectOutput, error) {
	normalizedName := normalizeProjectName(input.Name)
	if input.TemplateID == ManagedOpenRestyTemplateID {
		normalizedName = ManagedOpenRestyProjectID
	}
	if !projectNamePattern.MatchString(normalizedName) {
		return CreateProjectOutput{}, fmt.Errorf("项目名只能包含小写字母、数字、下划线和中划线")
	}

	if _, err := u.projects.FindByName(ctx, normalizedName); err == nil {
		return CreateProjectOutput{}, fmt.Errorf("项目名已存在")
	} else if !errors.Is(err, projectsdomain.ErrProjectNotFound) {
		return CreateProjectOutput{}, err
	}

	if input.TemplateID == ManagedOpenRestyTemplateID {
		count, err := u.projects.CountByTemplateID(ctx, ManagedOpenRestyTemplateID)
		if err != nil {
			return CreateProjectOutput{}, err
		}
		if count > 0 {
			return CreateProjectOutput{}, fmt.Errorf("固定 OpenResty 已存在，不支持重复部署")
		}
	}

	templateItem, err := u.templates.Get(input.TemplateID)
	if err != nil {
		return CreateProjectOutput{}, err
	}
	normalized, err := templateItem.ValidateAndNormalize(input.Parameters)
	if err != nil {
		return CreateProjectOutput{}, err
	}
	rendered, err := templateItem.Render(normalized, templateRuntime(u.cfg, normalizedName))
	if err != nil {
		return CreateProjectOutput{}, err
	}

	projectID := uuid.NewString()
	projectDir := filepath.Join(u.cfg.ProjectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return CreateProjectOutput{}, err
	}

	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(rendered), 0o644); err != nil {
		return CreateProjectOutput{}, err
	}

	if input.TemplateID != ManagedOpenRestyTemplateID {
		if err := u.runtime.EnsureNetwork(ctx, u.cfg.BridgeNetworkName, "bridge"); err != nil {
			return CreateProjectOutput{}, err
		}
	}
	if err := u.runtime.Deploy(ctx, normalizedName, composePath); err != nil {
		return CreateProjectOutput{}, err
	}

	project := projectsdomain.Project{
		ID:              projectID,
		Name:            normalizedName,
		Kind:            projectsdomain.KindTemplate,
		TemplateID:      templateItem.Spec.ID,
		TemplateVersion: templateItem.Spec.Version,
		Config:          normalized,
		ComposePath:     composePath,
		Status:          platformdocker.StatusRunning,
	}

	runtimeInfo, err := u.runtime.InspectProject(ctx, normalizedName)
	if err == nil && runtimeInfo.Status != "" {
		project.Status = runtimeInfo.Status
	} else {
		runtimeInfo = platformdocker.ProjectRuntime{Status: project.Status}
	}

	if err := u.projects.Create(ctx, project); err != nil {
		return CreateProjectOutput{}, err
	}

	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "project_deploy",
		TargetType: "project",
		TargetID:   project.ID,
		Metadata:   map[string]any{"name": project.Name, "template_id": project.TemplateID},
	})

	return CreateProjectOutput{Project: project, Runtime: runtimeInfo}, nil
}

func normalizeProjectName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
