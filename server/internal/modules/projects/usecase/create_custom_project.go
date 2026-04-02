package usecase

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"

	"github.com/google/uuid"
)

const (
	CustomComposeTemplateID      = "custom-compose"
	CustomComposeTemplateVersion = "1"
)

type CreateCustomProjectInput struct {
	ActorID string
	Name    string
	Compose string
}

type CreateCustomProject struct {
	projects  ProjectRepository
	templates TemplateCatalog
	runtime   Runtime
	audit     AuditRecorder
	cfg       ProjectConfig
}

func NewCreateCustomProject(projects ProjectRepository, templates TemplateCatalog, runtime Runtime, audit AuditRecorder, cfg ProjectConfig) *CreateCustomProject {
	return &CreateCustomProject{
		projects:  projects,
		templates: templates,
		runtime:   runtime,
		audit:     audit,
		cfg:       cfg,
	}
}

func (u *CreateCustomProject) Execute(ctx context.Context, input CreateCustomProjectInput) (CreateProjectOutput, error) {
	normalizedName := normalizeProjectName(input.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return CreateProjectOutput{}, fmt.Errorf("项目名只能包含小写字母、数字、下划线和中划线")
	}

	if _, err := u.projects.FindByName(ctx, normalizedName); err == nil {
		return CreateProjectOutput{}, fmt.Errorf("项目名已存在")
	} else if !errors.Is(err, projectsdomain.ErrProjectNotFound) {
		return CreateProjectOutput{}, err
	}

	composeContent := strings.TrimSpace(input.Compose)
	if composeContent == "" {
		return CreateProjectOutput{}, fmt.Errorf("Compose 内容不能为空")
	}
	composeContent += "\n"

	projectID := uuid.NewString()
	projectDir := filepath.Join(u.cfg.ProjectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return CreateProjectOutput{}, err
	}

	composePath := filepath.Join(projectDir, "compose.yaml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0o644); err != nil {
		return CreateProjectOutput{}, err
	}
	if err := u.runtime.Deploy(ctx, normalizedName, composePath); err != nil {
		return CreateProjectOutput{}, err
	}

	project := projectsdomain.Project{
		ID:              projectID,
		Name:            normalizedName,
		Kind:            projectsdomain.KindCustom,
		TemplateID:      CustomComposeTemplateID,
		TemplateVersion: CustomComposeTemplateVersion,
		Config:          map[string]any{"mode": CustomComposeTemplateID},
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
