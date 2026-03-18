package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const copilotProviderTypeOpenAI = "openai"

type copilotModelResponse struct {
	ID         string    `json:"id"`
	ProviderID string    `json:"provider_id"`
	Name       string    `json:"name"`
	Enabled    bool      `json:"enabled"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type copilotProviderResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	BaseURL      string                 `json:"base_url"`
	Enabled      bool                   `json:"enabled"`
	HasAPIKey    bool                   `json:"has_api_key"`
	APIKeyMasked string                 `json:"api_key_masked"`
	Models       []copilotModelResponse `json:"models"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type copilotConfigStatusResponse struct {
	Configured   bool   `json:"configured"`
	Source       string `json:"source"`
	ProviderID   string `json:"provider_id,omitempty"`
	ProviderName string `json:"provider_name,omitempty"`
	ModelID      string `json:"model_id,omitempty"`
	ModelName    string `json:"model_name,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
}

type createCopilotProviderRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Enabled bool   `json:"enabled"`
}

type updateCopilotProviderRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Enabled bool   `json:"enabled"`
}

type createCopilotModelRequest struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	IsDefault bool   `json:"is_default"`
}

type updateCopilotModelRequest struct {
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	IsDefault bool   `json:"is_default"`
}

func (a *App) listCopilotProviderResponses() ([]copilotProviderResponse, error) {
	var providers []model.AIProvider
	if err := a.db.Order("created_at desc").Find(&providers).Error; err != nil {
		return nil, err
	}

	var models []model.AIModel
	if err := a.db.Order("is_default desc, created_at asc").Find(&models).Error; err != nil {
		return nil, err
	}

	modelsByProvider := map[string][]copilotModelResponse{}
	for _, item := range models {
		modelsByProvider[item.ProviderID] = append(modelsByProvider[item.ProviderID], copilotModelToResponse(item))
	}

	items := make([]copilotProviderResponse, 0, len(providers))
	for _, provider := range providers {
		items = append(items, copilotProviderResponse{
			ID:           provider.ID,
			Name:         provider.Name,
			Type:         provider.Type,
			BaseURL:      provider.BaseURL,
			Enabled:      provider.Enabled,
			HasAPIKey:    provider.APIKey != "",
			APIKeyMasked: maskSecret(provider.APIKey),
			Models:       modelsByProvider[provider.ID],
			CreatedAt:    provider.CreatedAt,
			UpdatedAt:    provider.UpdatedAt,
		})
	}

	return items, nil
}

func (a *App) findCopilotProvider(providerID string) (model.AIProvider, error) {
	var provider model.AIProvider
	if err := a.db.First(&provider, "id = ?", providerID).Error; err != nil {
		return model.AIProvider{}, fmt.Errorf("模型服务不存在")
	}
	return provider, nil
}

func (a *App) findCopilotModel(modelID string) (model.AIModel, error) {
	var aiModel model.AIModel
	if err := a.db.First(&aiModel, "id = ?", modelID).Error; err != nil {
		return model.AIModel{}, fmt.Errorf("模型不存在")
	}
	return aiModel, nil
}

func (a *App) createCopilotProvider(actorID string, req createCopilotProviderRequest) (copilotProviderResponse, error) {
	provider := model.AIProvider{
		ID:      uuid.NewString(),
		Name:    strings.TrimSpace(req.Name),
		Type:    normalizeCopilotProviderType(req.Type),
		BaseURL: strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"),
		APIKey:  strings.TrimSpace(req.APIKey),
		Enabled: req.Enabled,
	}
	if err := validateCopilotProvider(provider, true); err != nil {
		return copilotProviderResponse{}, err
	}

	if err := a.db.Create(&provider).Error; err != nil {
		return copilotProviderResponse{}, fmt.Errorf("保存模型服务失败")
	}

	_ = a.recordAudit(actorID, "copilot_provider_create", "ai_provider", provider.ID, map[string]any{
		"name": provider.Name,
		"type": provider.Type,
	})

	return a.buildCopilotProviderResponse(provider), nil
}

func (a *App) updateCopilotProvider(actorID string, provider model.AIProvider, req updateCopilotProviderRequest) (copilotProviderResponse, error) {
	provider.Name = strings.TrimSpace(req.Name)
	provider.Type = normalizeCopilotProviderType(req.Type)
	provider.BaseURL = strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	provider.Enabled = req.Enabled
	if strings.TrimSpace(req.APIKey) != "" {
		provider.APIKey = strings.TrimSpace(req.APIKey)
	}
	if err := validateCopilotProvider(provider, false); err != nil {
		return copilotProviderResponse{}, err
	}

	if err := a.db.Save(&provider).Error; err != nil {
		return copilotProviderResponse{}, fmt.Errorf("更新模型服务失败")
	}

	_ = a.recordAudit(actorID, "copilot_provider_update", "ai_provider", provider.ID, map[string]any{
		"name": provider.Name,
		"type": provider.Type,
	})

	return a.buildCopilotProviderResponse(provider), nil
}

func (a *App) deleteCopilotProvider(actorID string, provider model.AIProvider) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("provider_id = ?", provider.ID).Delete(&model.AIModel{}).Error; err != nil {
			return fmt.Errorf("删除模型失败")
		}
		if err := tx.Delete(&provider).Error; err != nil {
			return fmt.Errorf("删除模型服务失败")
		}
		return nil
	})
}

func (a *App) createCopilotModel(actorID string, provider model.AIProvider, req createCopilotModelRequest) (copilotModelResponse, error) {
	aiModel := model.AIModel{
		ID:         uuid.NewString(),
		ProviderID: provider.ID,
		Name:       strings.TrimSpace(req.Name),
		Enabled:    req.Enabled,
		IsDefault:  req.IsDefault,
	}
	if err := validateCopilotModel(provider, aiModel); err != nil {
		return copilotModelResponse{}, err
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if aiModel.IsDefault {
			if err := tx.Model(&model.AIModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&aiModel).Error
	}); err != nil {
		return copilotModelResponse{}, fmt.Errorf("保存模型失败")
	}

	_ = a.recordAudit(actorID, "copilot_model_create", "ai_model", aiModel.ID, map[string]any{
		"name":        aiModel.Name,
		"provider_id": provider.ID,
	})

	return copilotModelToResponse(aiModel), nil
}

func (a *App) updateCopilotModel(actorID string, aiModel model.AIModel, req updateCopilotModelRequest) (copilotModelResponse, error) {
	provider, err := a.findCopilotProvider(aiModel.ProviderID)
	if err != nil {
		return copilotModelResponse{}, err
	}

	aiModel.Name = strings.TrimSpace(req.Name)
	aiModel.Enabled = req.Enabled
	aiModel.IsDefault = req.IsDefault
	if err := validateCopilotModel(provider, aiModel); err != nil {
		return copilotModelResponse{}, err
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		if aiModel.IsDefault {
			if err := tx.Model(&model.AIModel{}).Where("id <> ? AND is_default = ?", aiModel.ID, true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(&aiModel).Error
	}); err != nil {
		return copilotModelResponse{}, fmt.Errorf("更新模型失败")
	}

	_ = a.recordAudit(actorID, "copilot_model_update", "ai_model", aiModel.ID, map[string]any{
		"name":        aiModel.Name,
		"provider_id": aiModel.ProviderID,
	})

	return copilotModelToResponse(aiModel), nil
}

func (a *App) deleteCopilotModel(actorID string, aiModel model.AIModel) error {
	if err := a.db.Delete(&aiModel).Error; err != nil {
		return fmt.Errorf("删除模型失败")
	}

	_ = a.recordAudit(actorID, "copilot_model_delete", "ai_model", aiModel.ID, map[string]any{
		"name":        aiModel.Name,
		"provider_id": aiModel.ProviderID,
	})
	return nil
}

func (a *App) ResolveCopilotRuntimeConfig(ctx context.Context) (services.CopilotRuntimeConfig, error) {
	var aiModel model.AIModel
	if err := a.db.Where("is_default = ? AND enabled = ?", true, true).Order("updated_at desc").First(&aiModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services.CopilotRuntimeConfig{}, nil
		}
		return services.CopilotRuntimeConfig{}, err
	}

	provider, err := a.findCopilotProvider(aiModel.ProviderID)
	if err != nil {
		return services.CopilotRuntimeConfig{}, nil
	}
	if !provider.Enabled || provider.APIKey == "" || provider.BaseURL == "" {
		return services.CopilotRuntimeConfig{}, nil
	}

	_ = ctx
	return services.CopilotRuntimeConfig{
		Source:       "database",
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ModelID:      aiModel.ID,
		Model:        aiModel.Name,
		BaseURL:      provider.BaseURL,
		APIKey:       provider.APIKey,
	}, nil
}

func (a *App) copilotConfigStatus(ctx context.Context) (copilotConfigStatusResponse, error) {
	runtimeConfig, err := a.ResolveCopilotRuntimeConfig(ctx)
	if err != nil {
		return copilotConfigStatusResponse{}, err
	}
	if runtimeConfig.Model != "" && runtimeConfig.BaseURL != "" && runtimeConfig.APIKey != "" {
		return copilotConfigStatusResponse{
			Configured:   true,
			Source:       runtimeConfig.Source,
			ProviderID:   runtimeConfig.ProviderID,
			ProviderName: runtimeConfig.ProviderName,
			ModelID:      runtimeConfig.ModelID,
			ModelName:    runtimeConfig.Model,
			BaseURL:      runtimeConfig.BaseURL,
		}, nil
	}

	if a.cfg.AI.Model != "" && a.cfg.AI.BaseURL != "" && a.cfg.AI.APIKey != "" {
		return copilotConfigStatusResponse{
			Configured:   true,
			Source:       "env",
			ProviderName: "环境变量",
			ModelName:    a.cfg.AI.Model,
			BaseURL:      a.cfg.AI.BaseURL,
		}, nil
	}

	return copilotConfigStatusResponse{Configured: false, Source: "none"}, nil
}

func (a *App) buildCopilotProviderResponse(provider model.AIProvider) copilotProviderResponse {
	return copilotProviderResponse{
		ID:           provider.ID,
		Name:         provider.Name,
		Type:         provider.Type,
		BaseURL:      provider.BaseURL,
		Enabled:      provider.Enabled,
		HasAPIKey:    provider.APIKey != "",
		APIKeyMasked: maskSecret(provider.APIKey),
		Models:       []copilotModelResponse{},
		CreatedAt:    provider.CreatedAt,
		UpdatedAt:    provider.UpdatedAt,
	}
}

func copilotModelToResponse(aiModel model.AIModel) copilotModelResponse {
	return copilotModelResponse{
		ID:         aiModel.ID,
		ProviderID: aiModel.ProviderID,
		Name:       aiModel.Name,
		Enabled:    aiModel.Enabled,
		IsDefault:  aiModel.IsDefault,
		CreatedAt:  aiModel.CreatedAt,
		UpdatedAt:  aiModel.UpdatedAt,
	}
}

func validateCopilotProvider(provider model.AIProvider, requireAPIKey bool) error {
	if provider.Name == "" {
		return fmt.Errorf("服务商名称不能为空")
	}
	if provider.Type != copilotProviderTypeOpenAI {
		return fmt.Errorf("当前只支持 OpenAI Compatible 协议")
	}
	if provider.BaseURL == "" {
		return fmt.Errorf("Base URL 不能为空")
	}
	if requireAPIKey && provider.APIKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	if provider.APIKey == "" {
		return fmt.Errorf("API Key 不能为空")
	}
	return nil
}

func validateCopilotModel(provider model.AIProvider, aiModel model.AIModel) error {
	if aiModel.Name == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if aiModel.IsDefault && !aiModel.Enabled {
		return fmt.Errorf("默认模型必须启用")
	}
	if aiModel.IsDefault && !provider.Enabled {
		return fmt.Errorf("默认模型所属服务商必须启用")
	}
	return nil
}

func normalizeCopilotProviderType(value string) string {
	if strings.TrimSpace(value) == "" {
		return copilotProviderTypeOpenAI
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
