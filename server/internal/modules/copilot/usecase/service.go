package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	copilotdomain "camopanel/server/internal/modules/copilot/domain"
	"camopanel/server/internal/services"

	"github.com/google/uuid"
)

type Repository interface {
	ListProviders(ctx context.Context) ([]copilotdomain.Provider, error)
	FindProviderByID(ctx context.Context, providerID string) (copilotdomain.Provider, error)
	SaveProvider(ctx context.Context, item copilotdomain.Provider) error
	DeleteProvider(ctx context.Context, providerID string) error
	ListModels(ctx context.Context) ([]copilotdomain.Model, error)
	FindModelByID(ctx context.Context, modelID string) (copilotdomain.Model, error)
	SaveModel(ctx context.Context, item copilotdomain.Model) error
	DeleteModel(ctx context.Context, modelID string) error
}

type Service struct {
	repo     Repository
	fallback services.CopilotRuntimeConfig
}

func NewService(repo Repository, fallback services.CopilotRuntimeConfig) *Service {
	return &Service{repo: repo, fallback: fallback}
}

type ModelView struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"provider_id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ProviderView struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	BaseURL      string      `json:"base_url"`
	Enabled      bool        `json:"enabled"`
	HasAPIKey    bool        `json:"has_api_key"`
	APIKeyMasked string      `json:"api_key_masked"`
	Models       []ModelView `json:"models"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type ConfigStatus struct {
	Configured   bool   `json:"configured"`
	Source       string `json:"source"`
	ProviderID   string `json:"provider_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	ModelID      string `json:"model_id,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

func (s *Service) ListProviders(ctx context.Context) ([]ProviderView, error) {
	providers, err := s.repo.ListProviders(ctx)
	if err != nil {
		return nil, err
	}
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return nil, err
	}

	modelsByProvider := map[string][]ModelView{}
	for _, item := range models {
		modelsByProvider[item.ProviderID] = append(modelsByProvider[item.ProviderID], toModelView(item))
	}

	result := make([]ProviderView, 0, len(providers))
	for _, item := range providers {
		result = append(result, ProviderView{
			ID:           item.ID,
			Name:         item.Name,
			Type:         item.Type,
			BaseURL:      item.BaseURL,
			Enabled:      item.Enabled,
			HasAPIKey:    strings.TrimSpace(item.APIKey) != "",
			APIKeyMasked: maskSecret(item.APIKey),
			Models:       modelsByProvider[item.ID],
			CreatedAt:    item.CreatedAt,
			UpdatedAt:    item.UpdatedAt,
		})
	}
	return result, nil
}

func (s *Service) CreateProvider(ctx context.Context, name, providerType, baseURL, apiKey string, enabled bool) (ProviderView, error) {
	item := copilotdomain.Provider{
		ID:      uuid.NewString(),
		Name:    strings.TrimSpace(name),
		Type:    normalizeProviderType(providerType),
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:  strings.TrimSpace(apiKey),
		Enabled: enabled,
	}
	if err := validateProvider(item); err != nil {
		return ProviderView{}, err
	}
	if err := s.repo.SaveProvider(ctx, item); err != nil {
		return ProviderView{}, err
	}
	return ProviderView{
		ID:           item.ID,
		Name:         item.Name,
		Type:         item.Type,
		BaseURL:      item.BaseURL,
		Enabled:      item.Enabled,
		HasAPIKey:    item.APIKey != "",
		APIKeyMasked: maskSecret(item.APIKey),
		Models:       []ModelView{},
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}, nil
}

func (s *Service) UpdateProvider(ctx context.Context, providerID, name, providerType, baseURL, apiKey string, enabled bool) (ProviderView, error) {
	item, err := s.repo.FindProviderByID(ctx, providerID)
	if err != nil {
		return ProviderView{}, err
	}
	item.Name = strings.TrimSpace(name)
	item.Type = normalizeProviderType(providerType)
	item.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	item.Enabled = enabled
	if strings.TrimSpace(apiKey) != "" {
		item.APIKey = strings.TrimSpace(apiKey)
	}
	if err := validateProvider(item); err != nil {
		return ProviderView{}, err
	}
	if err := s.repo.SaveProvider(ctx, item); err != nil {
		return ProviderView{}, err
	}
	return s.providerView(ctx, item)
}

func (s *Service) DeleteProvider(ctx context.Context, providerID string) error {
	return s.repo.DeleteProvider(ctx, providerID)
}

func (s *Service) CreateModel(ctx context.Context, providerID, name string, enabled, isDefault bool) (ModelView, error) {
	provider, err := s.repo.FindProviderByID(ctx, providerID)
	if err != nil {
		return ModelView{}, err
	}
	item := copilotdomain.Model{
		ID:         uuid.NewString(),
		ProviderID: providerID,
		Name:       strings.TrimSpace(name),
		Enabled:    enabled,
		IsDefault:  isDefault,
	}
	if err := validateModel(provider, item); err != nil {
		return ModelView{}, err
	}
	if err := s.repo.SaveModel(ctx, item); err != nil {
		return ModelView{}, err
	}
	return toModelView(item), nil
}

func (s *Service) UpdateModel(ctx context.Context, modelID, name string, enabled, isDefault bool) (ModelView, error) {
	item, err := s.repo.FindModelByID(ctx, modelID)
	if err != nil {
		return ModelView{}, err
	}
	provider, err := s.repo.FindProviderByID(ctx, item.ProviderID)
	if err != nil {
		return ModelView{}, err
	}
	item.Name = strings.TrimSpace(name)
	item.Enabled = enabled
	item.IsDefault = isDefault
	if err := validateModel(provider, item); err != nil {
		return ModelView{}, err
	}
	if err := s.repo.SaveModel(ctx, item); err != nil {
		return ModelView{}, err
	}
	return toModelView(item), nil
}

func (s *Service) DeleteModel(ctx context.Context, modelID string) error {
	return s.repo.DeleteModel(ctx, modelID)
}

func (s *Service) ResolveCopilotRuntimeConfig(ctx context.Context) (services.CopilotRuntimeConfig, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return services.CopilotRuntimeConfig{}, err
	}
	for _, model := range models {
		if !model.IsDefault || !model.Enabled {
			continue
		}
		provider, err := s.repo.FindProviderByID(ctx, model.ProviderID)
		if err != nil {
			return services.CopilotRuntimeConfig{}, nil
		}
		if !provider.Enabled || provider.APIKey == "" || provider.BaseURL == "" {
			return services.CopilotRuntimeConfig{}, nil
		}
		return services.CopilotRuntimeConfig{
			Source:       "database",
			ProviderID:   provider.ID,
			ProviderName: provider.Name,
			ModelID:      model.ID,
			Model:        model.Name,
			BaseURL:      provider.BaseURL,
			APIKey:       provider.APIKey,
		}, nil
	}

	if s.fallback.BaseURL != "" && s.fallback.APIKey != "" && s.fallback.Model != "" {
		return s.fallback, nil
	}
	return services.CopilotRuntimeConfig{}, nil
}

func (s *Service) ConfigStatus(ctx context.Context) (ConfigStatus, error) {
	cfg, err := s.ResolveCopilotRuntimeConfig(ctx)
	if err != nil {
		return ConfigStatus{}, err
	}
	if cfg.Model != "" && cfg.BaseURL != "" && cfg.APIKey != "" {
		return ConfigStatus{
			Configured:   true,
			Source:       cfg.Source,
			ProviderID:   cfg.ProviderID,
			ProviderName: cfg.ProviderName,
			ModelID:      cfg.ModelID,
			ModelName:    cfg.Model,
			BaseURL:      cfg.BaseURL,
		}, nil
	}
	return ConfigStatus{Configured: false, Source: "none"}, nil
}

func (s *Service) providerView(ctx context.Context, provider copilotdomain.Provider) (ProviderView, error) {
	models, err := s.repo.ListModels(ctx)
	if err != nil {
		return ProviderView{}, err
	}
	items := []ModelView{}
	for _, item := range models {
		if item.ProviderID == provider.ID {
			items = append(items, toModelView(item))
		}
	}
	return ProviderView{
		ID:           provider.ID,
		Name:         provider.Name,
		Type:         provider.Type,
		BaseURL:      provider.BaseURL,
		Enabled:      provider.Enabled,
		HasAPIKey:    provider.APIKey != "",
		APIKeyMasked: maskSecret(provider.APIKey),
		Models:       items,
		CreatedAt:    provider.CreatedAt,
		UpdatedAt:    provider.UpdatedAt,
	}, nil
}

func toModelView(item copilotdomain.Model) ModelView {
	return ModelView{
		ID:         item.ID,
		ProviderID: item.ProviderID,
		Name:       item.Name,
		Enabled:    item.Enabled,
		IsDefault:  item.IsDefault,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}

func validateProvider(item copilotdomain.Provider) error {
	if item.Name == "" {
		return fmt.Errorf("服务商名称不能为空")
	}
	if item.Type != copilotdomain.ProviderTypeOpenAI {
		return fmt.Errorf("当前只支持 OpenAI Compatible 协议")
	}
	if item.BaseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}
	if item.APIKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	return nil
}

func validateModel(provider copilotdomain.Provider, item copilotdomain.Model) error {
	if item.Name == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if item.IsDefault && !item.Enabled {
		return fmt.Errorf("默认模型必须启用")
	}
	if item.IsDefault && !provider.Enabled {
		return fmt.Errorf("默认模型所属服务商必须启用")
	}
	return nil
}

func normalizeProviderType(value string) string {
	if strings.TrimSpace(value) == "" {
		return copilotdomain.ProviderTypeOpenAI
	}
	return strings.TrimSpace(value)
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "********" + value[len(value)-4:]
}
