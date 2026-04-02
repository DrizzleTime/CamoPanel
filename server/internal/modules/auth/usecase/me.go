package usecase

import (
	"context"

	"camopanel/server/internal/modules/auth/domain"
)

type Me struct {
	repo UserRepository
}

func NewMe(repo UserRepository) *Me {
	return &Me{repo: repo}
}

func (u *Me) Execute(ctx context.Context, userID string) (domain.User, error) {
	return u.repo.FindByID(ctx, userID)
}
