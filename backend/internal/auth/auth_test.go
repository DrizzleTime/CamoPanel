package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEnvironmentCredentialsOverrideLegacyAndAreNotPersisted(t *testing.T) {
	dir := t.TempDir()
	old, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Bootstrap("legacy-password"); err != nil {
		t.Fatal(err)
	}
	if err := old.SetViewer("legacy-viewer-password"); err != nil {
		t.Fatal(err)
	}
	store, err := OpenWithCredentials(dir, "operator", "environment-password")
	if err != nil {
		t.Fatal(err)
	}
	if !store.HasAdmin() || !store.EnvironmentManaged() {
		t.Fatal("environment credentials did not enable authentication")
	}
	for _, login := range []struct{ user, pass string }{{"operator", "wrong-password"}, {"admin", "environment-password"}, {"admin", "legacy-password"}, {"viewer", "legacy-viewer-password"}, {"", "environment-password"}} {
		if _, err := store.Login(login.user, login.pass, "", time.Now()); !errors.Is(err, ErrBadPassword) {
			t.Fatalf("unexpected authentication for %q: %v", login.user, err)
		}
	}
	session, err := store.Login("operator", "environment-password", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if session.Username != "operator" || session.Role != "admin" || session.ExpiresAt.IsZero() {
		t.Fatalf("invalid session: %+v", session)
	}
	if _, ok := store.Lookup(session.ID); !ok {
		t.Fatal("session unavailable")
	}
	for _, operation := range []func() error{func() error { return store.Bootstrap("another-password") }, func() error { return store.ChangePassword("operator", "environment-password", "another-password") }, func() error { return store.SetViewer("viewer-password") }} {
		if !errors.Is(operation(), ErrEnvironment) {
			t.Fatal("environment credentials were modified through API")
		}
	}
	secret, err := store.BeginTOTP()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "environment-password") || strings.Contains(string(raw), store.envAdmin.PasswordHash) || strings.Contains(string(raw), "$2") {
		t.Fatal("environment credential hash persisted to disk")
	}
	code, err := CodeAt(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.EnableTOTP(code, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Login("operator", "environment-password", "", time.Now()); !errors.Is(err, ErrTOTP) {
		t.Fatal("TOTP was not enforced")
	}
	session, err = store.Login("operator", "environment-password", code, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	store.Logout(session.ID)
	if _, ok := store.Lookup(session.ID); ok {
		t.Fatal("logged-out session remained valid")
	}
	restarted, err := OpenWithCredentials(dir, "operator", "rotated-environment-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Login("operator", "environment-password", code, time.Now()); !errors.Is(err, ErrBadPassword) {
		t.Fatal("old environment password remained valid")
	}
	if _, err := restarted.Login("operator", "rotated-environment-password", code, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, ok := restarted.Lookup(session.ID); ok {
		t.Fatal("session survived restart")
	}
}

func TestSessionExpiry(t *testing.T) {
	store, err := OpenWithCredentials(t.TempDir(), "admin", "test-password")
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.Login("admin", "test-password", "", time.Now().Add(-25*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Lookup(session.ID); ok {
		t.Fatal("expired session accepted")
	}
}
