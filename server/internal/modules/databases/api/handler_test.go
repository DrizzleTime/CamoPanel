package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	databasesapi "camopanel/server/internal/modules/databases/api"
	databasesdomain "camopanel/server/internal/modules/databases/domain"
	platformdocker "camopanel/server/internal/platform/docker"

	"github.com/gin-gonic/gin"
)

type databaseServiceStub struct{}

func (s *databaseServiceStub) ListInstances(context.Context, string) ([]databasesdomain.InstanceView, error) {
	return []databasesdomain.InstanceView{{
		ID:        "db-1",
		Name:      "mysql-demo",
		Engine:    databasesdomain.EngineMySQL,
		Status:    "running",
		Runtime:   platformdocker.ProjectRuntime{Status: "running"},
		CreatedAt: time.Unix(1, 0).UTC().Format(time.RFC3339),
		UpdatedAt: time.Unix(1, 0).UTC().Format(time.RFC3339),
	}}, nil
}
func (s *databaseServiceStub) GetOverview(context.Context, string) (databasesdomain.Overview, error) {
	return databasesdomain.Overview{}, nil
}
func (s *databaseServiceStub) CreateDatabase(context.Context, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) DeleteDatabase(context.Context, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) CreateAccount(context.Context, string, string, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) DeleteAccount(context.Context, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) UpdateAccountPassword(context.Context, string, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) GrantAccount(context.Context, string, string, string, string) error {
	return nil
}
func (s *databaseServiceStub) UpdateRedisConfig(context.Context, string, string, string, any) error {
	return nil
}

func TestHandlerListInstances(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api")
	databasesapi.NewModule(databasesapi.NewHandler(&databaseServiceStub{})).RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodGet, "/api/databases?engine=mysql", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
