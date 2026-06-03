package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/format"
)

func TestObjectSchema(t *testing.T) {
	schema := objectSchema(map[string]any{
		"sql": map[string]any{"type": "string"},
	}, []string{"sql"})
	if schema["type"] != "object" {
		t.Errorf("expected object type, got %v", schema["type"])
	}
	req, ok := schema["required"].([]any)
	if !ok || len(req) != 1 || req[0] != "sql" {
		t.Errorf("required not set correctly: %v", schema["required"])
	}
}

func TestObjectSchemaNilProperties(t *testing.T) {
	schema := objectSchema(nil, nil)
	props, ok := schema["properties"].(map[string]any)
	if !ok || len(props) != 0 {
		t.Errorf("expected empty properties object, got %v", schema["properties"])
	}
	if _, ok := schema["required"]; ok {
		t.Error("required should be omitted when empty")
	}
}

func TestArgString(t *testing.T) {
	args := map[string]any{"a": "x", "n": 1}
	if got := argString(args, "a"); got != "x" {
		t.Errorf("argString a = %q, want x", got)
	}
	if got := argString(args, "n"); got != "" {
		t.Errorf("argString of non-string should be empty, got %q", got)
	}
	if got := argString(nil, "a"); got != "" {
		t.Errorf("argString(nil) should be empty, got %q", got)
	}
}

func TestMCPOutputFormat(t *testing.T) {
	cases := []struct {
		in      string
		want    format.Type
		wantErr bool
	}{
		{"", format.Markdown, false},
		{"markdown", format.Markdown, false},
		{"json", format.JSON, false},
		{"csv", format.CSV, false},
		{"plain", format.Plain, false},
		{"pdf", "", true},
		{"bogus", "", true},
	}
	for _, c := range cases {
		args := map[string]any{}
		if c.in != "" {
			args["format"] = c.in
		}
		got, err := mcpOutputFormat(args)
		if c.wantErr {
			if err == nil {
				t.Errorf("mcpOutputFormat(%q) expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("mcpOutputFormat(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("mcpOutputFormat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormatStructured(t *testing.T) {
	headers := []string{"id", "name"}
	rows := [][]string{{"1", "Alice"}}

	plain, err := formatStructured(headers, rows, format.Plain)
	if err != nil || !strings.Contains(plain, "Alice") {
		t.Errorf("plain: %q err=%v", plain, err)
	}

	md, err := formatStructured(headers, rows, format.Markdown)
	if err != nil || !strings.Contains(md, "| id") {
		t.Errorf("markdown: %q err=%v", md, err)
	}

	js, err := formatStructured(headers, rows, format.JSON)
	if err != nil || !strings.Contains(js, `"name"`) {
		t.Errorf("json: %q err=%v", js, err)
	}
}

func TestMaskStructured(t *testing.T) {
	headers := []string{"id", "email"}
	rows := [][]string{{"1", "alice@example.com"}}
	conn := &config.Connection{Mask: &config.MaskConfig{Columns: []string{"email"}}}

	maskStructured(headers, rows, conn)

	if rows[0][0] != "1" {
		t.Errorf("id should be untouched, got %q", rows[0][0])
	}
	if rows[0][1] != "a***@example.com" {
		t.Errorf("email should be masked, got %q", rows[0][1])
	}
}

func TestMaskStructuredNoConfig(t *testing.T) {
	headers := []string{"email"}
	rows := [][]string{{"alice@example.com"}}
	conn := &config.Connection{} // no mask config

	maskStructured(headers, rows, conn)

	if rows[0][0] != "alice@example.com" {
		t.Errorf("no mask config should leave value unchanged, got %q", rows[0][0])
	}
}

func TestMCPListConnections(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg := &config.Config{Connections: []config.Connection{
		{
			Name: "prod",
			Env:  "production",
			DB:   config.DBConfig{Host: "db.example.com", Port: 3306, User: "app", Database: "myapp"},
			SSH:  &config.SSHConfig{Host: "bastion", User: "deploy"},
			Mask: &config.MaskConfig{Columns: []string{"email"}},
		},
		{
			Name:   "analytics",
			Env:    "production",
			Redash: &config.RedashConfig{URL: "https://redash.example.com", DataSourceID: 1, APIKey: "k"},
		},
	}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	out, err := mcpListConnections(nil)
	if err != nil {
		t.Fatalf("mcpListConnections: %v", err)
	}

	var infos []map[string]any
	if err := json.Unmarshal([]byte(out), &infos); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(infos))
	}
	if infos[0]["kind"] != "mysql" || infos[0]["ssh"] != "deploy@bastion" {
		t.Errorf("unexpected mysql conn info: %v", infos[0])
	}
	if infos[0]["masking_enabled"] != true {
		t.Errorf("expected masking_enabled true: %v", infos[0])
	}
	if infos[1]["kind"] != "redash" || infos[1]["redash"] != "https://redash.example.com" {
		t.Errorf("unexpected redash conn info: %v", infos[1])
	}
	// Secrets must never appear in the output.
	if strings.Contains(out, "\"k\"") || strings.Contains(strings.ToLower(out), "password") || strings.Contains(strings.ToLower(out), "api_key") {
		t.Errorf("connection list leaked a secret field:\n%s", out)
	}
}

func TestMCPListConnectionsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out, err := mcpListConnections(nil)
	if err != nil {
		t.Fatalf("mcpListConnections: %v", err)
	}
	if !strings.Contains(out, "No connections configured") {
		t.Errorf("expected empty message, got %q", out)
	}
}

func TestMCPQueryRequiresSQL(t *testing.T) {
	_, err := mcpQuery(map[string]any{})
	if err == nil {
		t.Error("expected error when sql argument is missing")
	}
}
