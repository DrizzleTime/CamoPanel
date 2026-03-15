package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidSession = errors.New("invalid session")

type AuthService struct {
	secret []byte
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{secret: []byte(secret)}
}

func HashPassword(raw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, raw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(raw)) == nil
}

func (s *AuthService) IssueSession(user model.User) (string, error) {
	expiry := time.Now().Add(7 * 24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", user.ID, expiry)
	signature := s.sign(payload)
	token := base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + signature))
	return token, nil
}

func (s *AuthService) ParseSession(token string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalidSession
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 {
		return "", ErrInvalidSession
	}

	payload := parts[0] + "|" + parts[1]
	expected := s.sign(payload)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return "", ErrInvalidSession
	}

	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return "", ErrInvalidSession
	}

	return parts[0], nil
}

func (s *AuthService) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
