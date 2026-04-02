package usecase

import (
	"context"

	runtimedomain "camopanel/server/internal/modules/runtime/domain"
	platformdocker "camopanel/server/internal/platform/docker"
)

type QueryRuntime struct {
	projects ProjectRepository
	runtime  RuntimeService
}

func NewQueryRuntime(projects ProjectRepository, runtime RuntimeService) *QueryRuntime {
	return &QueryRuntime{
		projects: projects,
		runtime:  runtime,
	}
}

func (u *QueryRuntime) Execute(ctx context.Context, projectID string) (platformdocker.ProjectRuntime, error) {
	project, err := u.projects.FindByID(ctx, projectID)
	if err != nil {
		return platformdocker.ProjectRuntime{}, err
	}
	return u.runtime.InspectProject(ctx, project.Name)
}

func (u *QueryRuntime) Logs(ctx context.Context, projectID string, tail int) (string, error) {
	project, err := u.projects.FindByID(ctx, projectID)
	if err != nil {
		return "", err
	}
	return u.runtime.ProjectLogs(ctx, project.Name, tail)
}

func normalizeStatus(project runtimedomain.ManagedProject, runtime platformdocker.ProjectRuntime) string {
	if runtime.Status == "" {
		return project.Status
	}
	return runtime.Status
}
