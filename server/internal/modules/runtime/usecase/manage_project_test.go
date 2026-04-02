package usecase_test

import (
	"context"
	"errors"
	"testing"

	runtimedomain "camopanel/server/internal/modules/runtime/domain"
	"camopanel/server/internal/modules/runtime/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
)

type projectRepositoryStub struct {
	project   runtimedomain.ManagedProject
	saved     runtimedomain.ManagedProject
	deleted   string
	findErr   error
	saveErr   error
	deleteErr error
}

func (s *projectRepositoryStub) FindByID(_ context.Context, projectID string) (runtimedomain.ManagedProject, error) {
	if s.findErr != nil {
		return runtimedomain.ManagedProject{}, s.findErr
	}
	if s.project.ID != projectID {
		return runtimedomain.ManagedProject{}, usecase.ErrProjectNotFound
	}
	return s.project, nil
}

func (s *projectRepositoryStub) Save(_ context.Context, project runtimedomain.ManagedProject) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.saved = project
	s.project = project
	return nil
}

func (s *projectRepositoryStub) Delete(_ context.Context, projectID string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = projectID
	return nil
}

type dockerRuntimeStub struct {
	ensureNetworkCalls int
	startCalls         int
	stopCalls          int
	restartCalls       int
	redeployCalls      int
	deleteCalls        int
	lastProject        string
	lastComposePath    string
	lastNetworkName    string
	lastNetworkDriver  string
	runtime            platformdocker.ProjectRuntime
	logs               string
}

func (s *dockerRuntimeStub) EnsureNetwork(_ context.Context, name, driver string) error {
	s.ensureNetworkCalls++
	s.lastNetworkName = name
	s.lastNetworkDriver = driver
	return nil
}

func (s *dockerRuntimeStub) Deploy(_ context.Context, projectName, composePath string) error {
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) Start(_ context.Context, projectName, composePath string) error {
	s.startCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) Stop(_ context.Context, projectName, composePath string) error {
	s.stopCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) Restart(_ context.Context, projectName, composePath string) error {
	s.restartCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) Redeploy(_ context.Context, projectName, composePath string) error {
	s.redeployCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) Delete(_ context.Context, projectName, composePath string) error {
	s.deleteCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *dockerRuntimeStub) InspectProject(_ context.Context, _ string) (platformdocker.ProjectRuntime, error) {
	return s.runtime, nil
}

func (s *dockerRuntimeStub) ProjectLogs(_ context.Context, _ string, _ int) (string, error) {
	return s.logs, nil
}

type auditRecorderStub struct {
	entries []platformaudit.Entry
}

func (s *auditRecorderStub) Record(_ context.Context, entry platformaudit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestManageProjectStartEnsuresNetworkAndPersistsRuntimeStatus(t *testing.T) {
	repo := &projectRepositoryStub{
		project: runtimedomain.ManagedProject{
			ID:          "project-1",
			Name:        "demo",
			TemplateID:  "demo",
			ComposePath: "/tmp/demo/compose.yaml",
			Status:      platformdocker.StatusStopped,
		},
	}
	runtime := &dockerRuntimeStub{
		runtime: platformdocker.ProjectRuntime{
			Status: platformdocker.StatusRunning,
			Containers: []platformdocker.ProjectContainer{
				{Name: "demo-app", State: "running"},
			},
		},
	}
	audit := &auditRecorderStub{}
	uc := usecase.NewManageProject(repo, runtime, audit, usecase.ManageProjectConfig{
		BridgeNetworkName: "camopanel",
	})

	got, err := uc.Execute(context.Background(), usecase.ManageProjectInput{
		ProjectID: "project-1",
		ActorID:   "user-1",
		Action:    runtimedomain.ActionStart,
	})
	if err != nil {
		t.Fatalf("execute manage project: %v", err)
	}

	if runtime.ensureNetworkCalls != 1 {
		t.Fatalf("expected ensure network once, got %d", runtime.ensureNetworkCalls)
	}
	if runtime.startCalls != 1 {
		t.Fatalf("expected start once, got %d", runtime.startCalls)
	}
	if repo.saved.Status != platformdocker.StatusRunning {
		t.Fatalf("expected saved status %s, got %s", platformdocker.StatusRunning, repo.saved.Status)
	}
	if got.Project.Status != platformdocker.StatusRunning {
		t.Fatalf("expected returned project status %s, got %s", platformdocker.StatusRunning, got.Project.Status)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "project_start" {
		t.Fatalf("unexpected audit entries: %+v", audit.entries)
	}
}

func TestManageProjectDeleteRemovesRecord(t *testing.T) {
	repo := &projectRepositoryStub{
		project: runtimedomain.ManagedProject{
			ID:          "project-1",
			Name:        "demo",
			TemplateID:  "demo",
			ComposePath: "/tmp/demo/compose.yaml",
		},
	}
	runtime := &dockerRuntimeStub{}
	uc := usecase.NewManageProject(repo, runtime, &auditRecorderStub{}, usecase.ManageProjectConfig{
		BridgeNetworkName: "camopanel",
	})

	got, err := uc.Execute(context.Background(), usecase.ManageProjectInput{
		ProjectID: "project-1",
		ActorID:   "user-1",
		Action:    runtimedomain.ActionDelete,
	})
	if err != nil {
		t.Fatalf("execute manage project delete: %v", err)
	}

	if runtime.deleteCalls != 1 {
		t.Fatalf("expected delete once, got %d", runtime.deleteCalls)
	}
	if repo.deleted != "project-1" {
		t.Fatalf("expected project deletion to persist, got %s", repo.deleted)
	}
	if !got.Deleted {
		t.Fatal("expected deleted result")
	}
}

func TestQueryRuntimeReturnsProjectLogs(t *testing.T) {
	repo := &projectRepositoryStub{
		project: runtimedomain.ManagedProject{
			ID:          "project-1",
			Name:        "demo",
			ComposePath: "/tmp/demo/compose.yaml",
		},
	}
	runtime := &dockerRuntimeStub{logs: "demo logs"}
	uc := usecase.NewQueryRuntime(repo, runtime)

	logs, err := uc.Logs(context.Background(), "project-1", 200)
	if err != nil {
		t.Fatalf("query runtime logs: %v", err)
	}
	if logs != "demo logs" {
		t.Fatalf("expected logs demo logs, got %s", logs)
	}
}

func TestManageProjectReturnsNotFoundWhenProjectIsMissing(t *testing.T) {
	repo := &projectRepositoryStub{findErr: usecase.ErrProjectNotFound}
	uc := usecase.NewManageProject(repo, &dockerRuntimeStub{}, &auditRecorderStub{}, usecase.ManageProjectConfig{})

	_, err := uc.Execute(context.Background(), usecase.ManageProjectInput{
		ProjectID: "missing",
		Action:    runtimedomain.ActionStart,
	})
	if !errors.Is(err, usecase.ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got %v", err)
	}
}
