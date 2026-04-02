package usecase

import (
	"context"
	"time"

	projectsusecase "camopanel/server/internal/modules/projects/usecase"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
	platformsystem "camopanel/server/internal/platform/system"
	"camopanel/server/internal/services"
)

type ProjectLister interface {
	Execute(ctx context.Context) ([]projectsusecase.ProjectView, error)
}

type WebsiteLister interface {
	Execute(ctx context.Context) ([]websitesdomain.Website, error)
}

type DashboardSnapshot struct {
	Metrics     services.HostMetrics          `json:"metrics"`
	Projects    []projectsusecase.ProjectView `json:"projects"`
	Websites    []websitesdomain.Website      `json:"websites"`
	GeneratedAt time.Time                     `json:"generated_at"`
}

type Dashboard struct {
	system   *platformsystem.Service
	projects ProjectLister
	websites WebsiteLister
}

func NewDashboard(system *platformsystem.Service, projects ProjectLister, websites WebsiteLister) *Dashboard {
	return &Dashboard{system: system, projects: projects, websites: websites}
}

func (u *Dashboard) Build(ctx context.Context) (DashboardSnapshot, error) {
	metrics, err := u.system.Metrics(ctx)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	projects, err := u.projects.Execute(ctx)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	websites, err := u.websites.Execute(ctx)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	return DashboardSnapshot{
		Metrics:     metrics,
		Projects:    projects,
		Websites:    websites,
		GeneratedAt: time.Now().UTC(),
	}, nil
}
