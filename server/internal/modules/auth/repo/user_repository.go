package repo

import (
	"context"
	"errors"

	"camopanel/server/internal/modules/auth/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	var record UserRecord
	err := r.db.WithContext(ctx).First(&record, "username = ?", username).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return record.toDomain(), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	var record UserRecord
	err := r.db.WithContext(ctx).First(&record, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	return record.toDomain(), nil
}
