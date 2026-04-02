package usecase_test

import (
	"context"
	"errors"
	"testing"

	"camopanel/server/internal/modules/auth/domain"
	"camopanel/server/internal/modules/auth/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformauth "camopanel/server/internal/platform/auth"
)

type loginRepoStub struct {
	user domain.User
	err  error
}

func (s *loginRepoStub) FindByUsername(_ context.Context, username string) (domain.User, error) {
	if s.err != nil {
		return domain.User{}, s.err
	}
	if s.user.Username != username {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.user, nil
}

func (s *loginRepoStub) FindByID(_ context.Context, _ string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}

type auditRecorderStub struct {
	entries []platformaudit.Entry
}

func (s *auditRecorderStub) Record(_ context.Context, entry platformaudit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestLoginSuccessIssuesSessionAndReturnsUser(t *testing.T) {
	passwordHash, err := platformauth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &loginRepoStub{
		user: domain.User{
			ID:           "u-1",
			Username:     "admin",
			Role:         "super_admin",
			PasswordHash: passwordHash,
		},
	}
	audit := &auditRecorderStub{}
	uc := usecase.NewLogin(repo, platformauth.NewSessionManager("test-secret"), audit)

	got, err := uc.Execute(context.Background(), usecase.LoginInput{
		Username: "admin",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("execute login: %v", err)
	}

	if got.Token == "" {
		t.Fatal("expected session token")
	}
	if got.User.ID != "u-1" || got.User.Username != "admin" || got.User.Role != "super_admin" {
		t.Fatalf("unexpected user: %+v", got.User)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "login_success" {
		t.Fatalf("unexpected audit entries: %+v", audit.entries)
	}
}

func TestLoginWrongPasswordReturnsUnauthenticated(t *testing.T) {
	passwordHash, err := platformauth.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	repo := &loginRepoStub{
		user: domain.User{
			ID:           "u-1",
			Username:     "admin",
			Role:         "super_admin",
			PasswordHash: passwordHash,
		},
	}
	audit := &auditRecorderStub{}
	uc := usecase.NewLogin(repo, platformauth.NewSessionManager("test-secret"), audit)

	_, err = uc.Execute(context.Background(), usecase.LoginInput{
		Username: "admin",
		Password: "wrong",
	})
	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "login_failed" {
		t.Fatalf("unexpected audit entries: %+v", audit.entries)
	}
}
