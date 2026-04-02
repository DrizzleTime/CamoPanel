package usecase

import (
	"context"
	"errors"

	runtimedomain "camopanel/server/internal/modules/runtime/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
)

var ErrProjectNotFound = errors.New("project not found")

type ProjectRepository interface {
	FindByID(ctx context.Context, projectID string) (runtimedomain.ManagedProject, error)
	Save(ctx context.Context, project runtimedomain.ManagedProject) error
	Delete(ctx context.Context, projectID string) error
}

type RuntimeService interface {
	platformdocker.Runtime
}

type AuditRecorder interface {
	Record(ctx context.Context, entry platformaudit.Entry) error
}

type ManageProjectConfig struct {
	BridgeNetworkName string
}

type ManageProjectInput struct {
	ProjectID string
	ActorID   string
	Action    string
}

type ManageProjectOutput struct {
	Project platformdocker.ProjectRuntime `json:"runtime"`
	Deleted bool                          `json:"deleted"`
}

type ManageProject struct {
	projects ProjectRepository
	runtime  RuntimeService
	audit    AuditRecorder
	cfg      ManageProjectConfig
}

func NewManageProject(projects ProjectRepository, runtime RuntimeService, audit AuditRecorder, cfg ManageProjectConfig) *ManageProject {
	return &ManageProject{
		projects: projects,
		runtime:  runtime,
		audit:    audit,
		cfg:      cfg,
	}
}

func (u *ManageProject) Execute(ctx context.Context, input ManageProjectInput) (ManageProjectOutput, error) {
	project, err := u.projects.FindByID(ctx, input.ProjectID)
	if err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			return ManageProjectOutput{}, ErrProjectNotFound
		}
		return ManageProjectOutput{}, err
	}

	switch input.Action {
	case runtimedomain.ActionStart:
		if err := u.ensureBridgeNetwork(ctx, project); err != nil {
			return ManageProjectOutput{}, u.failProject(ctx, project, err)
		}
		err = u.runtime.Start(ctx, project.Name, project.ComposePath)
	case runtimedomain.ActionStop:
		err = u.runtime.Stop(ctx, project.Name, project.ComposePath)
	case runtimedomain.ActionRestart:
		if err := u.ensureBridgeNetwork(ctx, project); err != nil {
			return ManageProjectOutput{}, u.failProject(ctx, project, err)
		}
		err = u.runtime.Restart(ctx, project.Name, project.ComposePath)
	case runtimedomain.ActionRedeploy:
		if err := u.ensureBridgeNetwork(ctx, project); err != nil {
			return ManageProjectOutput{}, u.failProject(ctx, project, err)
		}
		err = u.runtime.Redeploy(ctx, project.Name, project.ComposePath)
	case runtimedomain.ActionDelete:
		err = u.runtime.Delete(ctx, project.Name, project.ComposePath)
	default:
		return ManageProjectOutput{}, errors.New("unsupported action")
	}
	if err != nil {
		return ManageProjectOutput{}, u.failProject(ctx, project, err)
	}

	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "project_" + input.Action,
		TargetType: "project",
		TargetID:   project.ID,
		Metadata:   map[string]any{"name": project.Name, "action": input.Action},
	})

	if input.Action == runtimedomain.ActionDelete {
		if err := u.projects.Delete(ctx, project.ID); err != nil {
			return ManageProjectOutput{}, err
		}
		return ManageProjectOutput{Deleted: true}, nil
	}

	runtimeInfo, err := u.runtime.InspectProject(ctx, project.Name)
	if err != nil {
		return ManageProjectOutput{}, err
	}

	project.Status = normalizeStatus(project, runtimeInfo)
	project.LastError = ""
	if err := u.projects.Save(ctx, project); err != nil {
		return ManageProjectOutput{}, err
	}

	return ManageProjectOutput{
		Project: runtimeInfo,
	}, nil
}

func (u *ManageProject) ensureBridgeNetwork(ctx context.Context, project runtimedomain.ManagedProject) error {
	if project.TemplateID == runtimedomain.TemplateIDOpenResty || u.cfg.BridgeNetworkName == "" {
		return nil
	}
	return u.runtime.EnsureNetwork(ctx, u.cfg.BridgeNetworkName, "bridge")
}

func (u *ManageProject) failProject(ctx context.Context, project runtimedomain.ManagedProject, err error) error {
	project.LastError = err.Error()
	_ = u.projects.Save(ctx, project)
	return err
}
