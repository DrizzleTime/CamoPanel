package usecase

import (
	"context"

	runtimedomain "camopanel/server/internal/modules/runtime/domain"
	runtimeusecase "camopanel/server/internal/modules/runtime/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
)

type ProjectActionInput struct {
	ProjectID string
	ActorID   string
	Action    string
}

type ProjectActionOutput struct {
	Runtime platformdocker.ProjectRuntime
	Deleted bool
}

type ProjectActionRunner interface {
	Execute(ctx context.Context, input ProjectActionInput) (ProjectActionOutput, error)
	Logs(ctx context.Context, projectID string, tail int) (string, error)
}

type RunActionInput struct {
	ProjectID string
	ActorID   string
	Action    string
}

type RunActionOutput struct {
	Project *ProjectView
	Deleted bool
}

type RunAction struct {
	projects ProjectRepository
	runner   ProjectActionRunner
}

func NewRunAction(projects ProjectRepository, runner ProjectActionRunner) *RunAction {
	return &RunAction{projects: projects, runner: runner}
}

func (u *RunAction) Execute(ctx context.Context, input RunActionInput) (RunActionOutput, error) {
	if _, err := u.projects.FindByID(ctx, input.ProjectID); err != nil {
		return RunActionOutput{}, err
	}

	result, err := u.runner.Execute(ctx, ProjectActionInput{
		ProjectID: input.ProjectID,
		ActorID:   input.ActorID,
		Action:    input.Action,
	})
	if err != nil {
		return RunActionOutput{}, err
	}
	if result.Deleted {
		return RunActionOutput{Deleted: true}, nil
	}

	view, err := NewGetProject(u.projects, &staticRuntime{runtime: result.Runtime}).Execute(ctx, input.ProjectID)
	if err != nil {
		return RunActionOutput{}, err
	}
	return RunActionOutput{Project: &view}, nil
}

func (u *RunAction) Logs(ctx context.Context, projectID string, tail int) (string, error) {
	if _, err := u.projects.FindByID(ctx, projectID); err != nil {
		return "", err
	}
	return u.runner.Logs(ctx, projectID, tail)
}

type staticRuntime struct {
	runtime platformdocker.ProjectRuntime
}

func (r *staticRuntime) EnsureNetwork(context.Context, string, string) error { return nil }
func (r *staticRuntime) Deploy(context.Context, string, string) error        { return nil }
func (r *staticRuntime) Start(context.Context, string, string) error         { return nil }
func (r *staticRuntime) Stop(context.Context, string, string) error          { return nil }
func (r *staticRuntime) Restart(context.Context, string, string) error       { return nil }
func (r *staticRuntime) Redeploy(context.Context, string, string) error      { return nil }
func (r *staticRuntime) Delete(context.Context, string, string) error        { return nil }
func (r *staticRuntime) InspectProject(context.Context, string) (platformdocker.ProjectRuntime, error) {
	return r.runtime, nil
}
func (r *staticRuntime) ProjectLogs(context.Context, string, int) (string, error) { return "", nil }

type runtimeActionRunner struct {
	manage *runtimeusecase.ManageProject
	query  *runtimeusecase.QueryRuntime
}

func NewRuntimeActionRunner(projects ProjectRepository, runtime Runtime, audit *platformaudit.Service, cfg ProjectConfig) ProjectActionRunner {
	adapter := &runtimeProjectRepository{projects: projects}
	return &runtimeActionRunner{
		manage: runtimeusecase.NewManageProject(adapter, runtime, audit, runtimeusecase.ManageProjectConfig{
			BridgeNetworkName: cfg.BridgeNetworkName,
		}),
		query: runtimeusecase.NewQueryRuntime(adapter, runtime),
	}
}

func (r *runtimeActionRunner) Execute(ctx context.Context, input ProjectActionInput) (ProjectActionOutput, error) {
	result, err := r.manage.Execute(ctx, runtimeusecase.ManageProjectInput{
		ProjectID: input.ProjectID,
		ActorID:   input.ActorID,
		Action:    input.Action,
	})
	if err != nil {
		return ProjectActionOutput{}, err
	}
	return ProjectActionOutput{
		Runtime: result.Project,
		Deleted: result.Deleted,
	}, nil
}

func (r *runtimeActionRunner) Logs(ctx context.Context, projectID string, tail int) (string, error) {
	return r.query.Logs(ctx, projectID, tail)
}

type runtimeProjectRepository struct {
	projects ProjectRepository
}

func (r *runtimeProjectRepository) FindByID(ctx context.Context, projectID string) (runtimedomain.ManagedProject, error) {
	project, err := r.projects.FindByID(ctx, projectID)
	if err != nil {
		return runtimedomain.ManagedProject{}, err
	}
	return runtimedomain.ManagedProject{
		ID:          project.ID,
		Name:        project.Name,
		TemplateID:  project.TemplateID,
		ComposePath: project.ComposePath,
		Status:      project.Status,
		LastError:   project.LastError,
	}, nil
}

func (r *runtimeProjectRepository) Save(ctx context.Context, project runtimedomain.ManagedProject) error {
	current, err := r.projects.FindByID(ctx, project.ID)
	if err != nil {
		return err
	}
	current.Name = project.Name
	current.TemplateID = project.TemplateID
	current.ComposePath = project.ComposePath
	current.Status = project.Status
	current.LastError = project.LastError
	return r.projects.Save(ctx, current)
}

func (r *runtimeProjectRepository) Delete(ctx context.Context, projectID string) error {
	return r.projects.Delete(ctx, projectID)
}
