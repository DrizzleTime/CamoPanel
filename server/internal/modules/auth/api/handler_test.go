package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authapi "camopanel/server/internal/modules/auth/api"
	"camopanel/server/internal/modules/auth/domain"
	"camopanel/server/internal/modules/auth/usecase"
	platformauth "camopanel/server/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

type loginUsecaseStub struct {
	output usecase.LoginOutput
	err    error
	input  usecase.LoginInput
}

func (s *loginUsecaseStub) Execute(_ context.Context, input usecase.LoginInput) (usecase.LoginOutput, error) {
	s.input = input
	if s.err != nil {
		return usecase.LoginOutput{}, s.err
	}
	return s.output, nil
}

type meUsecaseStub struct {
	user domain.User
	err  error
}

func (s *meUsecaseStub) Execute(_ context.Context, userID string) (domain.User, error) {
	if s.err != nil {
		return domain.User{}, s.err
	}
	if s.user.ID != userID {
		return domain.User{}, domain.ErrUserNotFound
	}
	return s.user, nil
}

func TestLoginRouteSetsCookieAndReturnsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loginUC := &loginUsecaseStub{
		output: usecase.LoginOutput{
			Token: "signed-token",
			User: usecase.UserView{
				ID:       "u-1",
				Username: "admin",
				Role:     "super_admin",
			},
		},
	}
	meUC := &meUsecaseStub{}

	router := gin.New()
	handler := authapi.NewHandler("camopanel_session", platformauth.NewSessionManager("test-secret"), loginUC, meUC)
	authapi.RegisterRoutes(router, handler)

	body, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "secret",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" {
		t.Fatal("expected Set-Cookie header")
	}
}

func TestMeRouteReturnsCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessions := platformauth.NewSessionManager("test-secret")
	token, err := sessions.Issue("u-1")
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	loginUC := &loginUsecaseStub{}
	meUC := &meUsecaseStub{
		user: domain.User{
			ID:       "u-1",
			Username: "admin",
			Role:     "super_admin",
		},
	}

	router := gin.New()
	handler := authapi.NewHandler("camopanel_session", sessions, loginUC, meUC)
	authapi.RegisterRoutes(router, handler)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "camopanel_session", Value: token, Path: "/"})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
}

func TestLogoutRouteClearsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	handler := authapi.NewHandler("camopanel_session", platformauth.NewSessionManager("test-secret"), &loginUsecaseStub{}, &meUsecaseStub{})
	authapi.RegisterRoutes(router, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Set-Cookie"); got == "" {
		t.Fatal("expected Set-Cookie header")
	}
}
