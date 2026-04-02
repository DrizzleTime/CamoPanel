package auth

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

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidSession = errors.New("invalid session")

type SessionManager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewSessionManager(secret string) *SessionManager {
	return &SessionManager{
		secret: []byte(secret),
		ttl:    7 * 24 * time.Hour,
		now:    time.Now,
	}
}

func (m *SessionManager) Issue(userID string) (string, error) {
	expiry := m.now().Add(m.ttl).Unix()
	payload := fmt.Sprintf("%s|%d", userID, expiry)
	signature := m.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload + "|" + signature)), nil
}

func (m *SessionManager) Parse(token string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", ErrInvalidSession
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 3 {
		return "", ErrInvalidSession
	}

	payload := parts[0] + "|" + parts[1]
	expected := m.sign(payload)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[2])) != 1 {
		return "", ErrInvalidSession
	}

	expiry, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || m.now().Unix() > expiry {
		return "", ErrInvalidSession
	}

	return parts[0], nil
}

func (m *SessionManager) sign(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
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
