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

type createWebsiteApprovalPayload struct {
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

	approval, err := a.createWebsiteApproval(c.Request.Context(), currentUser(c).ID, "ui", req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{"approval": approval})
}

func (a *App) createWebsiteApproval(ctx context.Context, actorID, source string, req createWebsiteRequest) (model.ApprovalRequest, error) {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return model.ApprovalRequest{}, err
	}

	normalizedName := normalizeProjectName(req.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return model.ApprovalRequest{}, fmt.Errorf("网站名只能包含小写字母、数字、下划线和中划线")
	}

	websiteType := strings.TrimSpace(req.Type)
	switch websiteType {
	case model.WebsiteTypeStatic, model.WebsiteTypeProxy:
	default:
		return model.ApprovalRequest{}, fmt.Errorf("不支持的网站类型")
	}

	domain := strings.TrimSpace(req.Domain)
	if domain == "" {
		return model.ApprovalRequest{}, fmt.Errorf("域名不能为空")
	}

	proxyPass := strings.TrimSpace(req.ProxyPass)
	if websiteType == model.WebsiteTypeProxy && proxyPass == "" {
		return model.ApprovalRequest{}, fmt.Errorf("代理地址不能为空")
	}
	if websiteType == model.WebsiteTypeProxy {
		target, err := url.Parse(proxyPass)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return model.ApprovalRequest{}, fmt.Errorf("代理地址格式不正确")
		}
	}

	var nameCount int64
	if err := a.db.Model(&model.Website{}).Where("name = ?", normalizedName).Count(&nameCount).Error; err != nil {
		return model.ApprovalRequest{}, err
	}
	if nameCount > 0 {
		return model.ApprovalRequest{}, fmt.Errorf("网站名已存在")
	}

	var domainCount int64
	if err := a.db.Model(&model.Website{}).Where("domain = ?", domain).Count(&domainCount).Error; err != nil {
		return model.ApprovalRequest{}, err
	}
	if domainCount > 0 {
		return model.ApprovalRequest{}, fmt.Errorf("域名已存在")
	}

	payload := createWebsiteApprovalPayload{
		Name:      normalizedName,
		Type:      websiteType,
		Domain:    domain,
		ProxyPass: proxyPass,
	}
	return a.saveApproval(
		actorID,
		source,
		model.ApprovalActionCreateWebsite,
		"website",
		normalizedName,
		payload,
		fmt.Sprintf("创建网站 %s（%s）", domain, websiteTypeLabel(websiteType)),
	)
}

func (a *App) executeCreateWebsite(ctx context.Context, payload createWebsiteApprovalPayload) error {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return err
	}

	var nameCount int64
	if err := a.db.Model(&model.Website{}).Where("name = ?", payload.Name).Count(&nameCount).Error; err != nil {
		return err
	}
	if nameCount > 0 {
		return fmt.Errorf("网站名已存在")
	}

	var domainCount int64
	if err := a.db.Model(&model.Website{}).Where("domain = ?", payload.Domain).Count(&domainCount).Error; err != nil {
		return err
	}
	if domainCount > 0 {
		return fmt.Errorf("域名已存在")
	}

	materialized, err := a.openresty.CreateWebsite(ctx, services.WebsiteSpec{
		Name:      payload.Name,
		Type:      payload.Type,
		Domain:    payload.Domain,
		ProxyPass: payload.ProxyPass,
	})
	if err != nil {
		return err
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
	return a.db.Create(&website).Error
}

func (a *App) listWebsites() ([]model.Website, error) {
	var websites []model.Website
	if err := a.db.Order("created_at desc").Find(&websites).Error; err != nil {
		return nil, err
	}
	return websites, nil
}
