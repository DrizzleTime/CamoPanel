package firewall

import "testing"

func TestDetectedBackendDoesNotSimulateRuleChanges(t *testing.T) {
	m := NewSystem(8080)
	m.lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	status := m.Status()
	if status.Available || status.Backend != "ufw" || status.Message == "" {
		t.Fatalf("unimplemented backend reported available: %+v", status)
	}
	for _, policy := range []string{"allow", "deny"} {
		if _, err := m.Add(8080, policy, true); err == nil {
			t.Fatal("unimplemented backend accepted a rule")
		}
	}
	if len(m.Status().Rules) != 0 {
		t.Fatal("unimplemented backend retained simulated rules")
	}
}
