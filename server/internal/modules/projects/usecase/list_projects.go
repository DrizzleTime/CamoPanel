package usecase

import (
	"context"
	"errors"
	"time"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	platformdocker "camopanel/server/internal/platform/docker"
)

type ProjectView struct {
	ID              string                        `json:"id"`
	Name            string                        `json:"name"`
	Kind            string                        `json:"kind"`
	TemplateID      string                        `json:"template_id"`
	TemplateVersion string                        `json:"template_version"`
	Config          map[string]any                `json:"config"`
	ComposePath     string                        `json:"compose_path"`
	Status          string                        `json:"status"`
	LastError       string                        `json:"last_error"`
	Runtime         platformdocker.ProjectRuntime `json:"runtime"`
	CreatedAt       time.Time                     `json:"created_at"`
	UpdatedAt       time.Time                     `json:"updated_at"`
}

type ListProjects struct {
	projects ProjectRepository
	runtime  Runtime
}

func NewListProjects(projects ProjectRepository, runtime Runtime) *ListProjects {
	return &ListProjects{projects: projects, runtime: runtime}
}

func (u *ListProjects) Execute(ctx context.Context) ([]ProjectView, error) {
	projects, err := u.projects.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ProjectView, 0, len(projects))
	for _, project := range projects {
		view, err := buildProjectView(ctx, u.projects, u.runtime, project)
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return items, nil
}

type GetProject struct {
	projects ProjectRepository
	runtime  Runtime
}

func NewGetProject(projects ProjectRepository, runtime Runtime) *GetProject {
	return &GetProject{projects: projects, runtime: runtime}
}

func (u *GetProject) Execute(ctx context.Context, projectID string) (ProjectView, error) {
	project, err := u.projects.FindByID(ctx, projectID)
	if err != nil {
		return ProjectView{}, err
	}
	return buildProjectView(ctx, u.projects, u.runtime, project)
}

func buildProjectView(ctx context.Context, projects ProjectRepository, runtime Runtime, project projectsdomain.Project) (ProjectView, error) {
	runtimeInfo, err := runtime.InspectProject(ctx, project.Name)
	if err != nil {
		if errors.Is(err, platformdocker.ErrUnavailable) {
			runtimeInfo = platformdocker.ProjectRuntime{
				Status:     "docker_unavailable",
				Containers: []platformdocker.ProjectContainer{},
			}
		} else {
			return ProjectView{}, err
		}
	}

	if runtimeInfo.Status != "" && runtimeInfo.Status != "docker_unavailable" && project.Status != runtimeInfo.Status {
		project.Status = runtimeInfo.Status
		project.LastError = ""
		_ = projects.Save(ctx, project)
	}

	return ProjectView{
		ID:              project.ID,
		Name:            project.Name,
		Kind:            project.Kind,
		TemplateID:      project.TemplateID,
		TemplateVersion: project.TemplateVersion,
		Config:          project.Config,
		ComposePath:     project.ComposePath,
		Status:          project.Status,
		LastError:       project.LastError,
		Runtime:         runtimeInfo,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
	}, nil
}
