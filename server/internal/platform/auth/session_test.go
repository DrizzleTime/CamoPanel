package auth_test

import (
	"errors"
	"testing"

	authdomain "camopanel/server/internal/modules/auth/domain"
	platformauth "camopanel/server/internal/platform/auth"
	"camopanel/server/internal/services"
)

func TestSessionManager_ParseLegacyIssuedToken(t *testing.T) {
	t.Parallel()

	legacy := services.NewAuthService("compat-secret")
	token, err := legacy.IssueSession(authdomain.User{ID: "user-1"})
	if err != nil {
		t.Fatalf("issue legacy session: %v", err)
	}

	manager := platformauth.NewSessionManager("compat-secret")
	gotUserID, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if gotUserID != "user-1" {
		t.Fatalf("userID = %q, want %q", gotUserID, "user-1")
	}
}

func TestSessionManager_IssueTokenCanBeParsedByLegacy(t *testing.T) {
	t.Parallel()

	manager := platformauth.NewSessionManager("compat-secret")
	token, err := manager.Issue("user-2")
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}

	legacy := services.NewAuthService("compat-secret")
	gotUserID, err := legacy.ParseSession(token)
	if err != nil {
		t.Fatalf("legacy parse token: %v", err)
	}
	if gotUserID != "user-2" {
		t.Fatalf("userID = %q, want %q", gotUserID, "user-2")
	}
}

func TestSessionManager_ParseInvalidToken(t *testing.T) {
	t.Parallel()

	manager := platformauth.NewSessionManager("compat-secret")
	_, err := manager.Parse("invalid-token")
	if !errors.Is(err, platformauth.ErrInvalidSession) {
		t.Fatalf("err = %v, want %v", err, platformauth.ErrInvalidSession)
	}
}

func TestPasswordHasher(t *testing.T) {
	t.Parallel()

	hash, err := platformauth.HashPassword("pass-1234")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !platformauth.CheckPassword(hash, "pass-1234") {
		t.Fatalf("expected password to match")
	}
	if platformauth.CheckPassword(hash, "wrong") {
		t.Fatalf("expected wrong password not to match")
	}
}
