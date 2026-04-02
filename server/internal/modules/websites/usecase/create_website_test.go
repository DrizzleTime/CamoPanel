package usecase_test

import (
	"context"
	"testing"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
	"camopanel/server/internal/modules/websites/usecase"
	platformaudit "camopanel/server/internal/platform/audit"
	platformopenresty "camopanel/server/internal/platform/openresty"
)

type websiteRepoStub struct {
	websites []websitesdomain.Website
	saved    websitesdomain.Website
}

func (s *websiteRepoStub) List(_ context.Context) ([]websitesdomain.Website, error) {
	return append([]websitesdomain.Website(nil), s.websites...), nil
}

func (s *websiteRepoStub) FindByID(_ context.Context, websiteID string) (websitesdomain.Website, error) {
	for _, item := range s.websites {
		if item.ID == websiteID {
			return item, nil
		}
	}
	return websitesdomain.Website{}, websitesdomain.ErrWebsiteNotFound
}

func (s *websiteRepoStub) Save(_ context.Context, website websitesdomain.Website) error {
	s.saved = website
	found := false
	for i, item := range s.websites {
		if item.ID == website.ID {
			s.websites[i] = website
			found = true
			break
		}
	}
	if !found {
		s.websites = append(s.websites, website)
	}
	return nil
}

func (s *websiteRepoStub) Delete(_ context.Context, websiteID string) error {
	filtered := s.websites[:0]
	for _, item := range s.websites {
		if item.ID != websiteID {
			filtered = append(filtered, item)
		}
	}
	s.websites = filtered
	return nil
}

type websiteCertificateRepoStub struct{}

func (s *websiteCertificateRepoStub) List(_ context.Context) ([]websitesdomain.Certificate, error) {
	return nil, nil
}

func (s *websiteCertificateRepoStub) FindByID(_ context.Context, _ string) (websitesdomain.Certificate, error) {
	return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
}

func (s *websiteCertificateRepoStub) FindByDomain(_ context.Context, _ string) (websitesdomain.Certificate, error) {
	return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
}

func (s *websiteCertificateRepoStub) Save(_ context.Context, _ websitesdomain.Certificate) error {
	return nil
}

func (s *websiteCertificateRepoStub) Delete(_ context.Context, _ string) error {
	return nil
}

type projectReaderStub struct {
	project projectsdomain.Project
}

func (s *projectReaderStub) FindByID(_ context.Context, projectID string) (projectsdomain.Project, error) {
	if s.project.ID != projectID {
		return projectsdomain.Project{}, projectsdomain.ErrProjectNotFound
	}
	return s.project, nil
}

type openrestyStub struct {
	createCalls int
	updateCalls int
	deleteCalls int
	lastSpec    platformopenresty.WebsiteSpec
}

func (s *openrestyStub) Status(context.Context) platformopenresty.Status {
	return platformopenresty.Status{Ready: true}
}
func (s *openrestyStub) EnsureReady(context.Context) error { return nil }
func (s *openrestyStub) PreviewWebsiteConfig(spec platformopenresty.WebsiteSpec) (string, error) {
	return "server_name " + spec.Domain + ";", nil
}
func (s *openrestyStub) CreateWebsite(_ context.Context, spec platformopenresty.WebsiteSpec) (platformopenresty.WebsiteMaterialized, error) {
	s.createCalls++
	s.lastSpec = spec
	return platformopenresty.WebsiteMaterialized{RootPath: "/tmp/" + spec.Name, ConfigPath: "/tmp/" + spec.Name + ".conf"}, nil
}
func (s *openrestyStub) UpdateWebsite(_ context.Context, spec platformopenresty.WebsiteSpec, configPath string) (platformopenresty.WebsiteMaterialized, error) {
	s.updateCalls++
	s.lastSpec = spec
	return platformopenresty.WebsiteMaterialized{RootPath: spec.RootPath, ConfigPath: configPath}, nil
}
func (s *openrestyStub) SyncWebsite(_ context.Context, spec platformopenresty.WebsiteSpec) (platformopenresty.WebsiteMaterialized, error) {
	s.lastSpec = spec
	return platformopenresty.WebsiteMaterialized{RootPath: spec.RootPath, ConfigPath: "/tmp/" + spec.Name + ".conf"}, nil
}
func (s *openrestyStub) DeleteWebsite(_ context.Context, configPath string) error {
	s.deleteCalls++
	_ = configPath
	return nil
}
func (s *openrestyStub) IssueCertificate(context.Context, platformopenresty.CertificateSpec) (platformopenresty.CertificateMaterialized, error) {
	return platformopenresty.CertificateMaterialized{}, nil
}
func (s *openrestyStub) DeleteCertificate(context.Context, string) error { return nil }

type websiteAuditStub struct {
	entries []platformaudit.Entry
}

func (s *websiteAuditStub) Record(_ context.Context, entry platformaudit.Entry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func TestCreateWebsitePersistsMetadataAndCallsOpenResty(t *testing.T) {
	repo := &websiteRepoStub{}
	openresty := &openrestyStub{}
	audit := &websiteAuditStub{}
	uc := usecase.NewCreateWebsite(repo, &websiteCertificateRepoStub{}, &projectReaderStub{}, openresty, audit, usecase.WebsiteConfig{
		OpenRestyDataDir: "/var/lib/camopanel/openresty",
	})

	got, err := uc.Execute(context.Background(), usecase.CreateWebsiteInput{
		ActorID:   "user-1",
		Name:      "demo-site",
		Type:      websitesdomain.TypeProxy,
		Domain:    "demo.local",
		ProxyPass: "http://127.0.0.1:3000",
	})
	if err != nil {
		t.Fatalf("create website: %v", err)
	}
	if openresty.createCalls != 1 {
		t.Fatalf("expected create call once, got %d", openresty.createCalls)
	}
	if got.Website.Domain != "demo.local" {
		t.Fatalf("expected domain demo.local, got %s", got.Website.Domain)
	}
	if len(repo.websites) != 1 {
		t.Fatalf("expected 1 website saved, got %d", len(repo.websites))
	}
}

func TestUpdateWebsiteCanSwitchToProxyMode(t *testing.T) {
	repo := &websiteRepoStub{
		websites: []websitesdomain.Website{{
			ID:     "website-1",
			Name:   "demo-site",
			Type:   websitesdomain.TypeStatic,
			Domain: "demo.local",
			Status: "ready",
		}},
	}
	openresty := &openrestyStub{}
	uc := usecase.NewUpdateWebsite(repo, &websiteCertificateRepoStub{}, &projectReaderStub{}, openresty, &websiteAuditStub{}, usecase.WebsiteConfig{
		OpenRestyDataDir: "/var/lib/camopanel/openresty",
	})

	got, err := uc.Execute(context.Background(), usecase.UpdateWebsiteInput{
		ActorID:   "user-1",
		WebsiteID: "website-1",
		Name:      "demo-site",
		Type:      websitesdomain.TypeProxy,
		Domain:    "updated.local",
		Domains:   []string{"www.updated.local"},
		ProxyPass: "http://127.0.0.1:3000",
	})
	if err != nil {
		t.Fatalf("update website: %v", err)
	}
	if openresty.updateCalls != 1 {
		t.Fatalf("expected update call once, got %d", openresty.updateCalls)
	}
	if got.Website.Type != websitesdomain.TypeProxy {
		t.Fatalf("expected type proxy, got %s", got.Website.Type)
	}
}
