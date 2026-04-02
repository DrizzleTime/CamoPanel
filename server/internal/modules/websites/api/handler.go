package api

import (
	"context"
	"net/http"

	websitesdomain "camopanel/server/internal/modules/websites/domain"
	"camopanel/server/internal/modules/websites/usecase"
	"camopanel/server/internal/platform/errs"
	"camopanel/server/internal/platform/httpx"
	platformopenresty "camopanel/server/internal/platform/openresty"

	"github.com/gin-gonic/gin"
)

type OpenRestyManager interface {
	Status(ctx context.Context) platformopenresty.Status
}

type Handler struct {
	openresty         OpenRestyManager
	listWebsites      *usecase.ListWebsites
	createWebsite     *usecase.CreateWebsite
	updateWebsite     *usecase.UpdateWebsite
	deleteWebsite     *usecase.DeleteWebsite
	previewConfig     *usecase.PreviewConfig
	listCertificates  *usecase.ListCertificates
	issueCertificate  *usecase.IssueCertificate
	deleteCertificate *usecase.DeleteCertificate
}

func NewHandler(openresty OpenRestyManager, listWebsites *usecase.ListWebsites, createWebsite *usecase.CreateWebsite, updateWebsite *usecase.UpdateWebsite, deleteWebsite *usecase.DeleteWebsite, previewConfig *usecase.PreviewConfig, listCertificates *usecase.ListCertificates, issueCertificate *usecase.IssueCertificate, deleteCertificate *usecase.DeleteCertificate) *Handler {
	return &Handler{openresty: openresty, listWebsites: listWebsites, createWebsite: createWebsite, updateWebsite: updateWebsite, deleteWebsite: deleteWebsite, previewConfig: previewConfig, listCertificates: listCertificates, issueCertificate: issueCertificate, deleteCertificate: deleteCertificate}
}

func (h *Handler) OpenRestyStatus(c *gin.Context) {
	httpx.OK(c, h.openresty.Status(c.Request.Context()))
}

func (h *Handler) ListWebsites(c *gin.Context) {
	items, err := h.listWebsites.Execute(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "list websites", err))
		return
	}
	httpx.OK(c, gin.H{"items": serializeWebsites(items)})
}

func (h *Handler) CreateWebsite(c *gin.Context) {
	var req struct {
		Name          string   `json:"name"`
		Type          string   `json:"type"`
		Domain        string   `json:"domain"`
		Domains       []string `json:"domains"`
		RootPath      string   `json:"root_path"`
		IndexFiles    string   `json:"index_files"`
		ProxyPass     string   `json:"proxy_pass"`
		PHPProjectID  string   `json:"php_project_id"`
		RewriteMode   string   `json:"rewrite_mode"`
		RewritePreset string   `json:"rewrite_preset"`
		RewriteRules  string   `json:"rewrite_rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.createWebsite.Execute(c.Request.Context(), usecase.CreateWebsiteInput{
		Name: req.Name, Type: req.Type, Domain: req.Domain, Domains: req.Domains, RootPath: req.RootPath, IndexFiles: req.IndexFiles, ProxyPass: req.ProxyPass, PHPProjectID: req.PHPProjectID, RewriteMode: req.RewriteMode, RewritePreset: req.RewritePreset, RewriteRules: req.RewriteRules,
	})
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"website": serializeWebsite(item.Website)})
}

func (h *Handler) UpdateWebsite(c *gin.Context) {
	var req struct {
		Name          string   `json:"name"`
		Type          string   `json:"type"`
		Domain        string   `json:"domain"`
		Domains       []string `json:"domains"`
		RootPath      string   `json:"root_path"`
		IndexFiles    string   `json:"index_files"`
		ProxyPass     string   `json:"proxy_pass"`
		PHPProjectID  string   `json:"php_project_id"`
		RewriteMode   string   `json:"rewrite_mode"`
		RewritePreset string   `json:"rewrite_preset"`
		RewriteRules  string   `json:"rewrite_rules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.updateWebsite.Execute(c.Request.Context(), usecase.UpdateWebsiteInput{
		WebsiteID: c.Param("id"), Name: req.Name, Type: req.Type, Domain: req.Domain, Domains: req.Domains, RootPath: req.RootPath, IndexFiles: req.IndexFiles, ProxyPass: req.ProxyPass, PHPProjectID: req.PHPProjectID, RewriteMode: req.RewriteMode, RewritePreset: req.RewritePreset, RewriteRules: req.RewriteRules,
	})
	if err != nil {
		status := errs.CodeInvalidArgument
		if err == websitesdomain.ErrWebsiteNotFound {
			status = errs.CodeNotFound
		}
		httpx.ErrorFrom(c, errs.E(status, err.Error()))
		return
	}
	httpx.OK(c, gin.H{"website": serializeWebsite(item.Website)})
}

func (h *Handler) DeleteWebsite(c *gin.Context) {
	if err := h.deleteWebsite.Execute(c.Request.Context(), "", c.Param("id")); err != nil {
		status := errs.CodeInvalidArgument
		if err == websitesdomain.ErrWebsiteNotFound {
			status = errs.CodeNotFound
		}
		httpx.ErrorFrom(c, errs.E(status, err.Error()))
		return
	}
	httpx.OK(c, gin.H{"deleted": true})
}

func (h *Handler) PreviewConfig(c *gin.Context) {
	config, err := h.previewConfig.Execute(c.Request.Context(), c.Param("id"))
	if err != nil {
		status := errs.CodeInvalidArgument
		if err == websitesdomain.ErrWebsiteNotFound {
			status = errs.CodeNotFound
		}
		httpx.ErrorFrom(c, errs.E(status, err.Error()))
		return
	}
	httpx.OK(c, gin.H{"config": config})
}

func (h *Handler) ListCertificates(c *gin.Context) {
	items, err := h.listCertificates.Execute(c.Request.Context())
	if err != nil {
		httpx.ErrorFrom(c, errs.Wrap(errs.CodeInternal, "list certificates", err))
		return
	}
	httpx.OK(c, gin.H{"items": serializeCertificates(items)})
}

func (h *Handler) CreateCertificate(c *gin.Context) {
	var req struct {
		Domain string `json:"domain"`
		Email  string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, "请求格式错误"))
		return
	}
	item, err := h.issueCertificate.Execute(c.Request.Context(), usecase.IssueCertificateInput{Domain: req.Domain, Email: req.Email})
	if err != nil {
		httpx.ErrorFrom(c, errs.E(errs.CodeInvalidArgument, err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"certificate": serializeCertificate(item.Certificate)})
}

func (h *Handler) DeleteCertificate(c *gin.Context) {
	if err := h.deleteCertificate.Execute(c.Request.Context(), usecase.DeleteCertificateInput{CertificateID: c.Param("id")}); err != nil {
		status := errs.CodeInvalidArgument
		if err == websitesdomain.ErrCertificateNotFound {
			status = errs.CodeNotFound
		}
		httpx.ErrorFrom(c, errs.E(status, err.Error()))
		return
	}
	c.Status(http.StatusNoContent)
}
