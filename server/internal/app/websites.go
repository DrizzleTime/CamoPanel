package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createWebsiteRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Domain    string `json:"domain"`
	ProxyPass string `json:"proxy_pass"`
}

type createWebsitePayload struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Domain    string `json:"domain"`
	ProxyPass string `json:"proxy_pass"`
}

func (a *App) handleOpenRestyStatus(c *gin.Context) {
	c.JSON(http.StatusOK, a.openresty.Status(c.Request.Context()))
}

func (a *App) handleWebsites(c *gin.Context) {
	websites, err := a.listWebsites()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": websites})
}

func (a *App) handleCreateWebsite(c *gin.Context) {
	var req createWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	website, err := a.createWebsite(c.Request.Context(), currentUser(c).ID, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"website": website})
}

func (a *App) createWebsite(ctx context.Context, actorID string, req createWebsiteRequest) (model.Website, error) {
	payload, err := a.prepareWebsitePayload(ctx, req)
	if err != nil {
		return model.Website{}, err
	}

	website, err := a.executeCreateWebsite(ctx, payload)
	if err != nil {
		return model.Website{}, err
	}

	_ = a.recordAudit(actorID, "website_create", "website", website.ID, map[string]any{
		"name":   website.Name,
		"domain": website.Domain,
		"type":   website.Type,
	})
	return website, nil
}

func (a *App) prepareWebsitePayload(ctx context.Context, req createWebsiteRequest) (createWebsitePayload, error) {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return createWebsitePayload{}, err
	}

	normalizedName := normalizeProjectName(req.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return createWebsitePayload{}, fmt.Errorf("网站名只能包含小写字母、数字、下划线和中划线")
	}

	websiteType := strings.TrimSpace(req.Type)
	switch websiteType {
	case model.WebsiteTypeStatic, model.WebsiteTypeProxy:
	default:
		return createWebsitePayload{}, fmt.Errorf("不支持的网站类型")
	}

	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return createWebsitePayload{}, fmt.Errorf("域名不能为空")
	}

	proxyPass := strings.TrimSpace(req.ProxyPass)
	if websiteType == model.WebsiteTypeProxy && proxyPass == "" {
		return createWebsitePayload{}, fmt.Errorf("代理地址不能为空")
	}
	if websiteType == model.WebsiteTypeProxy {
		target, err := url.Parse(proxyPass)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return createWebsitePayload{}, fmt.Errorf("代理地址格式不正确")
		}
	}

	var nameCount int64
	if err := a.db.Model(&model.Website{}).Where("name = ?", normalizedName).Count(&nameCount).Error; err != nil {
		return createWebsitePayload{}, err
	}
	if nameCount > 0 {
		return createWebsitePayload{}, fmt.Errorf("网站名已存在")
	}

	var domainCount int64
	if err := a.db.Model(&model.Website{}).Where("domain = ?", domain).Count(&domainCount).Error; err != nil {
		return createWebsitePayload{}, err
	}
	if domainCount > 0 {
		return createWebsitePayload{}, fmt.Errorf("域名已存在")
	}

	return createWebsitePayload{
		Name:      normalizedName,
		Type:      websiteType,
		Domain:    domain,
		ProxyPass: proxyPass,
	}, nil
}

func (a *App) executeCreateWebsite(ctx context.Context, payload createWebsitePayload) (model.Website, error) {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return model.Website{}, err
	}

	var nameCount int64
	if err := a.db.Model(&model.Website{}).Where("name = ?", payload.Name).Count(&nameCount).Error; err != nil {
		return model.Website{}, err
	}
	if nameCount > 0 {
		return model.Website{}, fmt.Errorf("网站名已存在")
	}

	var domainCount int64
	if err := a.db.Model(&model.Website{}).Where("domain = ?", payload.Domain).Count(&domainCount).Error; err != nil {
		return model.Website{}, err
	}
	if domainCount > 0 {
		return model.Website{}, fmt.Errorf("域名已存在")
	}

	materialized, err := a.openresty.CreateWebsite(ctx, services.WebsiteSpec{
		Name:      payload.Name,
		Type:      payload.Type,
		Domain:    payload.Domain,
		ProxyPass: payload.ProxyPass,
	})
	if err != nil {
		return model.Website{}, err
	}

	website := model.Website{
		ID:         uuid.NewString(),
		Name:       payload.Name,
		Type:       payload.Type,
		Domain:     payload.Domain,
		RootPath:   materialized.RootPath,
		ProxyPass:  payload.ProxyPass,
		ConfigPath: materialized.ConfigPath,
		Status:     "ready",
	}
	if err := a.db.Create(&website).Error; err != nil {
		return model.Website{}, err
	}
	return website, nil
}

func (a *App) listWebsites() ([]model.Website, error) {
	var websites []model.Website
	if err := a.db.Order("created_at desc").Find(&websites).Error; err != nil {
		return nil, err
	}
	return websites, nil
}
