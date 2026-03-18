package app

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"camopanel/server/internal/config"
	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

type fakeExecutor struct {
	deployCalls        int
	restartCalls       int
	deleteCalls        int
	ensureNetworkCalls int
	lastProject        string
	lastCompose        string
	lastNetworkName    string
	lastNetworkDriver  string
	runtimeStatus      string
	logs               string
}

func (f *fakeExecutor) EnsureNetwork(_ context.Context, name, driver string) error {
	f.ensureNetworkCalls++
	f.lastNetworkName = name
	f.lastNetworkDriver = driver
	return nil
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
	ready                  bool
	createCalls            int
	updateCalls            int
	deleteCalls            int
	syncCalls              int
	issueCertificateCalls  int
	deleteCertificateCalls int
	lastSpec               services.WebsiteSpec
	lastConfig             string
	lastCertificate        services.CertificateSpec
}

type fakeDockerReader struct {
	startCalls       int
	stopCalls        int
	restartCalls     int
	deleteCalls      int
	removeImageCalls int
	pruneImageCalls  int
	lastID           string
}

func (f *fakeDockerReader) ListContainers(_ context.Context) ([]services.DockerContainer, error) {
	return nil, nil
}

func (f *fakeDockerReader) ListImages(_ context.Context) ([]services.DockerImage, error) {
	return nil, nil
}

func (f *fakeDockerReader) ListNetworks(_ context.Context) ([]services.DockerNetwork, error) {
	return nil, nil
}

func (f *fakeDockerReader) GetSystemInfo(_ context.Context) (services.DockerSystemInfo, error) {
	return services.DockerSystemInfo{}, nil
}

func (f *fakeDockerReader) ContainerLogs(_ context.Context, _ string, _ int) (string, error) {
	return "", nil
}

func (f *fakeDockerReader) StartContainer(_ context.Context, containerID string) error {
	f.startCalls++
	f.lastID = containerID
	return nil
}

func (f *fakeDockerReader) StopContainer(_ context.Context, containerID string) error {
	f.stopCalls++
	f.lastID = containerID
	return nil
}

func (f *fakeDockerReader) RestartContainer(_ context.Context, containerID string) error {
	f.restartCalls++
	f.lastID = containerID
	return nil
}

func (f *fakeDockerReader) DeleteContainer(_ context.Context, containerID string) error {
	f.deleteCalls++
	f.lastID = containerID
	return nil
}

func (f *fakeDockerReader) RemoveImage(_ context.Context, imageID string) error {
	f.removeImageCalls++
	f.lastID = imageID
	return nil
}

func (f *fakeDockerReader) PruneUnusedImages(_ context.Context) (services.DockerImagePruneResult, error) {
	f.pruneImageCalls++
	return services.DockerImagePruneResult{ImagesDeleted: 2, SpaceReclaimed: 1024}, nil
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

func (f *fakeOpenResty) UpdateWebsite(_ context.Context, spec services.WebsiteSpec, configPath string) (services.WebsiteMaterialized, error) {
	if !f.ready {
		return services.WebsiteMaterialized{}, services.ErrOpenRestyUnavailable
	}
	f.updateCalls++
	f.lastSpec = spec
	f.lastConfig = configPath
	return services.WebsiteMaterialized{
		RootPath:   spec.RootPath,
		ConfigPath: configPath,
	}, nil
}

func (f *fakeOpenResty) SyncWebsite(_ context.Context, spec services.WebsiteSpec) (services.WebsiteMaterialized, error) {
	if !f.ready {
		return services.WebsiteMaterialized{}, services.ErrOpenRestyUnavailable
	}
	f.syncCalls++
	f.lastSpec = spec
	return services.WebsiteMaterialized{
		RootPath:   spec.RootPath,
		ConfigPath: filepath.Join("/tmp", spec.Name+".conf"),
	}, nil
}

func (f *fakeOpenResty) DeleteWebsite(_ context.Context, configPath string) error {
	if !f.ready {
		return services.ErrOpenRestyUnavailable
	}
	f.deleteCalls++
	f.lastConfig = configPath
	return nil
}

func (f *fakeOpenResty) PreviewWebsiteConfig(spec services.WebsiteSpec) (string, error) {
	return "server_name " + spec.Domain + ";", nil
}

func (f *fakeOpenResty) IssueCertificate(_ context.Context, spec services.CertificateSpec) (services.CertificateMaterialized, error) {
	if !f.ready {
		return services.CertificateMaterialized{}, services.ErrOpenRestyUnavailable
	}
	f.issueCertificateCalls++
	f.lastCertificate = spec
	return services.CertificateMaterialized{
		Provider:       "letsencrypt",
		FullchainPath:  filepath.Join("/tmp", "certs", spec.Domain, "fullchain.pem"),
		PrivateKeyPath: filepath.Join("/tmp", "certs", spec.Domain, "privkey.pem"),
		ExpiresAt:      time.Now().UTC().Add(90 * 24 * time.Hour),
	}, nil
}

func (f *fakeOpenResty) DeleteCertificate(_ context.Context, domain string) error {
	if !f.ready {
		return services.ErrOpenRestyUnavailable
	}
	f.deleteCertificateCalls++
	f.lastCertificate = services.CertificateSpec{Domain: domain}
	return nil
}

func TestCreateProjectLifecycle(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	project, err := instance.createProject(context.Background(), "tester", createProjectRequest{
		Name:       "demo-stack",
		TemplateID: "demo",
		Parameters: map[string]any{"port": 8080, "password": "secret"},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if executor.deployCalls != 1 {
		t.Fatalf("expected deploy call once, got %d", executor.deployCalls)
	}
	if executor.ensureNetworkCalls != 1 {
		t.Fatalf("expected ensure network call once, got %d", executor.ensureNetworkCalls)
	}
	if executor.lastNetworkName != "camopanel" {
		t.Fatalf("expected bridge network camopanel, got %s", executor.lastNetworkName)
	}
	if executor.lastNetworkDriver != "bridge" {
		t.Fatalf("expected bridge network driver, got %s", executor.lastNetworkDriver)
	}
	if project.Name != "demo-stack" {
		t.Fatalf("expected project demo-stack, got %s", project.Name)
	}

	rendered, err := os.ReadFile(executor.lastCompose)
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

	var count int64
	if err := instance.db.Model(&model.Project{}).Count(&count).Error; err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 project, got %d", count)
	}
}

func TestCreateCustomProjectLifecycle(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	project, err := instance.createCustomProject(context.Background(), "tester", createCustomProjectRequest{
		Name:    "custom-blog",
		Compose: "services:\n  app:\n    image: nginx:alpine",
	})
	if err != nil {
		t.Fatalf("create custom project: %v", err)
	}

	if executor.deployCalls != 1 {
		t.Fatalf("expected deploy call once, got %d", executor.deployCalls)
	}
	if project.TemplateID != customComposeTemplateID {
		t.Fatalf("expected template id %s, got %s", customComposeTemplateID, project.TemplateID)
	}

	rendered, err := os.ReadFile(executor.lastCompose)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}
	if string(rendered) != "services:\n  app:\n    image: nginx:alpine\n" {
		t.Fatalf("unexpected custom compose content: %s", string(rendered))
	}
}

func TestRunProjectActionDeletesProject(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	project, err := instance.createProject(context.Background(), "tester", createProjectRequest{
		Name:       "delete-me",
		TemplateID: "demo",
		Parameters: map[string]any{"port": 8080, "password": "secret"},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	if err := instance.runProjectAction(context.Background(), "tester", project, model.ActionDelete); err != nil {
		t.Fatalf("run delete action: %v", err)
	}

	if executor.deleteCalls != 1 {
		t.Fatalf("expected delete call once, got %d", executor.deleteCalls)
	}

	var count int64
	if err := instance.db.Model(&model.Project{}).Count(&count).Error; err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 projects, got %d", count)
	}
}

func TestCreateWebsiteLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	website, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:      "demo-site",
		Type:      model.WebsiteTypeProxy,
		Domain:    "demo.local",
		ProxyPass: "http://127.0.0.1:3000",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}
	if openresty.createCalls != 1 {
		t.Fatalf("expected create website call once, got %d", openresty.createCalls)
	}
	if website.Domain != "demo.local" {
		t.Fatalf("expected website domain demo.local, got %s", website.Domain)
	}

	websites, err := instance.listWebsites()
	if err != nil {
		t.Fatalf("list websites: %v", err)
	}
	if len(websites) != 1 {
		t.Fatalf("expected 1 website, got %d", len(websites))
	}
}

func TestCreateWebsiteRequiresOpenResty(t *testing.T) {
	instance := newTestApp(t)
	instance.openresty = &fakeOpenResty{ready: false}

	_, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:   "demo-site",
		Type:   model.WebsiteTypeStatic,
		Domain: "demo.local",
	})
	if err == nil {
		t.Fatalf("expected openresty availability error")
	}
}

func TestCreateCertificateLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	_, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:        "demo-site",
		Type:        model.WebsiteTypeStatic,
		Domain:      "demo.local",
		IndexFiles:  "index.html index.htm",
		RewriteMode: "off",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}

	item, err := instance.createCertificate(context.Background(), "tester", createCertificateRequest{
		Domain: "demo.local",
		Email:  "admin@example.com",
	})
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	if openresty.issueCertificateCalls != 1 {
		t.Fatalf("expected issue certificate call once, got %d", openresty.issueCertificateCalls)
	}
	if openresty.syncCalls != 2 {
		t.Fatalf("expected sync website call twice, got %d", openresty.syncCalls)
	}
	if item.Domain != "demo.local" {
		t.Fatalf("expected certificate domain demo.local, got %s", item.Domain)
	}

	var count int64
	if err := instance.db.Model(&model.Certificate{}).Count(&count).Error; err != nil {
		t.Fatalf("count certificates: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 certificate, got %d", count)
	}
}

func TestDeleteCertificateLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	website, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:        "demo-site",
		Type:        model.WebsiteTypeStatic,
		Domain:      "demo.local",
		IndexFiles:  "index.html index.htm",
		RewriteMode: "off",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}
	if _, err := instance.createCertificate(context.Background(), "tester", createCertificateRequest{
		Domain: "demo.local",
		Email:  "admin@example.com",
	}); err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certificate, err := instance.findCertificateByDomain(website.Domain)
	if err != nil {
		t.Fatalf("find certificate: %v", err)
	}
	if err := instance.deleteCertificate(context.Background(), "tester", certificate); err != nil {
		t.Fatalf("delete certificate: %v", err)
	}
	if openresty.deleteCertificateCalls != 1 {
		t.Fatalf("expected delete certificate call once, got %d", openresty.deleteCertificateCalls)
	}

	var count int64
	if err := instance.db.Model(&model.Certificate{}).Count(&count).Error; err != nil {
		t.Fatalf("count certificates: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 certificates, got %d", count)
	}
}

func TestUpdateWebsiteLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	website, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:   "demo-site",
		Type:   model.WebsiteTypeStatic,
		Domain: "demo.local",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}

	updated, err := instance.updateWebsite(context.Background(), "tester", website, updateWebsiteRequest{
		Name:        "demo-site",
		Type:        model.WebsiteTypeProxy,
		Domain:      "updated.local",
		Domains:     []string{"www.updated.local"},
		ProxyPass:   "http://127.0.0.1:3000",
		RewriteMode: "off",
		IndexFiles:  "index.html index.htm",
	})
	if err != nil {
		t.Fatalf("update website: %v", err)
	}
	if openresty.updateCalls != 1 {
		t.Fatalf("expected update website call once, got %d", openresty.updateCalls)
	}
	if updated.Domain != "updated.local" {
		t.Fatalf("expected updated domain, got %s", updated.Domain)
	}
	if updated.Type != model.WebsiteTypeProxy {
		t.Fatalf("expected updated type proxy, got %s", updated.Type)
	}
}

func TestDeleteWebsiteLifecycle(t *testing.T) {
	instance := newTestApp(t)
	openresty := &fakeOpenResty{ready: true}
	instance.openresty = openresty

	website, err := instance.createWebsite(context.Background(), "tester", createWebsiteRequest{
		Name:   "demo-site",
		Type:   model.WebsiteTypeStatic,
		Domain: "demo.local",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}

	if err := instance.deleteWebsite(context.Background(), "tester", website); err != nil {
		t.Fatalf("delete website: %v", err)
	}
	if openresty.deleteCalls != 1 {
		t.Fatalf("expected delete website call once, got %d", openresty.deleteCalls)
	}

	websites, err := instance.listWebsites()
	if err != nil {
		t.Fatalf("list websites: %v", err)
	}
	if len(websites) != 0 {
		t.Fatalf("expected 0 websites, got %d", len(websites))
	}
}

func TestCopilotProviderAndModelLifecycle(t *testing.T) {
	instance := newTestApp(t)

	provider, err := instance.createCopilotProvider("tester", createCopilotProviderRequest{
		Name:    "OpenAI",
		Type:    "openai",
		BaseURL: "https://api.example.com/v1",
		APIKey:  "secret-key",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}
	if provider.APIKeyMasked == "" {
		t.Fatalf("expected masked api key")
	}

	providerModel, err := instance.findCopilotProvider(provider.ID)
	if err != nil {
		t.Fatalf("find provider: %v", err)
	}

	aiModel, err := instance.createCopilotModel("tester", providerModel, createCopilotModelRequest{
		Name:      "gpt-5",
		Enabled:   true,
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	if !aiModel.IsDefault {
		t.Fatalf("expected model to be default")
	}

	items, err := instance.listCopilotProviderResponses()
	if err != nil {
		t.Fatalf("list providers: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(items))
	}
	if len(items[0].Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(items[0].Models))
	}

	runtimeConfig, err := instance.ResolveCopilotRuntimeConfig(context.Background())
	if err != nil {
		t.Fatalf("resolve copilot runtime config: %v", err)
	}
	if runtimeConfig.Source != "database" {
		t.Fatalf("expected source database, got %s", runtimeConfig.Source)
	}
	if runtimeConfig.Model != "gpt-5" {
		t.Fatalf("expected model gpt-5, got %s", runtimeConfig.Model)
	}
	if runtimeConfig.ProviderName != "OpenAI" {
		t.Fatalf("expected provider OpenAI, got %s", runtimeConfig.ProviderName)
	}
}

func TestCopilotConfigStatusFallsBackToEnv(t *testing.T) {
	instance := newTestApp(t)
	instance.cfg.AI.BaseURL = "https://env.example.com/v1"
	instance.cfg.AI.Model = "gpt-env"
	instance.cfg.AI.APIKey = "env-secret"

	status, err := instance.copilotConfigStatus(context.Background())
	if err != nil {
		t.Fatalf("copilot config status: %v", err)
	}
	if !status.Configured {
		t.Fatalf("expected configured status")
	}
	if status.Source != "env" {
		t.Fatalf("expected source env, got %s", status.Source)
	}
	if status.ModelName != "gpt-env" {
		t.Fatalf("expected model gpt-env, got %s", status.ModelName)
	}
}

func TestHandleDockerContainerActionStart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	instance := newTestApp(t)
	docker := &fakeDockerReader{}
	instance.docker = docker

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/docker/containers/demo/actions", bytes.NewBufferString(`{"action":"start"}`))
	ctx.Params = gin.Params{{Key: "id", Value: "demo"}}

	instance.handleDockerContainerAction(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if docker.startCalls != 1 {
		t.Fatalf("expected start call once, got %d", docker.startCalls)
	}
	if docker.lastID != "demo" {
		t.Fatalf("expected container id demo, got %s", docker.lastID)
	}
}

func TestHandleDockerContainerActionDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)

	instance := newTestApp(t)
	docker := &fakeDockerReader{}
	instance.docker = docker

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/docker/containers/demo/actions", bytes.NewBufferString(`{"action":"delete"}`))
	ctx.Params = gin.Params{{Key: "id", Value: "demo"}}

	instance.handleDockerContainerAction(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if docker.deleteCalls != 1 {
		t.Fatalf("expected delete call once, got %d", docker.deleteCalls)
	}
}

func TestHandleDockerContainerActionRejectsUnsupportedAction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	instance := newTestApp(t)
	instance.docker = &fakeDockerReader{}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/docker/containers/demo/actions", bytes.NewBufferString(`{"action":"redeploy"}`))
	ctx.Params = gin.Params{{Key: "id", Value: "demo"}}

	instance.handleDockerContainerAction(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestManagedOpenRestyDeployUsesFixedProjectAndRuntimePaths(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	project, err := instance.createProject(context.Background(), "tester", createProjectRequest{
		Name:       "anything",
		TemplateID: managedOpenRestyTemplateID,
		Parameters: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	if executor.lastProject != managedOpenRestyProjectID {
		t.Fatalf("expected fixed project name %s, got %s", managedOpenRestyProjectID, executor.lastProject)
	}
	if project.Name != managedOpenRestyProjectID {
		t.Fatalf("expected project name %s, got %s", managedOpenRestyProjectID, project.Name)
	}

	rendered, err := os.ReadFile(executor.lastCompose)
	if err != nil {
		t.Fatalf("read compose: %v", err)
	}

	confDir := filepath.Join(instance.cfg.OpenRestyDataDir, "conf.d")
	siteDir := filepath.Join(instance.cfg.OpenRestyDataDir, "www")
	content := string(rendered)
	if !strings.Contains(content, "container_name: camopanel-openresty") {
		t.Fatalf("expected fixed container name, got %s", content)
	}
	if executor.ensureNetworkCalls != 0 {
		t.Fatalf("expected managed openresty to skip bridge network ensure, got %d", executor.ensureNetworkCalls)
	}
	if !strings.Contains(content, "network_mode: host") {
		t.Fatalf("expected host network mode, got %s", content)
	}
	if !strings.Contains(content, confDir+":/etc/nginx/conf.d") {
		t.Fatalf("expected nginx conf bind, got %s", content)
	}
	if !strings.Contains(content, confDir+":/etc/openresty/conf.d") {
		t.Fatalf("expected compatibility conf bind, got %s", content)
	}
	if !strings.Contains(content, siteDir+":/var/www/openresty") {
		t.Fatalf("expected site dir bind, got %s", content)
	}
	certDir := filepath.Join(instance.cfg.OpenRestyDataDir, "certs")
	if !strings.Contains(content, certDir+":/etc/camopanel/certs") {
		t.Fatalf("expected cert dir bind, got %s", content)
	}
}

func TestManagedOpenRestyOnlyAllowsSingleDeployment(t *testing.T) {
	instance := newTestApp(t)
	executor := &fakeExecutor{runtimeStatus: "running"}
	instance.executor = executor

	_, err := instance.createProject(context.Background(), "tester", createProjectRequest{
		Name:       managedOpenRestyProjectID,
		TemplateID: managedOpenRestyTemplateID,
		Parameters: map[string]any{},
	})
	if err != nil {
		t.Fatalf("create first project: %v", err)
	}

	_, err = instance.createProject(context.Background(), "tester", createProjectRequest{
		Name:       "another-openresty",
		TemplateID: managedOpenRestyTemplateID,
		Parameters: map[string]any{},
	})
	if err == nil {
		t.Fatalf("expected duplicate managed openresty deployment to fail")
	}
}

func TestCleanupLegacyApprovalData(t *testing.T) {
	instance := newTestApp(t)

	if err := instance.db.Exec("CREATE TABLE approval_requests (id TEXT PRIMARY KEY)").Error; err != nil {
		t.Fatalf("create approval_requests table: %v", err)
	}
	if err := instance.db.Create(&model.AuditEvent{
		ID:         "approval-event",
		Action:     "approval_created",
		TargetType: "project",
		TargetID:   "demo",
	}).Error; err != nil {
		t.Fatalf("create approval audit event: %v", err)
	}
	if err := instance.db.Create(&model.AuditEvent{
		ID:         "normal-event",
		Action:     "login_success",
		TargetType: "user",
		TargetID:   "admin",
	}).Error; err != nil {
		t.Fatalf("create normal audit event: %v", err)
	}

	if err := cleanupLegacyApprovalData(instance.db); err != nil {
		t.Fatalf("cleanup legacy approval data: %v", err)
	}

	if instance.db.Migrator().HasTable("approval_requests") {
		t.Fatalf("expected approval_requests table to be dropped")
	}

	var count int64
	if err := instance.db.Model(&model.AuditEvent{}).Where("action = ?", "approval_created").Count(&count).Error; err != nil {
		t.Fatalf("count approval audit events: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 approval audit events, got %d", count)
	}
	if err := instance.db.Model(&model.AuditEvent{}).Where("action = ?", "login_success").Count(&count).Error; err != nil {
		t.Fatalf("count remaining audit events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 normal audit event, got %d", count)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()

	root := t.TempDir()
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
	writeTemplate(t, templatesDir, managedOpenRestyTemplateID, `id: openresty
name: OpenResty
version: "1"
description: managed openresty
params: []
`, `services:
  app:
    image: openresty/openresty:alpine
    container_name: {{ .Runtime.OpenRestyContainer }}
    network_mode: host
    volumes:
      - "{{ .Runtime.OpenRestyHostConfDir }}:/etc/nginx/conf.d"
      - "{{ .Runtime.OpenRestyHostConfDir }}:/etc/openresty/conf.d"
      - "{{ .Runtime.OpenRestyHostSiteDir }}:/var/www/openresty"
      - "{{ .Runtime.OpenRestyHostCertDir }}:/etc/camopanel/certs"
`)

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
		BridgeNetworkName:  "camopanel",
		HostControlHelper:  "/usr/local/bin/camopanel-hostctl",
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
