package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	projectsapi "camopanel/server/internal/modules/projects/api"
	projectsdomain "camopanel/server/internal/modules/projects/domain"
	"camopanel/server/internal/modules/projects/usecase"
	platformdocker "camopanel/server/internal/platform/docker"

	"github.com/gin-gonic/gin"
)

type createProjectStub struct{}

func (s *createProjectStub) Execute(_ context.Context, input usecase.CreateProjectInput) (usecase.CreateProjectOutput, error) {
	return usecase.CreateProjectOutput{
		Project: projectsdomain.Project{
			ID:              "project-1",
			Name:            input.Name,
			TemplateID:      input.TemplateID,
			TemplateVersion: "1",
			Config:          input.Parameters,
			ComposePath:     "/tmp/demo/compose.yaml",
			Status:          platformdocker.StatusRunning,
			CreatedAt:       time.Unix(1, 0),
			UpdatedAt:       time.Unix(1, 0),
		},
		Runtime: platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
	}, nil
}

type createCustomProjectStub struct{}

func (s *createCustomProjectStub) Execute(_ context.Context, input usecase.CreateCustomProjectInput) (usecase.CreateProjectOutput, error) {
	return usecase.CreateProjectOutput{
		Project: projectsdomain.Project{
			ID:          "project-2",
			Name:        input.Name,
			TemplateID:  usecase.CustomComposeTemplateID,
			ComposePath: "/tmp/custom/compose.yaml",
			Status:      platformdocker.StatusRunning,
			CreatedAt:   time.Unix(1, 0),
			UpdatedAt:   time.Unix(1, 0),
		},
		Runtime: platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
	}, nil
}

type listProjectsStub struct{}

func (s *listProjectsStub) Execute(_ context.Context) ([]usecase.ProjectView, error) {
	return []usecase.ProjectView{
		{
			ID:          "project-1",
			Name:        "demo",
			ComposePath: "/tmp/demo/compose.yaml",
			Status:      platformdocker.StatusRunning,
			Runtime:     platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
		},
	}, nil
}

type getProjectStub struct{}

func (s *getProjectStub) Execute(_ context.Context, projectID string) (usecase.ProjectView, error) {
	return usecase.ProjectView{
		ID:          projectID,
		Name:        "demo",
		ComposePath: "/tmp/demo/compose.yaml",
		Status:      platformdocker.StatusRunning,
		Runtime:     platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
	}, nil
}

type runActionStub struct{}

func (s *runActionStub) Execute(_ context.Context, _ usecase.RunActionInput) (usecase.RunActionOutput, error) {
	return usecase.RunActionOutput{
		Project: &usecase.ProjectView{
			ID:      "project-1",
			Name:    "demo",
			Status:  platformdocker.StatusRunning,
			Runtime: platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
		},
	}, nil
}

func (s *runActionStub) Logs(_ context.Context, _ string, _ int) (string, error) {
	return "demo logs", nil
}

type templateCatalogStub struct{}

func (s *templateCatalogStub) List() []projectsdomain.Template {
	return []projectsdomain.Template{{ID: "demo", Name: "Demo", Version: "1"}}
}

func (s *templateCatalogStub) Get(id string) (projectsdomain.Template, error) {
	return projectsdomain.Template{ID: id, Name: "Demo", Version: "1"}, nil
}

func TestHandlerCreateProjectReturnsCreatedProject(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := projectsapi.NewHandler(
		&templateCatalogStub{},
		&createProjectStub{},
		&createCustomProjectStub{},
		&listProjectsStub{},
		&getProjectStub{},
		&runActionStub{},
	)

	router := gin.New()
	api := router.Group("/api")
	projectsapi.NewModule(handler).RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewBufferString(`{"name":"demo","template_id":"demo","parameters":{"port":8080}}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}
