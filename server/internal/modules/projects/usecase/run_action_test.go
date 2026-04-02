package usecase_test

import (
	"context"
	"testing"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	"camopanel/server/internal/modules/projects/usecase"
	platformdocker "camopanel/server/internal/platform/docker"
)

type actionRunnerStub struct {
	deleted bool
}

func (s *actionRunnerStub) Execute(_ context.Context, input usecase.ProjectActionInput) (usecase.ProjectActionOutput, error) {
	if input.Action == projectsdomain.ActionDelete {
		return usecase.ProjectActionOutput{Deleted: true}, nil
	}
	return usecase.ProjectActionOutput{
		Runtime: platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
	}, nil
}

func (s *actionRunnerStub) Logs(_ context.Context, _ string, _ int) (string, error) {
	return "demo logs", nil
}

func TestRunActionReturnsDeletedResult(t *testing.T) {
	repo := newProjectRepoStub()
	repo.items["project-1"] = projectsdomain.Project{ID: "project-1", Name: "demo"}
	uc := usecase.NewRunAction(repo, &actionRunnerStub{})

	got, err := uc.Execute(context.Background(), usecase.RunActionInput{
		ProjectID: "project-1",
		Action:    projectsdomain.ActionDelete,
	})
	if err != nil {
		t.Fatalf("execute run action: %v", err)
	}
	if !got.Deleted {
		t.Fatal("expected deleted result")
	}
}

func TestRunActionCanReadProjectLogs(t *testing.T) {
	repo := newProjectRepoStub()
	repo.items["project-1"] = projectsdomain.Project{ID: "project-1", Name: "demo"}
	uc := usecase.NewRunAction(repo, &actionRunnerStub{})

	logs, err := uc.Logs(context.Background(), "project-1", 200)
	if err != nil {
		t.Fatalf("project logs: %v", err)
	}
	if logs != "demo logs" {
		t.Fatalf("expected demo logs, got %s", logs)
	}
}
