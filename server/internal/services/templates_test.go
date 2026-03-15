package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateCatalogValidateAndRender(t *testing.T) {
	root := t.TempDir()
	templateDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("mkdir template dir: %v", err)
	}

	spec := `id: demo
name: Demo
version: "1"
description: test
params:
  - name: port
    type: number
    required: true
  - name: password
    type: secret
    required: true
`
	compose := `services:
  app:
    image: nginx
    ports:
      - "{{ .Values.port }}:80"
    environment:
      PASSWORD: {{ .Values.password }}
`
	if err := os.WriteFile(filepath.Join(templateDir, "template.yaml"), []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	if err := os.WriteFile(filepath.Join(templateDir, "compose.yaml.tmpl"), []byte(compose), 0o644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	catalog, err := NewTemplateCatalog(root)
	if err != nil {
		t.Fatalf("new catalog: %v", err)
	}

	item, err := catalog.Get("demo")
	if err != nil {
		t.Fatalf("get template: %v", err)
	}

	normalized, err := item.ValidateAndNormalize(map[string]any{
		"port":     8080,
		"password": "secret",
	})
	if err != nil {
		t.Fatalf("validate input: %v", err)
	}

	rendered, err := item.Render(normalized, TemplateRuntime{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(rendered, "8080:80") {
		t.Fatalf("expected rendered compose to include port mapping, got %s", rendered)
	}
	if !strings.Contains(rendered, "PASSWORD: secret") {
		t.Fatalf("expected rendered compose to include password, got %s", rendered)
	}
}

func TestTemplateCatalogRejectsMissingRequiredParam(t *testing.T) {
	item := LoadedTemplate{
		Spec: TemplateSpec{
			ID: "demo",
			Params: []TemplateParam{
				{Name: "password", Type: ParamSecret, Required: true},
			},
		},
	}

	if _, err := item.ValidateAndNormalize(map[string]any{}); err == nil {
		t.Fatalf("expected missing required param error")
	}
}

func TestTemplateRenderSupportsRuntimeContext(t *testing.T) {
	item := LoadedTemplate{
		ComposeTemplate: `services:
  app:
    container_name: {{ .Runtime.OpenRestyContainer }}
    volumes:
      - "{{ .Runtime.OpenRestyHostConfDir }}:/etc/nginx/conf.d"
`,
	}

	rendered, err := item.Render(map[string]any{}, TemplateRuntime{
		OpenRestyContainer:   "camopanel-openresty",
		OpenRestyHostConfDir: "/var/lib/camopanel/openresty/conf.d",
	})
	if err != nil {
		t.Fatalf("render with runtime: %v", err)
	}

	if !strings.Contains(rendered, "container_name: camopanel-openresty") {
		t.Fatalf("expected rendered compose to include container name, got %s", rendered)
	}
	if !strings.Contains(rendered, "/var/lib/camopanel/openresty/conf.d:/etc/nginx/conf.d") {
		t.Fatalf("expected rendered compose to include conf dir bind, got %s", rendered)
	}
}
