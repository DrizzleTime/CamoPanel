package usecase_test

import (
	"context"
	"testing"
	"time"

	projectsusecase "camopanel/server/internal/modules/projects/usecase"
	systemusecase "camopanel/server/internal/modules/system/usecase"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
	platformsystem "camopanel/server/internal/platform/system"
	"camopanel/server/internal/services"
)

type hostStub struct{}

func (s *hostStub) Summary(context.Context) (services.HostSummary, error) {
	return services.HostSummary{}, nil
}
func (s *hostStub) Metrics(context.Context) (services.HostMetrics, error) {
	return services.HostMetrics{Summary: services.HostSummary{Hostname: "demo"}, SampleIntervalSeconds: 5}, nil
}

type hostControlStub struct{}

func (s *hostControlStub) GetDockerSettings(context.Context) (services.DockerDaemonSettings, error) {
	return services.DockerDaemonSettings{}, nil
}
func (s *hostControlStub) UpdateDockerSettings(context.Context, []string) (services.DockerDaemonSettings, error) {
	return services.DockerDaemonSettings{}, nil
}
func (s *hostControlStub) RestartDocker(context.Context) error { return nil }

type dockerStub struct{}

func (s *dockerStub) GetSystemInfo(context.Context) (struct{}, error) { return struct{}{}, nil }

type projectListerStub struct{}

func (s *projectListerStub) Execute(context.Context) ([]projectsusecase.ProjectView, error) {
	return []projectsusecase.ProjectView{{ID: "project-1", Name: "demo"}}, nil
}

type websiteListerStub struct{}

func (s *websiteListerStub) Execute(context.Context) ([]websitesdomain.Website, error) {
	return []websitesdomain.Website{{ID: "website-1", Name: "demo"}}, nil
}

func TestDashboardBuildReturnsSnapshot(t *testing.T) {
	system := platformsystem.NewService(&hostStub{}, &hostControlStub{}, nil)
	dashboard := systemusecase.NewDashboard(system, &projectListerStub{}, &websiteListerStub{})

	item, err := dashboard.Build(context.Background())
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	if item.Metrics.Summary.Hostname != "demo" {
		t.Fatalf("expected hostname demo, got %s", item.Metrics.Summary.Hostname)
	}
	if len(item.Projects) != 1 || len(item.Websites) != 1 {
		t.Fatalf("unexpected snapshot: %+v", item)
	}
	if item.GeneratedAt.Before(time.Now().UTC().Add(-time.Minute)) {
		t.Fatalf("expected recent generated time, got %s", item.GeneratedAt)
	}
}
