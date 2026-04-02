package usecase

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	projectsdomain "camopanel/server/internal/modules/projects/domain"
	websitesdomain "camopanel/server/internal/modules/websites/domain"
	platformaudit "camopanel/server/internal/platform/audit"
	platformopenresty "camopanel/server/internal/platform/openresty"

	"github.com/google/uuid"
)

type WebsiteRepository interface {
	List(ctx context.Context) ([]websitesdomain.Website, error)
	FindByID(ctx context.Context, websiteID string) (websitesdomain.Website, error)
	Save(ctx context.Context, website websitesdomain.Website) error
	Delete(ctx context.Context, websiteID string) error
}

type CertificateRepository interface {
	List(ctx context.Context) ([]websitesdomain.Certificate, error)
	FindByID(ctx context.Context, certificateID string) (websitesdomain.Certificate, error)
	FindByDomain(ctx context.Context, domain string) (websitesdomain.Certificate, error)
	Save(ctx context.Context, item websitesdomain.Certificate) error
	Delete(ctx context.Context, certificateID string) error
}

type ProjectReader interface {
	FindByID(ctx context.Context, projectID string) (projectsdomain.Project, error)
}

type OpenRestyManager interface {
	platformopenresty.Manager
}

type AuditRecorder interface {
	Record(ctx context.Context, entry platformaudit.Entry) error
}

type WebsiteConfig struct {
	OpenRestyDataDir string
}

type CreateWebsiteInput struct {
	ActorID       string
	Name          string
	Type          string
	Domain        string
	Domains       []string
	RootPath      string
	IndexFiles    string
	ProxyPass     string
	PHPProjectID  string
	RewriteMode   string
	RewritePreset string
	RewriteRules  string
}

type WebsiteOutput struct {
	Website websitesdomain.Website
}

type CreateWebsite struct {
	websites     WebsiteRepository
	certificates CertificateRepository
	projects     ProjectReader
	openresty    OpenRestyManager
	audit        AuditRecorder
	cfg          WebsiteConfig
}

func NewCreateWebsite(websites WebsiteRepository, certificates CertificateRepository, projects ProjectReader, openresty OpenRestyManager, audit AuditRecorder, cfg WebsiteConfig) *CreateWebsite {
	return &CreateWebsite{websites: websites, certificates: certificates, projects: projects, openresty: openresty, audit: audit, cfg: cfg}
}

func (u *CreateWebsite) Execute(ctx context.Context, input CreateWebsiteInput) (WebsiteOutput, error) {
	payload, err := prepareWebsitePayload(ctx, input, "", u.websites, u.projects, u.openresty, u.cfg)
	if err != nil {
		return WebsiteOutput{}, err
	}
	materialized, err := u.openresty.CreateWebsite(ctx, websiteSpecFromPayload(payload))
	if err != nil {
		return WebsiteOutput{}, err
	}

	now := time.Now().UTC()
	website := websitesdomain.Website{
		ID:            uuid.NewString(),
		Name:          payload.Name,
		Type:          payload.Type,
		Domain:        payload.Domain,
		Domains:       payload.Domains,
		RootPath:      materialized.RootPath,
		IndexFiles:    payload.IndexFiles,
		ProxyPass:     payload.ProxyPass,
		PHPProjectID:  payload.PHPProjectID,
		PHPPort:       payload.PHPPort,
		RewriteMode:   payload.RewriteMode,
		RewritePreset: payload.RewritePreset,
		RewriteRules:  payload.RewriteRules,
		ConfigPath:    materialized.ConfigPath,
		Status:        "ready",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := u.websites.Save(ctx, website); err != nil {
		return WebsiteOutput{}, err
	}
	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "website_create",
		TargetType: "website",
		TargetID:   website.ID,
		Metadata:   map[string]any{"name": website.Name, "domain": website.Domain, "type": website.Type},
	})
	return WebsiteOutput{Website: website}, nil
}

type UpdateWebsiteInput struct {
	ActorID       string
	WebsiteID     string
	Name          string
	Type          string
	Domain        string
	Domains       []string
	RootPath      string
	IndexFiles    string
	ProxyPass     string
	PHPProjectID  string
	RewriteMode   string
	RewritePreset string
	RewriteRules  string
}

type UpdateWebsite struct {
	websites     WebsiteRepository
	certificates CertificateRepository
	projects     ProjectReader
	openresty    OpenRestyManager
	audit        AuditRecorder
	cfg          WebsiteConfig
}

func NewUpdateWebsite(websites WebsiteRepository, certificates CertificateRepository, projects ProjectReader, openresty OpenRestyManager, audit AuditRecorder, cfg WebsiteConfig) *UpdateWebsite {
	return &UpdateWebsite{websites: websites, certificates: certificates, projects: projects, openresty: openresty, audit: audit, cfg: cfg}
}

func (u *UpdateWebsite) Execute(ctx context.Context, input UpdateWebsiteInput) (WebsiteOutput, error) {
	current, err := u.websites.FindByID(ctx, input.WebsiteID)
	if err != nil {
		return WebsiteOutput{}, err
	}
	payload, err := prepareWebsitePayload(ctx, CreateWebsiteInput{
		Name:          input.Name,
		Type:          input.Type,
		Domain:        input.Domain,
		Domains:       input.Domains,
		RootPath:      input.RootPath,
		IndexFiles:    input.IndexFiles,
		ProxyPass:     input.ProxyPass,
		PHPProjectID:  input.PHPProjectID,
		RewriteMode:   input.RewriteMode,
		RewritePreset: input.RewritePreset,
		RewriteRules:  input.RewriteRules,
	}, current.ID, u.websites, u.projects, u.openresty, u.cfg)
	if err != nil {
		return WebsiteOutput{}, err
	}
	if payload.Name != current.Name {
		return WebsiteOutput{}, fmt.Errorf("当前版本不支持修改站点名")
	}

	materialized, err := u.openresty.UpdateWebsite(ctx, websiteSpecFromPayload(payload), current.ConfigPath)
	if err != nil {
		return WebsiteOutput{}, err
	}
	current.Type = payload.Type
	current.Domain = payload.Domain
	current.Domains = payload.Domains
	current.RootPath = materialized.RootPath
	current.IndexFiles = payload.IndexFiles
	current.ProxyPass = payload.ProxyPass
	current.PHPProjectID = payload.PHPProjectID
	current.PHPPort = payload.PHPPort
	current.RewriteMode = payload.RewriteMode
	current.RewritePreset = payload.RewritePreset
	current.RewriteRules = payload.RewriteRules
	current.ConfigPath = materialized.ConfigPath
	current.Status = "ready"
	current.UpdatedAt = time.Now().UTC()
	if err := u.websites.Save(ctx, current); err != nil {
		return WebsiteOutput{}, err
	}
	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    input.ActorID,
		Action:     "website_update",
		TargetType: "website",
		TargetID:   current.ID,
		Metadata:   map[string]any{"domain": current.Domain, "type": current.Type},
	})
	return WebsiteOutput{Website: current}, nil
}

type DeleteWebsite struct {
	websites  WebsiteRepository
	openresty OpenRestyManager
	audit     AuditRecorder
}

func NewDeleteWebsite(websites WebsiteRepository, openresty OpenRestyManager, audit AuditRecorder) *DeleteWebsite {
	return &DeleteWebsite{websites: websites, openresty: openresty, audit: audit}
}

func (u *DeleteWebsite) Execute(ctx context.Context, actorID, websiteID string) error {
	website, err := u.websites.FindByID(ctx, websiteID)
	if err != nil {
		return err
	}
	if err := u.openresty.DeleteWebsite(ctx, website.ConfigPath); err != nil {
		return err
	}
	if err := u.websites.Delete(ctx, websiteID); err != nil {
		return err
	}
	_ = u.audit.Record(ctx, platformaudit.Entry{
		ActorID:    actorID,
		Action:     "website_delete",
		TargetType: "website",
		TargetID:   website.ID,
		Metadata:   map[string]any{"name": website.Name, "domain": website.Domain},
	})
	return nil
}

type ListWebsites struct {
	websites WebsiteRepository
}

func NewListWebsites(websites WebsiteRepository) *ListWebsites {
	return &ListWebsites{websites: websites}
}

func (u *ListWebsites) Execute(ctx context.Context) ([]websitesdomain.Website, error) {
	return u.websites.List(ctx)
}

type PreviewConfig struct {
	websites  WebsiteRepository
	openresty OpenRestyManager
}

func NewPreviewConfig(websites WebsiteRepository, openresty OpenRestyManager) *PreviewConfig {
	return &PreviewConfig{websites: websites, openresty: openresty}
}

func (u *PreviewConfig) Execute(ctx context.Context, websiteID string) (string, error) {
	website, err := u.websites.FindByID(ctx, websiteID)
	if err != nil {
		return "", err
	}
	if err := u.openresty.EnsureReady(ctx); err != nil {
		return "", err
	}
	return u.openresty.PreviewWebsiteConfig(websiteSpecFromWebsite(website))
}

type websitePayload struct {
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

func prepareWebsitePayload(ctx context.Context, input CreateWebsiteInput, currentWebsiteID string, websites WebsiteRepository, projects ProjectReader, openresty OpenRestyManager, cfg WebsiteConfig) (websitePayload, error) {
	if err := openresty.EnsureReady(ctx); err != nil {
		return websitePayload{}, err
	}

	name := normalizeName(input.Name)
	if name == "" || !validWebsiteName(name) {
		return websitePayload{}, fmt.Errorf("网站名只能包含小写字母、数字、下划线和中划线")
	}
	websiteType := strings.TrimSpace(input.Type)
	switch websiteType {
	case websitesdomain.TypeStatic, websitesdomain.TypePHP, websitesdomain.TypeProxy:
	default:
		return websitePayload{}, fmt.Errorf("不支持的网站类型")
	}
	domain := normalizeDomain(input.Domain)
	if domain == "" {
		return websitePayload{}, fmt.Errorf("主域名不能为空")
	}
	domains := normalizeDomains(input.Domains)
	if slices.Contains(domains, domain) {
		return websitePayload{}, fmt.Errorf("附加域名不能和主域名重复")
	}

	proxyPass := strings.TrimSpace(input.ProxyPass)
	if websiteType == websitesdomain.TypeProxy && proxyPass == "" {
		return websitePayload{}, fmt.Errorf("代理地址不能为空")
	}
	if websiteType == websitesdomain.TypeProxy {
		target, err := url.Parse(proxyPass)
		if err != nil || target.Scheme == "" || target.Host == "" {
			return websitePayload{}, fmt.Errorf("代理地址格式不正确")
		}
	}

	rootPath := ""
	if websiteType == websitesdomain.TypeStatic || websiteType == websitesdomain.TypePHP {
		rootPath = normalizeWebsiteRootPath(cfg.OpenRestyDataDir, name, input.RootPath)
		if err := ensurePathWithin(filepath.Join(cfg.OpenRestyDataDir, "www"), rootPath); err != nil {
			return websitePayload{}, err
		}
	}

	indexFiles := normalizeIndexFilesByType(websiteType, input.IndexFiles)
	rewriteMode, rewritePreset, rewriteRules, err := normalizeRewriteConfig(input.RewriteMode, input.RewritePreset, input.RewriteRules)
	if err != nil {
		return websitePayload{}, err
	}

	phpProjectID := strings.TrimSpace(input.PHPProjectID)
	phpPort := 0
	if websiteType == websitesdomain.TypePHP {
		project, err := findPHPEnvironmentProject(ctx, projects, phpProjectID)
		if err != nil {
			return websitePayload{}, err
		}
		phpPort, err = projectConfigPort(project)
		if err != nil {
			return websitePayload{}, err
		}
		proxyPass = ""
	}
	if websiteType == websitesdomain.TypeProxy {
		rewriteMode = "off"
		rewritePreset = ""
		rewriteRules = ""
		rootPath = ""
		phpProjectID = ""
		phpPort = 0
	}

	existing, err := websites.List(ctx)
	if err != nil {
		return websitePayload{}, err
	}
	for _, item := range existing {
		if currentWebsiteID != "" && item.ID == currentWebsiteID {
			continue
		}
		if item.Name == name {
			return websitePayload{}, fmt.Errorf("网站名已存在")
		}
		for _, candidate := range append([]string{item.Domain}, item.Domains...) {
			if candidate == domain || slices.Contains(domains, candidate) {
				return websitePayload{}, fmt.Errorf("域名 %s 已存在", candidate)
			}
		}
	}

	return websitePayload{
		Name:          name,
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

func websiteSpecFromPayload(payload websitePayload) platformopenresty.WebsiteSpec {
	return platformopenresty.WebsiteSpec{
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

func websiteSpecFromWebsite(website websitesdomain.Website) platformopenresty.WebsiteSpec {
	return platformopenresty.WebsiteSpec{
		Name:          website.Name,
		Type:          website.Type,
		Domain:        website.Domain,
		Domains:       website.Domains,
		RootPath:      website.RootPath,
		IndexFiles:    website.IndexFiles,
		ProxyPass:     website.ProxyPass,
		PHPPort:       website.PHPPort,
		RewriteMode:   website.RewriteMode,
		RewritePreset: website.RewritePreset,
		RewriteRules:  website.RewriteRules,
	}
}
