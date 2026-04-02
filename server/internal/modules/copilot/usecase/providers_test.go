package usecase_test

import (
	"context"
	"testing"

	copilotdomain "camopanel/server/internal/modules/copilot/domain"
	copilotusecase "camopanel/server/internal/modules/copilot/usecase"
	"camopanel/server/internal/services"
)

type copilotRepoStub struct {
	providers []copilotdomain.Provider
	models    []copilotdomain.Model
}

func (s *copilotRepoStub) ListProviders(context.Context) ([]copilotdomain.Provider, error) {
	return append([]copilotdomain.Provider(nil), s.providers...), nil
}
func (s *copilotRepoStub) FindProviderByID(_ context.Context, providerID string) (copilotdomain.Provider, error) {
	for _, item := range s.providers {
		if item.ID == providerID {
			return item, nil
		}
	}
	return copilotdomain.Provider{}, copilotdomain.ErrProviderNotFound
}
func (s *copilotRepoStub) SaveProvider(_ context.Context, item copilotdomain.Provider) error {
	s.providers = append(s.providers, item)
	return nil
}
func (s *copilotRepoStub) DeleteProvider(context.Context, string) error { return nil }
func (s *copilotRepoStub) ListModels(context.Context) ([]copilotdomain.Model, error) {
	return append([]copilotdomain.Model(nil), s.models...), nil
}
func (s *copilotRepoStub) FindModelByID(_ context.Context, modelID string) (copilotdomain.Model, error) {
	for _, item := range s.models {
		if item.ID == modelID {
			return item, nil
		}
	}
	return copilotdomain.Model{}, copilotdomain.ErrModelNotFound
}
func (s *copilotRepoStub) SaveModel(_ context.Context, item copilotdomain.Model) error {
	s.models = append(s.models, item)
	return nil
}
func (s *copilotRepoStub) DeleteModel(context.Context, string) error { return nil }

func TestConfigStatusUsesFallbackWhenNoDatabaseConfig(t *testing.T) {
	svc := copilotusecase.NewService(&copilotRepoStub{}, services.CopilotRuntimeConfig{
		Source:       "env",
		ProviderName: "环境变量",
		Model:        "gpt-4.1",
		BaseURL:      "https://example.com",
		APIKey:       "secret",
	})

	item, err := svc.ConfigStatus(context.Background())
	if err != nil {
		t.Fatalf("config status: %v", err)
	}
	if !item.Configured || item.Source != "env" {
		t.Fatalf("unexpected status: %+v", item)
	}
}
