package repo

import "camopanel/server/internal/modules/auth/domain"

type UserRecord struct {
	ID           string `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null"`
}

func (UserRecord) TableName() string {
	return "users"
}

func (r UserRecord) toDomain() domain.User {
	return domain.User{
		ID:           r.ID,
		Username:     r.Username,
		Role:         r.Role,
		PasswordHash: r.PasswordHash,
	}
}
