package audit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	event, err := entry.toModel(uuid.NewString())
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("create audit event: %w", err)
	}
	return nil
}
