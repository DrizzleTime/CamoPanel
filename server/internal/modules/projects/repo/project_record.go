package repo

import "time"

type ProjectRecord struct {
	ID              string `gorm:"primaryKey"`
	Name            string `gorm:"uniqueIndex;not null"`
	Kind            string `gorm:"not null"`
	TemplateID      string `gorm:"not null"`
	TemplateVersion string `gorm:"not null"`
	ConfigJSON      string `gorm:"type:text;not null"`
	ComposePath     string `gorm:"not null"`
	Status          string `gorm:"not null"`
	LastError       string `gorm:"type:text"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (ProjectRecord) TableName() string {
	return "projects"
}
