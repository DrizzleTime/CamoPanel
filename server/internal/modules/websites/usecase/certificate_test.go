package usecase_test

import (
	"context"
	"testing"
	"time"

	websitesdomain "camopanel/server/internal/modules/websites/domain"
	"camopanel/server/internal/modules/websites/usecase"
	platformopenresty "camopanel/server/internal/platform/openresty"
)

type certificateRepoStub struct {
	items []websitesdomain.Certificate
	saved websitesdomain.Certificate
}

func (s *certificateRepoStub) List(_ context.Context) ([]websitesdomain.Certificate, error) {
	return append([]websitesdomain.Certificate(nil), s.items...), nil
}

func (s *certificateRepoStub) FindByID(_ context.Context, certificateID string) (websitesdomain.Certificate, error) {
	for _, item := range s.items {
		if item.ID == certificateID {
			return item, nil
		}
	}
	return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
}

func (s *certificateRepoStub) FindByDomain(_ context.Context, domain string) (websitesdomain.Certificate, error) {
	for _, item := range s.items {
		if item.Domain == domain {
			return item, nil
		}
	}
	return websitesdomain.Certificate{}, websitesdomain.ErrCertificateNotFound
}

func (s *certificateRepoStub) Save(_ context.Context, item websitesdomain.Certificate) error {
	s.saved = item
	found := false
	for i, current := range s.items {
		if current.ID == item.ID {
			s.items[i] = item
			found = true
			break
		}
	}
	if !found {
		s.items = append(s.items, item)
	}
	return nil
}

func (s *certificateRepoStub) Delete(_ context.Context, certificateID string) error {
	filtered := s.items[:0]
	for _, item := range s.items {
		if item.ID != certificateID {
			filtered = append(filtered, item)
		}
	}
	s.items = filtered
	return nil
}

type certificateOpenRestyStub struct {
	issueCalls  int
	deleteCalls int
	syncCalls   int
}

func (s *certificateOpenRestyStub) Status(context.Context) platformopenresty.Status {
	return platformopenresty.Status{Ready: true}
}
func (s *certificateOpenRestyStub) EnsureReady(context.Context) error { return nil }
func (s *certificateOpenRestyStub) PreviewWebsiteConfig(platformopenresty.WebsiteSpec) (string, error) {
	return "", nil
}
func (s *certificateOpenRestyStub) CreateWebsite(context.Context, platformopenresty.WebsiteSpec) (platformopenresty.WebsiteMaterialized, error) {
	return platformopenresty.WebsiteMaterialized{}, nil
}
func (s *certificateOpenRestyStub) UpdateWebsite(context.Context, platformopenresty.WebsiteSpec, string) (platformopenresty.WebsiteMaterialized, error) {
	return platformopenresty.WebsiteMaterialized{}, nil
}
func (s *certificateOpenRestyStub) SyncWebsite(_ context.Context, _ platformopenresty.WebsiteSpec) (platformopenresty.WebsiteMaterialized, error) {
	s.syncCalls++
	return platformopenresty.WebsiteMaterialized{}, nil
}
func (s *certificateOpenRestyStub) DeleteWebsite(context.Context, string) error { return nil }
func (s *certificateOpenRestyStub) IssueCertificate(_ context.Context, _ platformopenresty.CertificateSpec) (platformopenresty.CertificateMaterialized, error) {
	s.issueCalls++
	return platformopenresty.CertificateMaterialized{
		Provider:       "letsencrypt",
		FullchainPath:  "/tmp/fullchain.pem",
		PrivateKeyPath: "/tmp/privkey.pem",
		ExpiresAt:      time.Now().UTC().Add(90 * 24 * time.Hour),
	}, nil
}
func (s *certificateOpenRestyStub) DeleteCertificate(_ context.Context, _ string) error {
	s.deleteCalls++
	return nil
}

func TestIssueCertificatePersistsCertificateAndSyncsWebsite(t *testing.T) {
	websiteRepo := &websiteRepoStub{
		websites: []websitesdomain.Website{{
			ID:         "website-1",
			Name:       "demo-site",
			Type:       websitesdomain.TypeStatic,
			Domain:     "demo.local",
			IndexFiles: []string{"index.html", "index.htm"},
			Status:     "ready",
		}},
	}
	certificateRepo := &certificateRepoStub{}
	openresty := &certificateOpenRestyStub{}
	uc := usecase.NewIssueCertificate(websiteRepo, certificateRepo, openresty, &websiteAuditStub{})

	got, err := uc.Execute(context.Background(), usecase.IssueCertificateInput{
		ActorID: "user-1",
		Domain:  "demo.local",
		Email:   "admin@example.com",
	})
	if err != nil {
		t.Fatalf("issue certificate: %v", err)
	}
	if openresty.issueCalls != 1 {
		t.Fatalf("expected issue certificate once, got %d", openresty.issueCalls)
	}
	if openresty.syncCalls != 2 {
		t.Fatalf("expected sync website twice, got %d", openresty.syncCalls)
	}
	if got.Certificate.Domain != "demo.local" {
		t.Fatalf("expected domain demo.local, got %s", got.Certificate.Domain)
	}
}

func TestDeleteCertificateRemovesRecordAndCallsOpenResty(t *testing.T) {
	websiteRepo := &websiteRepoStub{
		websites: []websitesdomain.Website{{
			ID:     "website-1",
			Name:   "demo-site",
			Type:   websitesdomain.TypeStatic,
			Domain: "demo.local",
			Status: "ready",
		}},
	}
	certificateRepo := &certificateRepoStub{
		items: []websitesdomain.Certificate{{
			ID:     "certificate-1",
			Domain: "demo.local",
			Email:  "admin@example.com",
			Status: "issued",
		}},
	}
	openresty := &certificateOpenRestyStub{}
	uc := usecase.NewDeleteCertificate(websiteRepo, certificateRepo, openresty, &websiteAuditStub{})

	if err := uc.Execute(context.Background(), usecase.DeleteCertificateInput{
		ActorID:       "user-1",
		CertificateID: "certificate-1",
	}); err != nil {
		t.Fatalf("delete certificate: %v", err)
	}
	if openresty.deleteCalls != 1 {
		t.Fatalf("expected delete certificate once, got %d", openresty.deleteCalls)
	}
	if len(certificateRepo.items) != 0 {
		t.Fatalf("expected certificates to be removed, got %d", len(certificateRepo.items))
	}
}
