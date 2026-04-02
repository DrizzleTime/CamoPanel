package repo_test

import (
	"context"
	"errors"
	"testing"

	"camopanel/server/internal/modules/auth/domain"
	authrepo "camopanel/server/internal/modules/auth/repo"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFindByUsernameNotFoundReturnsDomainError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&authrepo.UserRecord{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}

	repository := authrepo.NewUserRepository(db)

	_, err = repository.FindByUsername(context.Background(), "missing")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected domain.ErrUserNotFound, got %v", err)
	}
}
