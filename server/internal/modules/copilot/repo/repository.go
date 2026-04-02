package repo

import (
	"context"

	copilotdomain "camopanel/server/internal/modules/copilot/domain"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ListProviders(ctx context.Context) ([]copilotdomain.Provider, error) {
	var records []ProviderRecord
	if err := r.db.WithContext(ctx).Order("created_at desc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]copilotdomain.Provider, 0, len(records))
	for _, item := range records {
		items = append(items, toProvider(item))
	}
	return items, nil
}

func (r *Repository) FindProviderByID(ctx context.Context, providerID string) (copilotdomain.Provider, error) {
	var record ProviderRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", providerID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return copilotdomain.Provider{}, copilotdomain.ErrProviderNotFound
		}
		return copilotdomain.Provider{}, err
	}
	return toProvider(record), nil
}

func (r *Repository) SaveProvider(ctx context.Context, item copilotdomain.Provider) error {
	return r.db.WithContext(ctx).Save(fromProvider(item)).Error
}

func (r *Repository) DeleteProvider(ctx context.Context, providerID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider_id = ?", providerID).Delete(&ModelRecord{}).Error; err != nil {
			return err
		}
		return tx.Delete(&ProviderRecord{}, "id = ?", providerID).Error
	})
}

func (r *Repository) ListModels(ctx context.Context) ([]copilotdomain.Model, error) {
	var records []ModelRecord
	if err := r.db.WithContext(ctx).Order("is_default desc, created_at asc").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]copilotdomain.Model, 0, len(records))
	for _, item := range records {
		items = append(items, toModel(item))
	}
	return items, nil
}

func (r *Repository) FindModelByID(ctx context.Context, modelID string) (copilotdomain.Model, error) {
	var record ModelRecord
	if err := r.db.WithContext(ctx).First(&record, "id = ?", modelID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return copilotdomain.Model{}, copilotdomain.ErrModelNotFound
		}
		return copilotdomain.Model{}, err
	}
	return toModel(record), nil
}

func (r *Repository) SaveModel(ctx context.Context, item copilotdomain.Model) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if item.IsDefault {
			if err := tx.Model(&ModelRecord{}).Where("id <> ? AND is_default = ?", item.ID, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(fromModel(item)).Error
	})
}

func (r *Repository) DeleteModel(ctx context.Context, modelID string) error {
	return r.db.WithContext(ctx).Delete(&ModelRecord{}, "id = ?", modelID).Error
}

func toProvider(item ProviderRecord) copilotdomain.Provider {
	return copilotdomain.Provider{
		ID:        item.ID,
		Name:      item.Name,
		Type:      item.Type,
		BaseURL:   item.BaseURL,
		APIKey:    item.APIKey,
		Enabled:   item.Enabled,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func fromProvider(item copilotdomain.Provider) ProviderRecord {
	return ProviderRecord{
		ID:        item.ID,
		Name:      item.Name,
		Type:      item.Type,
		BaseURL:   item.BaseURL,
		APIKey:    item.APIKey,
		Enabled:   item.Enabled,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}

func toModel(item ModelRecord) copilotdomain.Model {
	return copilotdomain.Model{
		ID:         item.ID,
		ProviderID: item.ProviderID,
		Name:       item.Name,
		Enabled:    item.Enabled,
		IsDefault:  item.IsDefault,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

func fromModel(item copilotdomain.Model) ModelRecord {
	return ModelRecord{
		ID:         item.ID,
		ProviderID: item.ProviderID,
		Name:       item.Name,
		Enabled:    item.Enabled,
		IsDefault:  item.IsDefault,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}
