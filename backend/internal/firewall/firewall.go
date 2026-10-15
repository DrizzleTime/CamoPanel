package firewall

import (
	"fmt"
	"os/exec"
)

type Rule struct {
	Port   int    `json:"port"`
	Policy string `json:"policy"`
}

type Manager struct {
	panelPort int
	lookPath  func(string) (string, error)
}

func NewSystem(panelPort int) *Manager {
	return &Manager{panelPort: panelPort, lookPath: exec.LookPath}
}

type Status struct {
	Available bool   `json:"available"`
	Backend   string `json:"backend"`
	Message   string `json:"message,omitempty"`
	Rules     []Rule `json:"rules"`
	PanelPort int    `json:"panel_port"`
}

func (m *Manager) Status() Status {
	backend, err := detect(m.lookPath)
	if err != nil {
		return Status{Available: false, Message: "当前系统没有可管理的防火墙", Rules: []Rule{}, PanelPort: m.panelPort}
	}
	return Status{Available: false, Backend: backend, Message: "防火墙规则管理尚未接入系统后端", Rules: []Rule{}, PanelPort: m.panelPort}
}

func detect(lookPath func(string) (string, error)) (string, error) {
	for _, name := range []string{"ufw", "firewall-cmd", "iptables"} {
		if _, err := lookPath(name); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("missing")
}

func (m *Manager) Add(_ int, policy string, _ bool) (Status, error) {
	if policy != "allow" && policy != "deny" {
		return Status{}, fmt.Errorf("规则只支持 allow 或 deny")
	}
	status := m.Status()
	return status, fmt.Errorf("%s", status.Message)
}
