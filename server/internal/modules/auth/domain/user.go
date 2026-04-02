package domain

import "errors"

var ErrUserNotFound = errors.New("user not found")

const RoleSuperAdmin = "super_admin"

type User struct {
	ID           string
	Username     string
	Role         string
	PasswordHash string
}
