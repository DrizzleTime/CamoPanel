package usecase_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	"camopanel/server/internal/modules/projects/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformdocker "camopanel/server/internal/platform/docker"
	"camopanel/server/internal/services"
)

type projectRepoStub struct {
	items map[string]projectsdomain.Project
}

func newProjectRepoStub() *projectRepoStub {
	return &projectRepoStub{items: map[string]projectsdomain.Project{}}
}

func (s *projectRepoStub) List(_ context.Context) ([]projectsdomain.Project, error) {
	items := make([]projectsdomain.Project, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *projectRepoStub) FindByID(_ context.Context, projectID string) (projectsdomain.Project, error) {
	item, ok := s.items[projectID]
	if !ok {
		return projectsdomain.Project{}, projectsdomain.ErrProjectNotFound
	}
	return item, nil
}

func (s *projectRepoStub) FindByName(_ context.Context, name string) (projectsdomain.Project, error) {
	for _, item := range s.items {
		if item.Name == name {
			return item, nil
		}
	}
	return projectsdomain.Project{}, projectsdomain.ErrProjectNotFound
}

func (s *projectRepoStub) CountByTemplateID(_ context.Context, templateID string) (int64, error) {
	var count int64
	for _, item := range s.items {
		if item.TemplateID == templateID {
			count++
		}
	}
	return count, nil
}

func (s *projectRepoStub) Create(_ context.Context, project projectsdomain.Project) error {
	s.items[project.ID] = project
	return nil
}

func (s *projectRepoStub) Save(_ context.Context, project projectsdomain.Project) error {
	s.items[project.ID] = project
	return nil
}

func (s *projectRepoStub) Delete(_ context.Context, projectID string) error {
	delete(s.items, projectID)
	return nil
}

type projectRuntimeStub struct {
	ensureNetworkCalls int
	deployCalls        int
	lastProject        string
	lastComposePath    string
	lastNetworkName    string
	lastNetworkDriver  string
	runtime            platformdocker.ProjectRuntime
}

func (s *projectRuntimeStub) EnsureNetwork(_ context.Context, name, driver string) error {
	s.ensureNetworkCalls++
	s.lastNetworkName = name
	s.lastNetworkDriver = driver
	return nil
}

func (s *projectRuntimeStub) Deploy(_ context.Context, projectName, composePath string) error {
	s.deployCalls++
	s.lastProject = projectName
	s.lastComposePath = composePath
	return nil
}

func (s *projectRuntimeStub) Start(_ context.Context, _, _ string) error    { return nil }
func (s *projectRuntimeStub) Stop(_ context.Context, _, _ string) error     { return nil }
func (s *projectRuntimeStub) Restart(_ context.Context, _, _ string) error  { return nil }
func (s *projectRuntimeStub) Redeploy(_ context.Context, _, _ string) error { return nil }
func (s *projectRuntimeStub) Delete(_ context.Context, _, _ string) error   { return nil }

func (s *projectRuntimeStub) InspectProject(_ context.Context, _ string) (platformdocker.ProjectRuntime, error) {
	return s.runtime, nil
}

func (s *projectRuntimeStub) ProjectLogs(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

type projectAuditStub struct {
	entries []platformaudit.Entry
}

func (s *projectAuditStub) Record(_ context.Context, entry platformaudit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestCreateProjectRendersComposeDeploysAndPersistsProject(t *testing.T) {
	root := t.TempDir()
	catalog := newTemplateCatalog(t, root)
	repo := newProjectRepoStub()
	runtime := &projectRuntimeStub{
		runtime: platformdocker.ProjectRuntime{
			Status: platformdocker.StatusRunning,
			Containers: []platformdocker.ProjectContainer{
				{Name: "demo-stack-app", State: "running"},
			},
		},
	}
	audit := &projectAuditStub{}
	uc := usecase.NewCreateProject(repo, catalog, runtime, audit, usecase.ProjectConfig{
		ProjectsDir:        filepath.Join(root, "projects"),
		BridgeNetworkName:  "camopanel",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	})

	got, err := uc.Execute(context.Background(), usecase.CreateProjectInput{
		ActorID:    "user-1",
		Name:       "demo-stack",
		TemplateID: "demo",
		Parameters: map[string]any{"port": 8080, "password": "secret"},
	})
	if err != nil {
		t.Fatalf("execute create project: %v", err)
	}

	if runtime.deployCalls != 1 {
		t.Fatalf("expected deploy once, got %d", runtime.deployCalls)
	}
	if runtime.ensureNetworkCalls != 1 {
		t.Fatalf("expected ensure network once, got %d", runtime.ensureNetworkCalls)
	}
	if runtime.lastProject != "demo-stack" {
		t.Fatalf("expected project name demo-stack, got %s", runtime.lastProject)
	}
	if got.Project.Name != "demo-stack" {
		t.Fatalf("expected stored project name demo-stack, got %s", got.Project.Name)
	}
	if got.Project.Status != platformdocker.StatusRunning {
		t.Fatalf("expected project status running, got %s", got.Project.Status)
	}

	rendered, err := os.ReadFile(runtime.lastComposePath)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	content := string(rendered)
	if !strings.Contains(content, `name: "camopanel"`) {
		t.Fatalf("expected rendered compose to include bridge network, got %s", content)
	}
	if !strings.Contains(content, `- "demo-stack"`) {
		t.Fatalf("expected rendered compose to include project alias, got %s", content)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "project_deploy" {
		t.Fatalf("unexpected audit entries: %+v", audit.entries)
	}
}

func TestCreateCustomProjectWritesComposeAndPersistsProject(t *testing.T) {
	root := t.TempDir()
	catalog := newTemplateCatalog(t, root)
	repo := newProjectRepoStub()
	runtime := &projectRuntimeStub{
		runtime: platformdocker.ProjectRuntime{Status: platformdocker.StatusRunning},
	}
	uc := usecase.NewCreateCustomProject(repo, catalog, runtime, &projectAuditStub{}, usecase.ProjectConfig{
		ProjectsDir:        filepath.Join(root, "projects"),
		BridgeNetworkName:  "camopanel",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	})

	got, err := uc.Execute(context.Background(), usecase.CreateCustomProjectInput{
		ActorID: "user-1",
		Name:    "custom-blog",
		Compose: "services:\n  app:\n    image: nginx:alpine",
	})
	if err != nil {
		t.Fatalf("execute create custom project: %v", err)
	}

	if got.Project.TemplateID != usecase.CustomComposeTemplateID {
		t.Fatalf("expected template id %s, got %s", usecase.CustomComposeTemplateID, got.Project.TemplateID)
	}
	rendered, err := os.ReadFile(runtime.lastComposePath)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	if string(rendered) != "services:\n  app:\n    image: nginx:alpine\n" {
		t.Fatalf("unexpected custom compose content: %s", string(rendered))
	}
}

func newTemplateCatalog(t *testing.T, root string) *services.TemplateCatalog {
	t.Helper()

	templatesDir := filepath.Join(root, "templates")
	writeTemplate(t, templatesDir, "demo", `id: demo
name: Demo Template
version: "1"
description: test
params:
  - name: port
    type: number
    required: true
  - name: password
    type: secret
    required: true
`, `services:
  app:
    image: nginx
    networks:
      camopanel:
        aliases:
          - "{{ .Runtime.ProjectName }}"
    ports:
      - "{{ .Values.port }}:80"
    environment:
      PASSWORD: {{ .Values.password }}
networks:
  camopanel:
    external: true
    name: "{{ .Runtime.BridgeNetworkName }}"
`)

	catalog, err := services.NewTemplateCatalog(templatesDir)
	if err != nil {
		t.Fatalf("new template catalog: %v", err)
	}
	return catalog
}

func writeTemplate(t *testing.T, root, id, spec, compose string) {
	t.Helper()

	templateDir := filepath.Join(root, id)
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "compose.yaml.tmpl"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}
}
