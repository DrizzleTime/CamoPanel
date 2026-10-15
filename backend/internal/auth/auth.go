package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const SessionCookie = "camo_session"

var (
	ErrBadPassword = errors.New("用户名或密码错误")
	ErrTOTP        = errors.New("需要二次验证码")
	ErrExists      = errors.New("管理员已初始化")
	ErrEnvironment = errors.New("账号由环境变量管理，请修改 .env 并重启服务")
)

type User struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
	TOTPSecret   string `json:"totp_secret,omitempty"`
	TOTPEnabled  bool   `json:"totp_enabled"`
}

type fileState struct {
	Admin  User `json:"admin"`
	Viewer User `json:"viewer"`
}

type Session struct {
	ID        string
	Username  string
	Role      string
	ExpiresAt time.Time
}

type Store struct {
	path     string
	mu       sync.Mutex
	state    fileState
	sessions map[string]Session
	envAdmin *User
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: filepath.Join(dataDir, "auth.json"), sessions: map[string]Session{}}
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &s.state); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// OpenWithCredentials keeps environment credentials in memory. Only optional
// authentication metadata (e.g. TOTP) may be persisted in auth.json.
func OpenWithCredentials(dataDir, username, password string) (*Store, error) {
	store, err := Open(dataDir)
	if err != nil {
		return nil, err
	}
	if password == "" {
		return store, nil
	}
	if strings.TrimSpace(username) == "" || len(password) < 8 || len(password) > 72 {
		return nil, fmt.Errorf("invalid environment login configuration")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	admin := User{Username: username, PasswordHash: string(hash)}
	if store.state.Admin.Username == username {
		admin.TOTPSecret = store.state.Admin.TOTPSecret
		admin.TOTPEnabled = store.state.Admin.TOTPEnabled
	}
	store.envAdmin = &admin
	return store, nil
}

func (s *Store) EnvironmentManaged() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.envAdmin != nil
}

func PasswordConfigured(dataDir string) bool {
	b, err := os.ReadFile(filepath.Join(dataDir, "auth.json"))
	if err != nil {
		return false
	}
	var state fileState
	if json.Unmarshal(b, &state) != nil {
		return false
	}
	return state.Admin.PasswordHash != ""
}

func (s *Store) HasAdmin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.envAdmin != nil || s.state.Admin.PasswordHash != ""
}

func (s *Store) Bootstrap(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envAdmin != nil {
		return ErrEnvironment
	}
	if s.state.Admin.PasswordHash != "" {
		return ErrExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.state.Admin = User{Username: "admin", PasswordHash: string(hash)}
	return s.save()
}

func (s *Store) SetViewer(password string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envAdmin != nil {
		return ErrEnvironment
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.state.Viewer = User{Username: "viewer", PasswordHash: string(hash)}
	return s.save()
}

func (s *Store) ChangePassword(username, current, next string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envAdmin != nil {
		return ErrEnvironment
	}
	user := s.user(username)
	if user == nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)) != nil {
		return ErrBadPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	return s.save()
}

func (s *Store) Login(username, password, code string, now time.Time) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.user(username)
	role := "admin"
	if user == &s.state.Viewer {
		role = "viewer"
	}
	if user == nil || user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return Session{}, ErrBadPassword
	}
	if role == "admin" && user.TOTPEnabled {
		if strings.TrimSpace(code) == "" || !validTOTP(user.TOTPSecret, code, now) {
			return Session{}, ErrTOTP
		}
	}
	id, err := randomID()
	if err != nil {
		return Session{}, err
	}
	for id, session := range s.sessions {
		if !now.Before(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	sess := Session{ID: id, Username: user.Username, Role: role, ExpiresAt: now.Add(24 * time.Hour)}
	s.sessions[id] = sess
	return sess, nil
}

func (s *Store) Lookup(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if ok && !time.Now().Before(sess.ExpiresAt) {
		delete(s.sessions, id)
		return Session{}, false
	}
	return sess, ok
}

func (s *Store) Logout(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

func (s *Store) BeginTOTP() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	admin := &s.state.Admin
	if s.envAdmin != nil {
		admin = s.envAdmin
	}
	admin.TOTPSecret = secret
	admin.TOTPEnabled = false
	if err := s.save(); err != nil {
		return "", err
	}
	return secret, nil
}

func (s *Store) EnableTOTP(code string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	admin := &s.state.Admin
	if s.envAdmin != nil {
		admin = s.envAdmin
	}
	if admin.TOTPSecret == "" || !validTOTP(admin.TOTPSecret, code, now) {
		return ErrTOTP
	}
	admin.TOTPEnabled = true
	return s.save()
}

func (s *Store) TOTPEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envAdmin != nil {
		return s.envAdmin.TOTPEnabled
	}
	return s.state.Admin.TOTPEnabled
}

func (s *Store) user(username string) *User {
	if s.envAdmin != nil {
		if username == s.envAdmin.Username {
			return s.envAdmin
		}
		return nil
	}
	switch username {
	case "admin", "":
		return &s.state.Admin
	case "viewer":
		return &s.state.Viewer
	default:
		if username == s.state.Admin.Username {
			return &s.state.Admin
		}
		if username == s.state.Viewer.Username {
			return &s.state.Viewer
		}
		return nil
	}
}

func (s *Store) save() error {
	state := s.state
	if s.envAdmin != nil {
		state.Admin = *s.envAdmin
		state.Admin.PasswordHash = ""
		state.Viewer = User{}
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// CodeAt is the TOTP code the login handler checks.
func CodeAt(secret string, now time.Time) (string, error) {
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(now.Unix() / 30)
	return hotp(raw, counter), nil
}

func validTOTP(secret, code string, now time.Time) bool {
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return false
	}
	counter := uint64(now.Unix() / 30)
	for _, delta := range []int64{-1, 0, 1} {
		c := hotp(raw, uint64(int64(counter)+delta))
		if subtle.ConstantTimeCompare([]byte(c), []byte(strings.TrimSpace(code))) == 1 {
			return true
		}
	}
	return false
}

func hotp(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1000000)
}
