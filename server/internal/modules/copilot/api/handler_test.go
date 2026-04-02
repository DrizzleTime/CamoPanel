package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	copilotapi "camopanel/server/internal/modules/copilot/api"
	copilotusecase "camopanel/server/internal/modules/copilot/usecase"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type chatStub struct{}

func (s *chatStub) CreateSession() services.CopilotSession {
	return services.CopilotSession{ID: "session-1"}
}
func (s *chatStub) Reply(context.Context, string, string) (services.CopilotReply, error) {
	return services.CopilotReply{Message: "ok"}, nil
}

type configStub struct{}

func (s *configStub) ListProviders(context.Context) ([]copilotusecase.ProviderView, error) {
	return nil, nil
}
func (s *configStub) CreateProvider(context.Context, string, string, string, string, bool) (copilotusecase.ProviderView, error) {
	return copilotusecase.ProviderView{}, nil
}
func (s *configStub) UpdateProvider(context.Context, string, string, string, string, string, bool) (copilotusecase.ProviderView, error) {
	return copilotusecase.ProviderView{}, nil
}
func (s *configStub) DeleteProvider(context.Context, string) error { return nil }
func (s *configStub) CreateModel(context.Context, string, string, bool, bool) (copilotusecase.ModelView, error) {
	return copilotusecase.ModelView{}, nil
}
func (s *configStub) UpdateModel(context.Context, string, string, bool, bool) (copilotusecase.ModelView, error) {
	return copilotusecase.ModelView{}, nil
}
func (s *configStub) DeleteModel(context.Context, string) error { return nil }
func (s *configStub) ConfigStatus(context.Context) (copilotusecase.ConfigStatus, error) {
	return copilotusecase.ConfigStatus{Configured: true, Source: "env"}, nil
}

func TestCreateSessionRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")
	copilotapi.NewModule(copilotapi.NewHandler(&chatStub{}, &configStub{})).RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodPost, "/api/copilot/sessions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}
