package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

type downloadClaim struct {
	root, path, actor, role string
	expires                 time.Time
}
type fileDownloads struct {
	mu    sync.Mutex
	links map[string]downloadClaim
}

func newFileDownloads() *fileDownloads { return &fileDownloads{links: make(map[string]downloadClaim)} }
func (d *fileDownloads) issue(claim downloadClaim) (string, error) {
	token := make([]byte, 24)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	for key, entry := range d.links {
		if now.After(entry.expires) {
			delete(d.links, key)
		}
	}
	if len(d.links) >= 512 {
		return "", fmt.Errorf("待下载任务过多，请稍后重试")
	}
	claim.expires = now.Add(2 * time.Minute)
	ticket := hex.EncodeToString(token)
	d.links[ticket] = claim
	return ticket, nil
}
func (d *fileDownloads) take(ticket string) (downloadClaim, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	claim, ok := d.links[ticket]
	if ok {
		delete(d.links, ticket)
	}
	return claim, ok && time.Now().Before(claim.expires)
}
