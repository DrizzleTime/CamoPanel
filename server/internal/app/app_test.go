package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"camopanel/server/internal/config"
	"camopanel/server/internal/model"
	"camopanel/server/internal/services"
)

type fakeExecutor struct {
	deployCalls   int
	restartCalls  int
	deleteCalls   int
	lastProject   string
	lastCompose   string
	runtimeStatus string
	logs          string
}

func (f *fakeExecutor) Deploy(_ context.Context, projectName, composePath string) error {
	f.deployCalls++
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) Start(_ context.Context, projectName, composePath string) error {
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) Stop(_ context.Context, projectName, composePath string) error {
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) Restart(_ context.Context, projectName, composePath string) error {
	f.restartCalls++
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) Redeploy(_ context.Context, projectName, composePath string) error {
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) Delete(_ context.Context, projectName, composePath string) error {
	f.deleteCalls++
	f.lastProject = projectName
	f.lastCompose = composePath
	return nil
}

func (f *fakeExecutor) InspectProject(_ context.Context, projectName string) (services.ProjectRuntime, error) {
	return services.ProjectRuntime{
		Status: f.runtimeStatus,
		Containers: []services.ProjectContainer{
			{Name: projectName + "-app", State: f.runtimeStatus},
		},
	}, nil
}

func (f *fakeExecutor) ProjectLogs(_ context.Context, _ string, _ int) (string, error) {
	return f.logs, nil
}

type fakeOpenResty struct {
	ready       bool
	createCalls int
	lastSpec    services.WebsiteSpec
}

func (f *fakeOpenResty) Status(_ context.Context) services.OpenRestyStatus {
	return services.OpenRestyStatus{
		Exists:          f.ready,
		Ready:           f.ready,
		ContainerName:   "camopanel-openresty",
		ContainerStatus: "running",
		HostConfigDir:   "/tmp/conf.d",
		HostSiteDir:     "/tmp/www",
		Message:         "ok",
	}
}

func (f *fakeOpenResty) EnsureReady(_ context.Context) error {
	if !f.ready {
		return services.ErrOpenRestyUnavailable
	}
	return nil
}

func (f *fakeOpenResty) CreateWebsite(_ context.Context, spec services.WebsiteSpec) (services.WebsiteMaterialized, error) {
	if !f.ready {
		return services.WebsiteMaterialized{}, services.ErrOpenRestyUnavailable
	}
	f.createCalls++
	f.lastSpec = spec
	return services.WebsiteMaterialized{
		RootPath:   filepath.Join("/tmp", spec.Name, "html"),
		ConfigPath: filepath.Join("/tmp", spec.Name+".conf"),
	}, nil
}

func TestDeployApprovalLifecycle(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	approval, err := instance.createDeployApproval("tester", "ui", createProjectRequest{
		Name:       "demo-stack",
		TemplateID: "demo",
		Parameters: map[string]any{"port": 8080, "password": "secret"},
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	approved, err := instance.approveRequest(context.Background(), approval.ID, "tester")
	if err != nil {
		t.Fatalf("approve request: %v", err)
	}

	if approved.Status != model.ApprovalStatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}
	if executor.deployCalls != 1 {
		t.Fatalf("expected deploy call once, got %d", executor.deployCalls)
	}

	var count int64
	if err := instance.db.Model(&model.Project{}).Count(&count).Error; err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 project, got %d", count)
	}
}

func TestAIProposalCreatesApproval(t *testing.T) {
	instance := newTestApp(t)

	approval, err := instance.createApprovalFromProposal("tester", &services.ProposedAction{
		Action:      model.ApprovalActionDeploy,
		TemplateID:  "demo",
		ProjectName: "from-ai",
		Parameters: map[string]any{
			"port":     8081,
			"password": "secret",
		},
	})
	if err != nil {
		t.Fatalf("create approval from proposal: %v", err)
	}

	if approval.Source != "ai" {
		t.Fatalf("expected source ai, got %s", approval.Source)
	}
	if approval.Status != model.ApprovalStatusPending {
		t.Fatalf("expected pending status, got %s", approval.Status)
	}
}

func TestRejectApproval(t *testing.T) {
	instance := newTestApp(t)

	approval, err := instance.createDeployApproval("tester", "ui", createProjectRequest{
		Name:       "reject-me",
		TemplateID: "demo",
		Parameters: map[string]any{"port": 8080, "password": "secret"},
	})
	if err != nil {
		t.Fatalf("create approval: %v", err)
	}

	rejected, err := instance.rejectRequest(approval.ID, "tester", "manual reject")
	if err != nil {
		t.Fatalf("reject request: %v", err)
	}
	if rejected.Status != model.ApprovalStatusRejected {
		t.Fatalf("expected rejected status, got %s", rejected.Status)
	}
}

func TestCreateWebsiteApprovalLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	approval, err := instance.createWebsiteApproval(context.Background(), "tester", "ui", createWebsiteRequest{
		Name:      "demo-site",
		Type:      model.WebsiteTypeProxy,
		Domain:    "demo.local",
		ProxyPass: "http://127.0.0.1:3000",
	})
	if err != nil {
		t.Fatalf("create website approval: %v", err)
	}

	approved, err := instance.approveRequest(context.Background(), approval.ID, "tester")
	if err != nil {
		t.Fatalf("approve website request: %v", err)
	}
	if approved.Status != model.ApprovalStatusApproved {
		t.Fatalf("expected approved status, got %s", approved.Status)
	}
	if openresty.createCalls != 1 {
		t.Fatalf("expected create website call once, got %d", openresty.createCalls)
	}

	var count int64
	if err := instance.db.Model(&model.Website{}).Count(&count).Error; err != nil {
		t.Fatalf("count websites: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 website, got %d", count)
	}
}

func TestCreateWebsiteApprovalRequiresOpenResty(t *testing.T) {
	instance := newTestApp(t)
	instance.openresty = &fakeOpenResty{ready: false}

	_, err := instance.createWebsiteApproval(context.Background(), "tester", "ui", createWebsiteRequest{
		Name:   "demo-site",
		Type:   model.WebsiteTypeStatic,
		Domain: "demo.local",
	})
	if err == nil {
		t.Fatalf("expected openresty availability error")
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	root := t.TempDir()
	templatesDir := filepath.Join(root, "templates")
	templateDir := filepath.Join(templatesDir, "demo")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}

	spec := `id: demo
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
`
	compose := `services:
  app:
    image: nginx
    ports:
      - "{{ .Values.port }}:80"
    environment:
      PASSWORD: {{ .Values.password }}
`
	if err := os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "compose.yaml.tmpl"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	cfg := config.Config{
		HTTPAddr:           ":0",
		DataDir:            root,
		DatabasePath:       filepath.Join(root, "camopanel.db"),
		ProjectsDir:        filepath.Join(root, "projects"),
		TemplatesDir:       templatesDir,
		SessionSecret:      "test-secret",
		CookieName:         "test-cookie",
		AdminUsername:      "admin",
		AdminPassword:      "admin123",
		OpenRestyContainer: "camopanel-openresty",
		OpenRestyDataDir:   filepath.Join(root, "openresty"),
	}

	instance, err := New(cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	instance.openresty = &fakeOpenResty{ready: true}
	return instance
}
