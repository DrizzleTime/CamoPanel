package repo

import "time"

type ProviderRecord struct {
	ID        string `gorm:"primaryKey"`
	Name      string `gorm:"uniqueIndex;not null"`
	Type      string `gorm:"not null"`
	BaseURL   string `gorm:"not null"`
	APIKey    string `gorm:"type:text;not null"`
	Enabled   bool   `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ProviderRecord) TableName() string { return "ai_providers" }
