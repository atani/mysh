package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/tunnel"
)

func TestWriteConnectionList(t *testing.T) {
	cfg := &config.Config{Connections: []config.Connection{
		{
			Name: "prod-db",
			Env:  "production",
			DB:   config.DBConfig{Host: "db.prod", Port: 3306, User: "app", Database: "main"},
			SSH:  &config.SSHConfig{User: "deploy", Host: "bastion"},
		},
		{
			Name:   "analytics",
			Env:    "staging",
			Redash: &config.RedashConfig{URL: "https://redash.example.com", DataSourceID: 7},
		},
		{
			Name: "legacy",
			Env:  "weird-env", // falls into "Other" bucket
			DB:   config.DBConfig{Host: "old", Port: 3307, User: "u", Database: "d"},
		},
	}}

	var buf bytes.Buffer
	if err := writeConnectionList(&buf, cfg); err != nil {
		t.Fatalf("writeConnectionList: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		"[Production]", "prod-db", "deploy@bastion",
		"[Staging]", "(Redash #7)", "https://redash.example.com",
		"[Other]", "legacy",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// Production should appear before Staging (environment ordering).
	if strings.Index(out, "[Production]") > strings.Index(out, "[Staging]") {
		t.Error("environments are not in declared order")
	}
}

func TestWriteTunnelList(t *testing.T) {
	tunnels := []*tunnel.TunnelInfo{
		{Name: "prod", PID: 4321, LocalPort: 13306, RemoteHost: "db.internal", RemotePort: 3306},
	}
	var buf bytes.Buffer
	if err := writeTunnelList(&buf, tunnels); err != nil {
		t.Fatalf("writeTunnelList: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"NAME", "prod", "4321", "13306", "db.internal:3306"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestBuildExported(t *testing.T) {
	conns := []config.Connection{
		{
			Name: "db1",
			Env:  "production",
			DB:   config.DBConfig{Host: "h", Port: 3306, User: "u", Database: "d", Password: "ENCRYPTED-SECRET", Driver: config.DriverNative},
			SSH:  &config.SSHConfig{User: "deploy", Host: "bastion"},
			Mask: &config.MaskConfig{Columns: []string{"email"}},
		},
		{
			Name:   "redash1",
			Redash: &config.RedashConfig{URL: "https://r", APIKey: "ENCRYPTED-KEY", DataSourceID: 3},
		},
	}

	exported := buildExported(conns)
	if len(exported) != 2 {
		t.Fatalf("expected 2 exported, got %d", len(exported))
	}

	// DB connection: password must be stripped, fields preserved.
	db := exported[0]
	if db.DB == nil {
		t.Fatal("db connection should have DB set")
	}
	if db.DB.Driver != config.DriverNative || db.DB.Host != "h" {
		t.Errorf("db fields wrong: %+v", db.DB)
	}
	if db.SSH == nil || db.SSH.User != "deploy" {
		t.Error("ssh config not preserved")
	}
	if db.Mask == nil {
		t.Error("mask config not preserved")
	}

	// Redash connection: API key must not be present in the exported struct.
	r := exported[1]
	if r.Redash == nil || r.Redash.URL != "https://r" || r.Redash.DataSourceID != 3 {
		t.Errorf("redash fields wrong: %+v", r.Redash)
	}
	if r.DB != nil {
		t.Error("redash connection should not have DB set")
	}
}

func TestResolveRedashClientPlaintext(t *testing.T) {
	// Plaintext key fallback path (no master password needed).
	conn := &config.Connection{Redash: &config.RedashConfig{URL: "https://r", APIKey: "plain-key"}}
	client, err := resolveRedashClient(conn)
	if err != nil {
		t.Fatalf("resolveRedashClient: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
}

func TestResolveRedashClientMissingKey(t *testing.T) {
	conn := &config.Connection{Redash: &config.RedashConfig{URL: "https://r", APIKey: ""}}
	if _, err := resolveRedashClient(conn); err == nil {
		t.Error("expected error for missing API key")
	}
}
