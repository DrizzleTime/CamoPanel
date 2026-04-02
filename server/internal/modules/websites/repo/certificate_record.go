package repo

import "time"

type CertificateRecord struct {
	ID             string `gorm:"primaryKey"`
	Domain         string `gorm:"uniqueIndex;not null"`
	Email          string `gorm:"not null"`
	Provider       string `gorm:"not null"`
	Status         string `gorm:"not null"`
	FullchainPath  string `gorm:"not null"`
	PrivateKeyPath string `gorm:"not null"`
	LastError      string `gorm:"type:text"`
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (CertificateRecord) TableName() string {
	return "certificates"
}
