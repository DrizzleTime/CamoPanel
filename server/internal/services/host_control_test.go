package services

import (
	"strings"
	"testing"
)

func TestParseRegistryMirrors(t *testing.T) {
	mirrors, err := parseRegistryMirrors([]byte(`{"registry-mirrors":["https://b.example","https://a.example"]}`))
	if err != nil {
		t.Fatalf("parse registry mirrors: %v", err)
	}
	if len(mirrors) != 2 || mirrors[0] != "https://a.example" || mirrors[1] != "https://b.example" {
		t.Fatalf("unexpected mirrors: %#v", mirrors)
	}
}

func TestUpdateRegistryMirrorsPreservesOtherFields(t *testing.T) {
	content, err := updateRegistryMirrors([]byte(`{"log-driver":"json-file"}`), []string{
		"https://mirror-1.example",
		"https://mirror-1.example",
		" https://mirror-2.example ",
	})
	if err != nil {
		t.Fatalf("update registry mirrors: %v", err)
	}

	rendered := string(content)
	if !strings.Contains(rendered, `"log-driver": "json-file"`) {
		t.Fatalf("expected existing key to be preserved, got %s", rendered)
	}
	if !strings.Contains(rendered, `"registry-mirrors": [`) {
		t.Fatalf("expected registry mirrors to be written, got %s", rendered)
	}
	if strings.Count(rendered, "mirror-1.example") != 1 {
		t.Fatalf("expected duplicate mirror to be removed, got %s", rendered)
	}
}

func TestUpdateRegistryMirrorsDeletesEmptyField(t *testing.T) {
	content, err := updateRegistryMirrors([]byte(`{"registry-mirrors":["https://mirror.example"],"debug":true}`), nil)
	if err != nil {
		t.Fatalf("delete registry mirrors: %v", err)
	}

	rendered := string(content)
	if strings.Contains(rendered, "registry-mirrors") {
		t.Fatalf("expected registry mirrors field to be removed, got %s", rendered)
	}
	if !strings.Contains(rendered, `"debug": true`) {
		t.Fatalf("expected other fields to remain, got %s", rendered)
	}
}
