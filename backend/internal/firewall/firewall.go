package firewall

import (
	"fmt"
	"os/exec"
	"sync"
)

type Rule struct {
	Port   int    `json:"port"`
	Policy string `json:"policy"`
}

type Manager struct {
	mu        sync.Mutex
	demo      bool
	panelPort int
	rules     []Rule
	lookPath  func(string) (string, error)
}

func NewDemo(panelPort int) *Manager {
	return &Manager{demo: true, panelPort: panelPort, rules: []Rule{}, lookPath: exec.LookPath}
}

func NewSystem(panelPort int) *Manager {
	return &Manager{demo: false, panelPort: panelPort, rules: []Rule{}, lookPath: exec.LookPath}
}

type Status struct {
	Available bool   `json:"available"`
	Backend   string `json:"backend"`
	Message   string `json:"message,omitempty"`
	Rules     []Rule `json:"rules"`
	PanelPort int    `json:"panel_port"`
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.demo {
		return Status{Available: true, Backend: "demo", Rules: append([]Rule{}, m.rules...), PanelPort: m.panelPort}
	}
	backend, err := detect(m.lookPath)
	if err != nil {
		return Status{Available: false, Message: "当前系统没有可管理的防火墙", Rules: []Rule{}, PanelPort: m.panelPort}
	}
	return Status{Available: true, Backend: backend, Rules: append([]Rule{}, m.rules...), PanelPort: m.panelPort}
}

func detect(lookPath func(string) (string, error)) (string, error) {
	for _, name := range []string{"ufw", "firewall-cmd", "iptables"} {
		if _, err := lookPath(name); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("missing")
}

func (m *Manager) Add(port int, policy string, confirm bool) (Status, error) {
	if policy != "allow" && policy != "deny" {
		return Status{}, fmt.Errorf("规则只支持 allow 或 deny")
	}
	status := m.Status()
	if !status.Available {
		return status, fmt.Errorf("%s", status.Message)
	}
	if policy == "deny" && port == m.panelPort && !confirm {
		return status, fmt.Errorf("这条规则会关掉面板端口 %d，确认后才会保存", m.panelPort)
	}
	m.mu.Lock()
	m.rules = append(m.rules, Rule{Port: port, Policy: policy})
	m.mu.Unlock()
	return m.Status(), nil
}
