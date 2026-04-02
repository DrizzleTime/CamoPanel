package domain

import (
	"errors"
	"time"
)

var ErrCertificateNotFound = errors.New("certificate not found")

type Certificate struct {
	ID             string
	Domain         string
	Email          string
	Provider       string
	Status         string
	FullchainPath  string
	PrivateKeyPath string
	LastError      string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
