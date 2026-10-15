package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Record struct {
	Time   string `json:"time"`
	Actor  string `json:"actor"`
	Method string `json:"method"`
	Path   string `json:"path"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type Log struct {
	path string
	mu   sync.Mutex
}

func Open(dataDir string) (*Log, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &Log{path: filepath.Join(dataDir, "audit.jsonl")}, nil
}

func (l *Log) Append(rec Record) error {
	if l == nil {
		return nil
	}
	rec.Time = time.Now().UTC().Format(time.RFC3339)
	rec.Detail = redact(rec.Detail)
	rec.Path = redact(rec.Path)
	b, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func (l *Log) Path() string { return l.path }

func redact(value string) string {
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, secret := range []string{"password", "private key", "begin rsa", "begin private", "bearer ", "api_key", "totp"} {
		if strings.Contains(lower, secret) && strings.Contains(value, "\n") {
			return "[redacted]"
		}
	}
	return value
}
