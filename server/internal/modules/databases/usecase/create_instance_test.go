package usecase_test

import (
	"context"
	"testing"
	"time"

	databasesdomain "camopanel/server/internal/modules/databases/domain"
	"camopanel/server/internal/modules/databases/usecase"
	platformdocker "camopanel/server/internal/platform/docker"
)

type databaseRepoStub struct {
	items []databasesdomain.Instance
	saved databasesdomain.Instance
	saveN int
}

func (s *databaseRepoStub) List(_ context.Context, engine string) ([]databasesdomain.Instance, error) {
	if engine == "" {
		return append([]databasesdomain.Instance(nil), s.items...), nil
	}
	result := []databasesdomain.Instance{}
	for _, item := range s.items {
		if item.Engine == engine {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *databaseRepoStub) FindByID(_ context.Context, instanceID string) (databasesdomain.Instance, error) {
	for _, item := range s.items {
		if item.ID == instanceID {
			return item, nil
		}
	}
	return databasesdomain.Instance{}, databasesdomain.ErrInstanceNotFound
}

func (s *databaseRepoStub) Save(_ context.Context, item databasesdomain.Instance) error {
	s.saved = item
	s.saveN++
	for i, current := range s.items {
		if current.ID == item.ID {
			s.items[i] = item
			return nil
		}
	}
	s.items = append(s.items, item)
	return nil
}

type databaseRuntimeStub struct {
	status platformdocker.ProjectRuntime
}

func (s *databaseRuntimeStub) EnsureNetwork(context.Context, string, string) error { return nil }
func (s *databaseRuntimeStub) Deploy(context.Context, string, string) error        { return nil }
func (s *databaseRuntimeStub) Start(context.Context, string, string) error         { return nil }
func (s *databaseRuntimeStub) Stop(context.Context, string, string) error          { return nil }
func (s *databaseRuntimeStub) Restart(context.Context, string, string) error       { return nil }
func (s *databaseRuntimeStub) Redeploy(context.Context, string, string) error      { return nil }
func (s *databaseRuntimeStub) Delete(context.Context, string, string) error        { return nil }
func (s *databaseRuntimeStub) InspectProject(context.Context, string) (platformdocker.ProjectRuntime, error) {
	return s.status, nil
}
func (s *databaseRuntimeStub) ProjectLogs(context.Context, string, int) (string, error) {
	return "", nil
}

type databaseContainerStub struct {
	last []string
}

func (s *databaseContainerStub) InspectContainer(context.Context, string) (platformdocker.ContainerStatus, error) {
	return platformdocker.ContainerStatus{}, nil
}

func (s *databaseContainerStub) ExecInContainer(_ context.Context, _ string, args ...string) (string, error) {
	s.last = append([]string(nil), args...)
	return "", nil
}

func TestListInstancesBuildsRuntimeView(t *testing.T) {
	repo := &databaseRepoStub{
		items: []databasesdomain.Instance{{
			ID:        "db-1",
			Name:      "mysql-demo",
			Engine:    databasesdomain.EngineMySQL,
			Config:    map[string]any{"port": 3306, "username": "app", "database": "demo"},
			Status:    "running",
			CreatedAt: time.Unix(1, 0),
			UpdatedAt: time.Unix(2, 0),
		}},
	}
	svc := usecase.NewService(repo, &databaseRuntimeStub{
		status: platformdocker.ProjectRuntime{
			Status:     platformdocker.StatusRunning,
			Containers: []platformdocker.ProjectContainer{{Name: "mysql-demo-1", State: "running"}},
		},
	}, &databaseContainerStub{}, nil, nil, usecase.Config{})

	items, err := svc.ListInstances(context.Background(), databasesdomain.EngineMySQL)
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(items))
	}
	if items[0].Connection.AdminUsername != "root" {
		t.Fatalf("expected mysql admin user root, got %s", items[0].Connection.AdminUsername)
	}
}
