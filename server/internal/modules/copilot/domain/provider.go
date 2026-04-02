package domain

import (
	"errors"
	"time"
)

const ProviderTypeOpenAI = "openai"

var (
	ErrProviderNotFound = errors.New("copilot provider not found")
	ErrModelNotFound    = errors.New("copilot model not found")
)

type Provider struct {
	ID        string
	Name      string
	Type      string
	BaseURL   string
	APIKey    string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Model struct {
	ID         string
	ProviderID string
	Name       string
	Enabled    bool
	IsDefault  bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
