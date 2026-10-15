package alert

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type Rule struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"`
	Threshold float64 `json:"threshold"`
	Webhook   string  `json:"webhook,omitempty"`
	SMTPHost  string  `json:"smtp_host,omitempty"`
}

type Manager struct {
	mu    sync.Mutex
	rules []Rule
	sent  map[string]int
}

func New() *Manager { return &Manager{rules: []Rule{}, sent: map[string]int{}} }

func (m *Manager) List() []Rule {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Rule{}, m.rules...)
}

func (m *Manager) Add(rule Rule) Rule {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	m.rules = append(m.rules, rule)
	return rule
}

type Sample struct {
	DiskPercent     float64 `json:"disk_percent"`
	CertDays        int     `json:"cert_days"`
	ContainerExited bool    `json:"container_exited"`
	SiteDown        bool    `json:"site_down"`
}

func (m *Manager) Evaluate(sample Sample) (int, error) {
	m.mu.Lock()
	rules := append([]Rule{}, m.rules...)
	m.mu.Unlock()
	sent := 0
	var first error
	for _, rule := range rules {
		if !firing(rule, sample) {
			continue
		}
		body, _ := json.Marshal(map[string]any{"rule": rule.ID, "kind": rule.Kind, "sample": sample})
		if rule.Webhook != "" {
			resp, err := http.Post(rule.Webhook, "application/json", bytes.NewReader(body))
			if err != nil {
				if first == nil {
					first = err
				}
			} else {
				resp.Body.Close()
				if resp.StatusCode >= 300 {
					if first == nil {
						first = fmt.Errorf("webhook %d", resp.StatusCode)
					}
				} else {
					m.mu.Lock()
					m.sent[rule.ID]++
					m.mu.Unlock()
					sent++
				}
			}
		}
		if rule.SMTPHost != "" {
			if err := sendSMTP(rule.SMTPHost, body); err != nil {
				if rule.Webhook == "" && first == nil {
					first = err
				}
			} else {
				sent++
			}
		}
	}
	return sent, first
}

func sendSMTP(host string, body []byte) error {
	conn, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	_, err = fmt.Fprintf(conn, "EHLO camopanel\r\nMAIL FROM:<panel@localhost>\r\nRCPT TO:<panel@localhost>\r\nDATA\r\n%s\r\n.\r\nQUIT\r\n", body)
	return err
}

func firing(rule Rule, sample Sample) bool {
	switch rule.Kind {
	case "disk":
		return sample.DiskPercent >= rule.Threshold
	case "cert_expiry":
		return float64(sample.CertDays) <= rule.Threshold
	case "container_exit":
		return sample.ContainerExited
	case "site_down":
		return sample.SiteDown
	default:
		return false
	}
}
