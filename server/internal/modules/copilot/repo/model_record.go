package repo

import "time"

type ModelRecord struct {
	ID         string `gorm:"primaryKey"`
	ProviderID string `gorm:"index;not null"`
	Name       string `gorm:"not null"`
	Enabled    bool   `gorm:"not null"`
	IsDefault  bool   `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (ModelRecord) TableName() string { return "ai_models" }
