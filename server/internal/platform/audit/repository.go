package audit

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, event Record) error
}

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, event Record) error {
	return r.db.WithContext(ctx).Create(&event).Error
}
