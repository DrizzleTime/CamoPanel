package system

import (
	"context"

	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/services"
)

type HostReader interface {
	Summary(ctx context.Context) (services.HostSummary, error)
	Metrics(ctx context.Context) (services.HostMetrics, error)
}

type HostController interface {
	GetDockerSettings(ctx context.Context) (services.DockerDaemonSettings, error)
	UpdateDockerSettings(ctx context.Context, mirrors []string) (services.DockerDaemonSettings, error)
	RestartDocker(ctx context.Context) error
}

type DockerReader interface {
	GetSystemInfo(ctx context.Context) (platformdocker.SystemInfo, error)
}

type Service struct {
	host        HostReader
	hostControl HostController
	docker      DockerReader
}

func NewService(host HostReader, hostControl HostController, docker DockerReader) *Service {
	return &Service{host: host, hostControl: hostControl, docker: docker}
}

func (s *Service) Summary(ctx context.Context) (services.HostSummary, error) {
	return s.host.Summary(ctx)
}

func (s *Service) Metrics(ctx context.Context) (services.HostMetrics, error) {
	return s.host.Metrics(ctx)
}

func (s *Service) DockerSystem(ctx context.Context) (platformdocker.SystemInfo, error) {
	return s.docker.GetSystemInfo(ctx)
}

func (s *Service) DockerSettings(ctx context.Context) (services.DockerDaemonSettings, error) {
	return s.hostControl.GetDockerSettings(ctx)
}

func (s *Service) UpdateDockerSettings(ctx context.Context, mirrors []string) (services.DockerDaemonSettings, error) {
	return s.hostControl.UpdateDockerSettings(ctx, mirrors)
}

func (s *Service) RestartDocker(ctx context.Context) error {
	return s.hostControl.RestartDocker(ctx)
}
