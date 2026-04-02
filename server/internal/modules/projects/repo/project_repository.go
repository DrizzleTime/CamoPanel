package repo

import (
	"context"
	"encoding/json"

	projectsdomain "camopanel/server/internal/modules/projects/domain"

	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) List(ctx context.Context) ([]projectsdomain.Project, error) {
	var records []ProjectRecord
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]projectsdomain.Project, 0, len(records))
	for _, record := range records {
		items = append(items, toDomain(record))
	}
	return items, nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, projectID string) (projectsdomain.Project, error) {
	var record ProjectRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return projectsdomain.Project{}, projectsdomain.ErrProjectNotFound
		}
		return projectsdomain.Project{}, err
	}
	return toDomain(record), nil
}

func (r *ProjectRepository) FindByName(ctx context.Context, name string) (projectsdomain.Project, error) {
	var record ProjectRecord
	if err := r.db.WithContext(ctx).First(&record, "name = ?", name).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return projectsdomain.Project{}, projectsdomain.ErrProjectNotFound
		}
		return projectsdomain.Project{}, err
	}
	return toDomain(record), nil
}

func (r *ProjectRepository) CountByTemplateID(ctx context.Context, templateID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ProjectRecord{}).Where("template_id = ?", templateID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *ProjectRepository) Create(ctx context.Context, project projectsdomain.Project) error {
	return r.db.WithContext(ctx).Create(fromDomain(project)).Error
}

func (r *ProjectRepository) Save(ctx context.Context, project projectsdomain.Project) error {
	return r.db.WithContext(ctx).Save(fromDomain(project)).Error
}

func (r *ProjectRepository) Delete(ctx context.Context, projectID string) error {
	return r.db.WithContext(ctx).Delete(&ProjectRecord{}, "id = ?", projectID).Error
}

func toDomain(record ProjectRecord) projectsdomain.Project {
	config := map[string]any{}
	if record.ConfigJSON != "" {
		_ = json.Unmarshal([]byte(record.ConfigJSON), &config)
	}
	return projectsdomain.Project{
		ID:              record.ID,
		Name:            record.Name,
		Kind:            record.Kind,
		TemplateID:      record.TemplateID,
		TemplateVersion: record.TemplateVersion,
		Config:          config,
		ComposePath:     record.ComposePath,
		Status:          record.Status,
		LastError:       record.LastError,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

func fromDomain(project projectsdomain.Project) ProjectRecord {
	configJSON := "{}"
	if project.Config != nil {
		if raw, err := json.Marshal(project.Config); err == nil {
			configJSON = string(raw)
		}
	}
	return ProjectRecord{
		ID:              project.ID,
		Name:            project.Name,
		Kind:            project.Kind,
		TemplateID:      project.TemplateID,
		TemplateVersion: project.TemplateVersion,
		ConfigJSON:      configJSON,
		ComposePath:     project.ComposePath,
		Status:          project.Status,
		LastError:       project.LastError,
		CreatedAt:       project.CreatedAt,
		UpdatedAt:       project.UpdatedAt,
	}
}
