package openresty

import (
	"context"

	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/services"
)

type Manager interface {
	Status(ctx context.Context) Status
	EnsureReady(ctx context.Context) error
	CreateWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error)
	UpdateWebsite(ctx context.Context, spec WebsiteSpec, configPath string) (WebsiteMaterialized, error)
	DeleteWebsite(ctx context.Context, configPath string) error
	PreviewWebsiteConfig(spec WebsiteSpec) (string, error)
	SyncWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error)
	IssueCertificate(ctx context.Context, spec CertificateSpec) (CertificateMaterialized, error)
	DeleteCertificate(ctx context.Context, domain string) error
}

type Service struct {
	inner services.OpenRestyManager
}

func NewService(docker platformdocker.ContainerOperator, containerName, dataDir string) *Service {
	return &Service{
		inner: services.NewOpenRestyService(dockerAdapter{inner: docker}, containerName, dataDir),
	}
}

func (s *Service) Status(ctx context.Context) Status {
	status := s.inner.Status(ctx)
	return Status(status)
}

func (s *Service) EnsureReady(ctx context.Context) error {
	return s.inner.EnsureReady(ctx)
}

func (s *Service) CreateWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error) {
	item, err := s.inner.CreateWebsite(ctx, services.WebsiteSpec(spec))
	return WebsiteMaterialized(item), err
}

func (s *Service) UpdateWebsite(ctx context.Context, spec WebsiteSpec, configPath string) (WebsiteMaterialized, error) {
	item, err := s.inner.UpdateWebsite(ctx, services.WebsiteSpec(spec), configPath)
	return WebsiteMaterialized(item), err
}

func (s *Service) DeleteWebsite(ctx context.Context, configPath string) error {
	return s.inner.DeleteWebsite(ctx, configPath)
}

func (s *Service) PreviewWebsiteConfig(spec WebsiteSpec) (string, error) {
	return s.inner.PreviewWebsiteConfig(services.WebsiteSpec(spec))
}

func (s *Service) SyncWebsite(ctx context.Context, spec WebsiteSpec) (WebsiteMaterialized, error) {
	item, err := s.inner.SyncWebsite(ctx, services.WebsiteSpec(spec))
	return WebsiteMaterialized(item), err
}

func (s *Service) IssueCertificate(ctx context.Context, spec CertificateSpec) (CertificateMaterialized, error) {
	item, err := s.inner.IssueCertificate(ctx, services.CertificateSpec(spec))
	return CertificateMaterialized(item), err
}

func (s *Service) DeleteCertificate(ctx context.Context, domain string) error {
	return s.inner.DeleteCertificate(ctx, domain)
}

type dockerAdapter struct {
	inner platformdocker.ContainerOperator
}

func (a dockerAdapter) InspectContainer(ctx context.Context, containerName string) (services.ContainerStatus, error) {
	status, err := a.inner.InspectContainer(ctx, containerName)
	if err != nil {
		return services.ContainerStatus{}, err
	}

	mounts := make([]services.ContainerMount, 0, len(status.Mounts))
	for _, item := range status.Mounts {
		mounts = append(mounts, services.ContainerMount{
			Source:      item.Source,
			Destination: item.Destination,
		})
	}

	return services.ContainerStatus{
		Exists:  status.Exists,
		Running: status.Running,
		Name:    status.Name,
		Image:   status.Image,
		Status:  status.Status,
		Mounts:  mounts,
	}, nil
}

func (a dockerAdapter) ExecInContainer(ctx context.Context, containerName string, args ...string) (string, error) {
	return a.inner.ExecInContainer(ctx, containerName, args...)
}
