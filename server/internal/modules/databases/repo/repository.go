package repo

import (
	"context"
	"slices"

	databasesdomain "camopanel/server/internal/modules/databases/domain"
	projectsdomain "camopanel/server/internal/modules/projects/domain"
	projectsrepo "camopanel/server/internal/modules/projects/repo"
)

type Repository struct {
	projects *projectsrepo.ProjectRepository
}

func NewRepository(projects *projectsrepo.ProjectRepository) *Repository {
	return &Repository{projects: projects}
}

func (r *Repository) List(ctx context.Context, engine string) ([]databasesdomain.Instance, error) {
	items, err := r.projects.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]databasesdomain.Instance, 0, len(items))
	for _, item := range items {
		if !slices.Contains(databasesdomain.ManagedEngines, item.TemplateID) {
			continue
		}
		if engine != "" && item.TemplateID != engine {
			continue
		}
		result = append(result, toInstance(item))
	}
	return result, nil
}

func (r *Repository) FindByID(ctx context.Context, instanceID string) (databasesdomain.Instance, error) {
	item, err := r.projects.FindByID(ctx, instanceID)
	if err != nil {
		if err == projectsdomain.ErrProjectNotFound {
			return databasesdomain.Instance{}, databasesdomain.ErrInstanceNotFound
		}
		return databasesdomain.Instance{}, err
	}
	if !slices.Contains(databasesdomain.ManagedEngines, item.TemplateID) {
		return databasesdomain.Instance{}, databasesdomain.ErrInstanceNotFound
	}
	return toInstance(item), nil
}

func (r *Repository) Save(ctx context.Context, instance databasesdomain.Instance) error {
	return r.projects.Save(ctx, projectsdomain.Project{
		ID:              instance.ID,
		Name:            instance.Name,
		Kind:            projectsdomain.KindTemplate,
		TemplateID:      instance.Engine,
		TemplateVersion: instance.TemplateVersion,
		Config:          instance.Config,
		ComposePath:     instance.ComposePath,
		Status:          instance.Status,
		LastError:       instance.LastError,
		CreatedAt:       instance.CreatedAt,
		UpdatedAt:       instance.UpdatedAt,
	})
}

func toInstance(item projectsdomain.Project) databasesdomain.Instance {
	return databasesdomain.Instance{
		ID:              item.ID,
		Name:            item.Name,
		Engine:          item.TemplateID,
		TemplateVersion: item.TemplateVersion,
		Config:          item.Config,
		ComposePath:     item.ComposePath,
		Status:          item.Status,
		LastError:       item.LastError,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}
