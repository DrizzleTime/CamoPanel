package usecase

import (
	"context"
	"errors"

	"camopanel/server/internal/modules/auth/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformauth "camopanel/server/internal/platform/auth"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}

type SessionIssuer interface {
	Issue(userID string) (string, error)
}

type AuditRecorder interface {
	Record(ctx context.Context, entry platformaudit.Entry) error
}

type LoginInput struct {
	Username string
	Password string
}

type UserView struct {
	ID       string
	Username string
	Role     string
}

type LoginOutput struct {
	Token string
	User  UserView
}

type Login struct {
	repo     UserRepository
	sessions SessionIssuer
	audit    AuditRecorder
}

func NewLogin(repo UserRepository, sessions SessionIssuer, audit AuditRecorder) *Login {
	return &Login{
		repo:     repo,
		sessions: sessions,
		audit:    audit,
	}
}

func (u *Login) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := u.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			_ = u.audit.Record(ctx, platformaudit.Entry{
				Action:     "login_failed",
				TargetType: "user",
				TargetID:   input.Username,
				Metadata:   map[string]any{"reason": "user_not_found"},
			})
			return LoginOutput{}, ErrInvalidCredentials
		}
		return LoginOutput{}, err
	}

	if !platformauth.CheckPassword(user.PasswordHash, input.Password) {
		_ = u.audit.Record(ctx, platformaudit.Entry{
			ActorID:    user.ID,
			Action:     "login_failed",
			TargetType: "user",
			TargetID:   user.ID,
			Metadata:   map[string]any{"reason": "invalid_password"},
		})
		return LoginOutput{}, ErrInvalidCredentials
	}

	token, err := u.sessions.Issue(user.ID)
	if err != nil {
		return LoginOutput{}, err
	}

	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    user.ID,
		Action:     "login_success",
		TargetType: "user",
		TargetID:   user.ID,
	})

	return LoginOutput{
		Token: token,
		User: UserView{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}
