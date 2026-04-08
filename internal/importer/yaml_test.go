package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestYAMLDiscoverFromFile(t *testing.T) {
	content := `- name: prod
  env: production
  ssh:
    host: bastion.example.com
    user: deploy
  db:
    host: 10.0.0.5
    port: 3306
    user: app
    database: myapp_production
  mask:
    columns: [email, phone]
- name: staging
  env: staging
  db:
    host: localhost
    port: 3306
    user: root
    database: myapp_staging
`

	dir := t.TempDir()
	path := filepath.Join(dir, "connections.yaml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	p := &yamlProvider{}
	conns, err := p.DiscoverFromFile(path)
	if err != nil {
		t.Fatalf("DiscoverFromFile: %v", err)
	}

	if len(conns) != 2 {
		t.Fatalf("got %d connections, want 2", len(conns))
	}

	// Check first connection
	if conns[0].Name != "prod" {
		t.Errorf("name = %q, want %q", conns[0].Name, "prod")
	}
	if conns[0].Env != "production" {
		t.Errorf("env = %q, want %q", conns[0].Env, "production")
	}
	if conns[0].SSH == nil {
		t.Error("SSH should not be nil")
	} else if conns[0].SSH.Host != "bastion.example.com" {
		t.Errorf("SSH host = %q, want %q", conns[0].SSH.Host, "bastion.example.com")
	}
	if conns[0].Mask == nil {
		t.Error("Mask should not be nil")
	} else if len(conns[0].Mask.Columns) != 2 {
		t.Errorf("mask columns = %d, want 2", len(conns[0].Mask.Columns))
	}
	if conns[0].DB.Password != "" {
		t.Error("password should be empty in exported YAML")
	}

	// Check second connection
	if conns[1].Name != "staging" {
		t.Errorf("name = %q, want %q", conns[1].Name, "staging")
	}
	if conns[1].SSH != nil {
		t.Error("SSH should be nil for staging")
	}
}

func TestYAMLDiscoverFromFileNotFound(t *testing.T) {
	p := &yamlProvider{}
	_, err := p.DiscoverFromFile("/nonexistent/file.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestYAMLDiscoverFromFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("not: [valid: yaml: list"), 0600); err != nil {
		t.Fatal(err)
	}

	p := &yamlProvider{}
	_, err := p.DiscoverFromFile(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestYAMLDiscoverFromFileEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}

	p := &yamlProvider{}
	conns, err := p.DiscoverFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conns != nil {
		t.Errorf("expected nil, got %d connections", len(conns))
	}
}

func TestYAMLDiscoverRequiresFile(t *testing.T) {
	p := &yamlProvider{}
	_, err := p.Discover()
	if err == nil {
		t.Error("expected error from Discover() without file")
	}
}
