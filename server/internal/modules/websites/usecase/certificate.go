package usecase

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"

	websitesdomain "camopanel/server/internal/modules/websites/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformopenresty "camopanel/server/internal/platform/openresty"

	"github.com/google/uuid"
)

type IssueCertificateInput struct {
	ActorID string
	Domain  string
	Email   string
}

type CertificateOutput struct {
	Certificate websitesdomain.Certificate
}

type IssueCertificate struct {
	websites     WebsiteRepository
	certificates CertificateRepository
	openresty    OpenRestyManager
	audit        AuditRecorder
}

func NewIssueCertificate(websites WebsiteRepository, certificates CertificateRepository, openresty OpenRestyManager, audit AuditRecorder) *IssueCertificate {
	return &IssueCertificate{websites: websites, certificates: certificates, openresty: openresty, audit: audit}
}

func (u *IssueCertificate) Execute(ctx context.Context, input IssueCertificateInput) (CertificateOutput, error) {
	domain := strings.TrimSpace(input.Domain)
	if domain == "" {
		return CertificateOutput{}, fmt.Errorf("域名不能为空")
	}
	email := strings.TrimSpace(input.Email)
	if email == "" {
		return CertificateOutput{}, fmt.Errorf("邮箱不能为空")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return CertificateOutput{}, fmt.Errorf("邮箱格式不正确")
	}

	var website *websitesdomain.Website
	websites, err := u.websites.List(ctx)
	if err != nil {
		return CertificateOutput{}, err
	}
	for _, item := range websites {
		if normalizeDomain(item.Domain) == normalizeDomain(domain) {
			copied := item
			website = &copied
			break
		}
	}
	if website != nil && len(website.Domains) > 0 {
		return CertificateOutput{}, fmt.Errorf("第一版暂不支持为多域名站点自动启用 HTTPS，请先移除附加域名")
	}

	certificate, err := u.certificates.FindByDomain(ctx, domain)
	if err != nil {
		certificate = websitesdomain.Certificate{
			ID:       uuid.NewString(),
			Domain:   domain,
			Provider: "letsencrypt",
		}
	}
	certificate.Email = email
	certificate.Status = "applying"
	certificate.LastError = ""
	if err := u.certificates.Save(ctx, certificate); err != nil {
		return CertificateOutput{}, err
	}

	if website != nil {
		if _, err := u.openresty.SyncWebsite(ctx, websiteSpecFromWebsite(*website)); err != nil {
			certificate.Status = "error"
			certificate.LastError = err.Error()
			_ = u.certificates.Save(ctx, certificate)
			return CertificateOutput{}, err
		}
	}

	materialized, err := u.openresty.IssueCertificate(ctx, platformopenresty.CertificateSpec{
		Domain:             domain,
		Email:              email,
		UseExistingWebsite: website != nil,
	})
	if err != nil {
		certificate.Status = "error"
		certificate.LastError = err.Error()
		_ = u.certificates.Save(ctx, certificate)
		return CertificateOutput{}, err
	}
	certificate.Provider = materialized.Provider
	certificate.Status = "issued"
	certificate.FullchainPath = materialized.FullchainPath
	certificate.PrivateKeyPath = materialized.PrivateKeyPath
	certificate.ExpiresAt = materialized.ExpiresAt
	certificate.LastError = ""
	certificate.UpdatedAt = time.Now().UTC()
	if err := u.certificates.Save(ctx, certificate); err != nil {
		return CertificateOutput{}, err
	}

	if website != nil {
		if _, err := u.openresty.SyncWebsite(ctx, websiteSpecFromWebsite(*website)); err != nil {
			certificate.LastError = "证书已签发，但网站启用 HTTPS 失败: " + err.Error()
			_ = u.certificates.Save(ctx, certificate)
			return CertificateOutput{}, err
		}
	}

	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "certificate_apply",
		TargetType: "certificate",
		TargetID:   certificate.ID,
		Metadata:   map[string]any{"domain": domain, "email": email},
	})
	return CertificateOutput{Certificate: certificate}, nil
}

type DeleteCertificateInput struct {
	ActorID       string
	CertificateID string
}

type DeleteCertificate struct {
	websites     WebsiteRepository
	certificates CertificateRepository
	openresty    OpenRestyManager
	audit        AuditRecorder
}

func NewDeleteCertificate(websites WebsiteRepository, certificates CertificateRepository, openresty OpenRestyManager, audit AuditRecorder) *DeleteCertificate {
	return &DeleteCertificate{websites: websites, certificates: certificates, openresty: openresty, audit: audit}
}

func (u *DeleteCertificate) Execute(ctx context.Context, input DeleteCertificateInput) error {
	certificate, err := u.certificates.FindByID(ctx, input.CertificateID)
	if err != nil {
		return err
	}
	if err := u.openresty.DeleteCertificate(ctx, certificate.Domain); err != nil {
		return err
	}

	websites, err := u.websites.List(ctx)
	if err != nil {
		return err
	}
	for _, item := range websites {
		if normalizeDomain(item.Domain) == normalizeDomain(certificate.Domain) {
			if _, err := u.openresty.SyncWebsite(ctx, websiteSpecFromWebsite(item)); err != nil {
				return err
			}
			break
		}
	}
	if err := u.certificates.Delete(ctx, certificate.ID); err != nil {
		return err
	}
	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "certificate_delete",
		TargetType: "certificate",
		TargetID:   certificate.ID,
		Metadata:   map[string]any{"domain": certificate.Domain},
	})
	return nil
}

type ListCertificates struct {
	websites     WebsiteRepository
	certificates CertificateRepository
}

func NewListCertificates(websites WebsiteRepository, certificates CertificateRepository) *ListCertificates {
	return &ListCertificates{websites: websites, certificates: certificates}
}

func (u *ListCertificates) Execute(ctx context.Context) ([]websitesdomain.Certificate, error) {
	return u.certificates.List(ctx)
}
