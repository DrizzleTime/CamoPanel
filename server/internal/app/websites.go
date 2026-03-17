package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createWebsiteRequest struct {
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

type updateWebsiteRequest = createWebsiteRequest

type createWebsitePayload struct {
	Name          string
	Type          string
	Domain        string
	Domains       []string
	RootPath      string
	IndexFiles    []string
	ProxyPass     string
	PHPProjectID  string
	PHPPort       int
	RewriteMode   string
	RewritePreset string
	RewriteRules  string
}

type websiteConfigPreviewResponse struct {
	Config string `json:"config"`
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

func (a *App) handleUpdateWebsite(c *gin.Context) {
	var req updateWebsiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求格式错误")
		return
	}

	website, err := a.findWebsite(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	updated, err := a.updateWebsite(c.Request.Context(), currentUser(c).ID, website, req)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"website": updated})
}

func (a *App) handleDeleteWebsite(c *gin.Context) {
	website, err := a.findWebsite(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	if err := a.deleteWebsite(c.Request.Context(), currentUser(c).ID, website); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (a *App) handleWebsiteConfigPreview(c *gin.Context) {
	website, err := a.findWebsite(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, err.Error())
		return
	}

	config, err := a.previewWebsiteConfig(c.Request.Context(), website)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	c.JSON(http.StatusOK, websiteConfigPreviewResponse{Config: config})
}

func (a *App) createWebsite(ctx context.Context, actorID string, req createWebsiteRequest) (model.Website, error) {
	payload, err := a.prepareWebsitePayload(ctx, req, "")
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

func (a *App) updateWebsite(ctx context.Context, actorID string, website model.Website, req updateWebsiteRequest) (model.Website, error) {
	payload, err := a.prepareWebsitePayload(ctx, req, website.ID)
	if err != nil {
		return model.Website{}, err
	}
	if payload.Name != website.Name {
		return model.Website{}, fmt.Errorf("当前版本不支持修改站点名")
	}

	spec := websiteSpecFromPayload(payload)
	materialized, err := a.openresty.UpdateWebsite(ctx, spec, website.ConfigPath)
	if err != nil {
		return model.Website{}, err
	}

	website.Type = payload.Type
	website.SiteMode = payload.Type
	website.Domain = payload.Domain
	website.DomainsJSON = mustJSON(payload.Domains)
	website.RootPath = materialized.RootPath
	website.IndexFiles = strings.Join(payload.IndexFiles, " ")
	website.ProxyPass = payload.ProxyPass
	website.PHPProjectID = payload.PHPProjectID
	website.PHPPort = payload.PHPPort
	website.RewriteMode = payload.RewriteMode
	website.RewritePreset = payload.RewritePreset
	website.RewriteRules = payload.RewriteRules
	website.ConfigPath = materialized.ConfigPath
	website.Status = "ready"
	website.UpdatedAt = time.Now().UTC()

	if err := a.saveWebsite(website); err != nil {
		return model.Website{}, err
	}

	_ = a.recordAudit(actorID, "website_update", "website", website.ID, map[string]any{
		"domain": website.Domain,
		"type":   website.Type,
	})
	return website, nil
}

func (a *App) deleteWebsite(ctx context.Context, actorID string, website model.Website) error {
	if err := a.openresty.DeleteWebsite(ctx, website.ConfigPath); err != nil {
		return err
	}
	if err := a.removeWebsite(website); err != nil {
		return err
	}
	_ = a.recordAudit(actorID, "website_delete", "website", website.ID, map[string]any{
		"name":   website.Name,
		"domain": website.Domain,
	})
	return nil
}

func (a *App) previewWebsiteConfig(ctx context.Context, website model.Website) (string, error) {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return "", err
	}
	return a.openresty.PreviewWebsiteConfig(websiteSpecFromModel(website))
}

func (a *App) prepareWebsitePayload(ctx context.Context, req createWebsiteRequest, currentWebsiteID string) (createWebsitePayload, error) {
	if err := a.openresty.EnsureReady(ctx); err != nil {
		return createWebsitePayload{}, err
	}

	normalizedName := normalizeProjectName(req.Name)
	if !projectNamePattern.MatchString(normalizedName) {
		return createWebsitePayload{}, fmt.Errorf("网站名只能包含小写字母、数字、下划线和中划线")
	}

	websiteType := strings.TrimSpace(req.Type)
	switch websiteType {
	case model.WebsiteTypeStatic, model.WebsiteTypePHP, model.WebsiteTypeProxy:
	default:
		return createWebsitePayload{}, fmt.Errorf("不支持的网站类型")
	}

	domain := normalizeDomain(req.Domain)
	if domain == "" {
		return createWebsitePayload{}, fmt.Errorf("主域名不能为空")
	}

	domains := normalizeDomains(req.Domains)
	if slices.Contains(domains, domain) {
		return createWebsitePayload{}, fmt.Errorf("附加域名不能和主域名重复")
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

	rootPath := ""
	if websiteType == model.WebsiteTypeStatic || websiteType == model.WebsiteTypePHP {
		rootPath = a.normalizeWebsiteRootPath(normalizedName, req.RootPath)
		if rootPath == "" {
			return createWebsitePayload{}, fmt.Errorf("站点目录不能为空")
		}
		if err := ensurePathWithin(filepath.Join(a.cfg.OpenRestyDataDir, "www"), rootPath); err != nil {
			return createWebsitePayload{}, err
		}
	}

	indexFiles := normalizeIndexFilesByType(websiteType, req.IndexFiles)
	rewriteMode, rewritePreset, rewriteRules, err := normalizeRewriteConfig(req.RewriteMode, req.RewritePreset, req.RewriteRules)
	if err != nil {
		return createWebsitePayload{}, err
	}
	phpProjectID := strings.TrimSpace(req.PHPProjectID)
	phpPort := 0
	if websiteType == model.WebsiteTypePHP {
		phpProject, err := a.findPHPEnvironmentProject(phpProjectID)
		if err != nil {
			return createWebsitePayload{}, err
		}
		phpPort, err = projectConfigPort(phpProject)
		if err != nil {
			return createWebsitePayload{}, err
		}
		proxyPass = ""
	}
	if websiteType == model.WebsiteTypeProxy {
		rewriteMode = "off"
		rewritePreset = ""
		rewriteRules = ""
		rootPath = ""
		phpProjectID = ""
		phpPort = 0
	}

	if err := a.ensureWebsiteNameAvailable(normalizedName, currentWebsiteID); err != nil {
		return createWebsitePayload{}, err
	}
	if err := a.ensureWebsiteDomainsAvailable(append([]string{domain}, domains...), currentWebsiteID); err != nil {
		return createWebsitePayload{}, err
	}

	return createWebsitePayload{
		Name:          normalizedName,
		Type:          websiteType,
		Domain:        domain,
		Domains:       domains,
		RootPath:      rootPath,
		IndexFiles:    indexFiles,
		ProxyPass:     proxyPass,
		PHPProjectID:  phpProjectID,
		PHPPort:       phpPort,
		RewriteMode:   rewriteMode,
		RewritePreset: rewritePreset,
		RewriteRules:  rewriteRules,
	}, nil
}

func (a *App) executeCreateWebsite(ctx context.Context, payload createWebsitePayload) (model.Website, error) {
	spec := websiteSpecFromPayload(payload)
	materialized, err := a.openresty.CreateWebsite(ctx, spec)
	if err != nil {
		return model.Website{}, err
	}

	website := model.Website{
		ID:            uuid.NewString(),
		Name:          payload.Name,
		Type:          payload.Type,
		Domain:        payload.Domain,
		DomainsJSON:   mustJSON(payload.Domains),
		SiteMode:      payload.Type,
		RootPath:      materialized.RootPath,
		IndexFiles:    strings.Join(payload.IndexFiles, " "),
		ProxyPass:     payload.ProxyPass,
		PHPProjectID:  payload.PHPProjectID,
		PHPPort:       payload.PHPPort,
		RewriteMode:   payload.RewriteMode,
		RewritePreset: payload.RewritePreset,
		RewriteRules:  payload.RewriteRules,
		ConfigPath:    materialized.ConfigPath,
		Status:        "ready",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := a.saveWebsite(website); err != nil {
		return model.Website{}, err
	}
	return website, nil
}

func (a *App) ensureWebsiteNameAvailable(name string, currentWebsiteID string) error {
	websites, err := a.listWebsites()
	if err != nil {
		return err
	}

	for _, website := range websites {
		if currentWebsiteID != "" && website.ID == currentWebsiteID {
			continue
		}
		if website.Name == name {
			return fmt.Errorf("网站名已存在")
		}
	}
	return nil
}

func (a *App) ensureWebsiteDomainsAvailable(domains []string, currentWebsiteID string) error {
	websites, err := a.listWebsites()
	if err != nil {
		return err
	}

	for _, domain := range domains {
		for _, website := range websites {
			if currentWebsiteID != "" && website.ID == currentWebsiteID {
				continue
			}
			allDomains := append([]string{normalizeDomain(website.Domain)}, normalizeDomains(parseJSONStringArray(website.DomainsJSON))...)
			if slices.Contains(allDomains, domain) {
				return fmt.Errorf("域名 %s 已存在", domain)
			}
		}
	}
	return nil
}

func (a *App) normalizeWebsiteRootPath(name string, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return filepath.Join(a.cfg.OpenRestyDataDir, "www", name)
	}

	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed)
	}
	return filepath.Join(a.cfg.OpenRestyDataDir, "www", filepath.Clean(trimmed))
}

func ensurePathWithin(root string, target string) error {
	relativePath, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	if err != nil || strings.HasPrefix(relativePath, "..") {
		return fmt.Errorf("站点目录必须位于 %s 下", root)
	}
	return nil
}

func normalizeDomains(items []string) []string {
	result := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		domain := normalizeDomain(item)
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		result = append(result, domain)
	}
	return result
}

func normalizeDomain(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeIndexFiles(raw string) []string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) == 0 {
		return []string{"index.html", "index.htm"}
	}
	return fields
}

func normalizeIndexFilesByType(websiteType string, raw string) []string {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) > 0 {
		return fields
	}
	if websiteType == model.WebsiteTypePHP {
		return []string{"index.php", "index.html", "index.htm"}
	}
	return []string{"index.html", "index.htm"}
}

func normalizeRewriteConfig(mode string, preset string, rules string) (string, string, string, error) {
	mode = strings.TrimSpace(mode)
	switch mode {
	case "", "off":
		return "off", "", "", nil
	case "preset":
		preset = strings.TrimSpace(preset)
		if !slices.Contains([]string{"spa", "front_controller"}, preset) {
			return "", "", "", fmt.Errorf("请选择合法的伪静态预设")
		}
		return "preset", preset, "", nil
	case "custom":
		rules = strings.TrimSpace(rules)
		if rules == "" {
			return "", "", "", fmt.Errorf("请输入自定义伪静态规则")
		}
		return "custom", "", rules, nil
	default:
		return "", "", "", fmt.Errorf("不支持的伪静态模式")
	}
}

func websiteSpecFromPayload(payload createWebsitePayload) services.WebsiteSpec {
	return services.WebsiteSpec{
		Name:          payload.Name,
		Type:          payload.Type,
		Domain:        payload.Domain,
		Domains:       payload.Domains,
		RootPath:      payload.RootPath,
		IndexFiles:    payload.IndexFiles,
		ProxyPass:     payload.ProxyPass,
		PHPPort:       payload.PHPPort,
		RewriteMode:   payload.RewriteMode,
		RewritePreset: payload.RewritePreset,
		RewriteRules:  payload.RewriteRules,
	}
}

func websiteSpecFromModel(website model.Website) services.WebsiteSpec {
	return services.WebsiteSpec{
		Name:          website.Name,
		Type:          firstNonBlank(website.SiteMode, website.Type),
		Domain:        website.Domain,
		Domains:       parseJSONStringArray(website.DomainsJSON),
		RootPath:      website.RootPath,
		IndexFiles:    normalizeIndexFilesByType(firstNonBlank(website.SiteMode, website.Type), website.IndexFiles),
		ProxyPass:     website.ProxyPass,
		PHPPort:       website.PHPPort,
		RewriteMode:   firstNonBlank(website.RewriteMode, "off"),
		RewritePreset: website.RewritePreset,
		RewriteRules:  website.RewriteRules,
	}
}

func parseJSONStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func mustJSON(value any) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (a *App) findPHPEnvironmentProject(projectID string) (model.Project, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return model.Project{}, fmt.Errorf("请选择 PHP 环境")
	}
	project, err := a.findProject(projectID)
	if err != nil {
		return model.Project{}, err
	}
	if project.TemplateID != "php-fpm" {
		return model.Project{}, fmt.Errorf("所选项目不是 PHP 环境")
	}
	return project, nil
}

func projectConfigPort(project model.Project) (int, error) {
	config := map[string]any{}
	_ = json.Unmarshal([]byte(project.ConfigJSON), &config)
	port, err := normalizeInt(config["port"])
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("PHP 环境缺少有效端口配置")
	}
	return port, nil
}
