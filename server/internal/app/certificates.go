package app

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createCertificateRequest struct {
	Domain string `json:"domain"`
	Email  string `json:"email"`
}

type certificateItem struct {
	ID             string `json:"id"`
	Domain         string `json:"domain"`
	Email          string `json:"email"`
	Provider       string `json:"provider"`
	Status         string `json:"status"`
	FullchainPath  string `json:"fullchain_path"`
	PrivateKeyPath string `json:"private_key_path"`
	LastError      string `json:"last_error"`
	ExpiresAt      string `json:"expires_at"`
	WebsiteID      string `json:"website_id,omitempty"`
	WebsiteName    string `json:"website_name,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (a *App) handleCertificates(c *gin.Context) {
	items, err := a.listCertificateItems()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) handleCreateCertificate(c *gin.Context) {
	var req createCertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	item, err := a.createCertificate(c.Request.Context(), currentUser(c).ID, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"certificate": item})
}

func (a *App) handleDeleteCertificate(c *gin.Context) {
	certificate, err := a.findCertificate(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}
	if err := a.deleteCertificate(c.Request.Context(), currentUser(c).ID, certificate); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *App) createCertificate(ctx context.Context, actorID string, req createCertificateRequest) (certificateItem, error) {
	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return certificateItem{}, fmt.Errorf("域名不能为空")
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return certificateItem{}, fmt.Errorf("邮箱不能为空")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return certificateItem{}, fmt.Errorf("邮箱格式不正确")
	}

	website, _ := a.findWebsiteByDomain(domain)
	if website != nil && len(parseJSONStringArray(website.DomainsJSON)) > 0 {
		return certificateItem{}, fmt.Errorf("第一版暂不支持为多域名站点自动启用 HTTPS，请先移除附加域名")
	}

	certificate := model.Certificate{}
	err := a.db.Where("domain = ?", domain).First(&certificate).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return certificateItem{}, err
	}
	if err == gorm.ErrRecordNotFound {
		certificate = model.Certificate{
			ID:       uuid.NewString(),
			Domain:   domain,
			Provider: "letsencrypt",
		}
	}
	certificate.Email = email
	certificate.Status = "applying"
	certificate.LastError = ""
	if saveErr := a.db.Save(&certificate).Error; saveErr != nil {
		return certificateItem{}, saveErr
	}

	if website != nil {
		if _, err := a.openresty.SyncWebsite(ctx, websiteSpecFromModel(*website)); err != nil {
			certificate.Status = "error"
			certificate.LastError = err.Error()
			_ = a.db.Save(&certificate).Error
			return certificateItem{}, err
		}
	}

	materialized, err := a.openresty.IssueCertificate(ctx, services.CertificateSpec{
		Domain:             domain,
		Email:              email,
		UseExistingWebsite: website != nil,
	})
	if err != nil {
		certificate.Status = "error"
		certificate.LastError = err.Error()
		_ = a.db.Save(&certificate).Error
		return certificateItem{}, err
	}

	certificate.Provider = materialized.Provider
	certificate.Status = "issued"
	certificate.FullchainPath = materialized.FullchainPath
	certificate.PrivateKeyPath = materialized.PrivateKeyPath
	certificate.ExpiresAt = materialized.ExpiresAt
	certificate.LastError = ""
	if err := a.db.Save(&certificate).Error; err != nil {
		return certificateItem{}, err
	}

	if website != nil {
		if _, err := a.openresty.SyncWebsite(ctx, websiteSpecFromModel(*website)); err != nil {
			certificate.LastError = "证书已签发，但网站启用 HTTPS 失败: " + err.Error()
			_ = a.db.Save(&certificate).Error
			return certificateItem{}, err
		}
	}

	_ = a.recordAudit(actorID, "certificate_apply", "certificate", certificate.ID, map[string]any{
		"domain": domain,
		"email":  email,
	})

	return a.buildCertificateItem(certificate, website), nil
}

func (a *App) deleteCertificate(ctx context.Context, actorID string, certificate model.Certificate) error {
	if err := a.openresty.DeleteCertificate(ctx, certificate.Domain); err != nil {
		return err
	}

	website, _ := a.findWebsiteByDomain(certificate.Domain)
	if website != nil {
		if _, err := a.openresty.SyncWebsite(ctx, websiteSpecFromModel(*website)); err != nil {
			return err
		}
	}

	if err := a.db.Delete(&certificate).Error; err != nil {
		return err
	}

	_ = a.recordAudit(actorID, "certificate_delete", "certificate", certificate.ID, map[string]any{
		"domain": certificate.Domain,
	})
	return nil
}

func (a *App) listCertificateItems() ([]certificateItem, error) {
	var certificates []model.Certificate
	if err := a.db.Order("updated_at desc").Find(&certificates).Error; err != nil {
		return nil, err
	}

	websites, err := a.listWebsites()
	if err != nil {
		return nil, err
	}
	websitesByDomain := map[string]model.Website{}
	for _, website := range websites {
		websitesByDomain[website.Domain] = website
	}

	items := make([]certificateItem, 0, len(certificates))
	for _, certificate := range certificates {
		var website *model.Website
		if matched, ok := websitesByDomain[certificate.Domain]; ok {
			website = &matched
		}
		items = append(items, a.buildCertificateItem(certificate, website))
	}
	return items, nil
}

func (a *App) buildCertificateItem(certificate model.Certificate, website *model.Website) certificateItem {
	item := certificateItem{
		ID:             certificate.ID,
		Domain:         certificate.Domain,
		Email:          certificate.Email,
		Provider:       certificate.Provider,
		Status:         certificate.Status,
		FullchainPath:  certificate.FullchainPath,
		PrivateKeyPath: certificate.PrivateKeyPath,
		LastError:      certificate.LastError,
		ExpiresAt:      certificate.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:      certificate.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      certificate.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if website != nil {
		item.WebsiteID = website.ID
		item.WebsiteName = website.Name
	}
	return item
}

func (a *App) findCertificate(id string) (model.Certificate, error) {
	var certificate model.Certificate
	if err := a.db.First(&certificate, "id = ?", id).Error; err != nil {
		return model.Certificate{}, fmt.Errorf("证书不存在")
	}
	return certificate, nil
}

func (a *App) findCertificateByDomain(domain string) (model.Certificate, error) {
	var certificate model.Certificate
	if err := a.db.First(&certificate, "domain = ?", domain).Error; err != nil {
		return model.Certificate{}, fmt.Errorf("证书不存在")
	}
	return certificate, nil
}

func (a *App) findWebsiteByDomain(domain string) (*model.Website, error) {
	websites, err := a.listWebsites()
	if err != nil {
		return nil, err
	}
	for _, website := range websites {
		if normalizeDomain(website.Domain) == normalizeDomain(domain) {
			matched := website
			return &matched, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
