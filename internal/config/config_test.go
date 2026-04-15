package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := filepath.Join(tmpDir, ".config", "mysh")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	return tmpDir, func() {
		_ = os.Setenv("HOME", origHome)
	}
}

func TestLoadEmpty(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(cfg.Connections))
	}
}

func TestAddAndFind(t *testing.T) {
	cfg := &Config{}

	conn := Connection{
		Name: "test-db",
		DB: DBConfig{
			Host:     "localhost",
			Port:     3306,
			User:     "root",
			Database: "myapp",
		},
	}

	if err := cfg.Add(conn); err != nil {
		t.Fatalf("Add: %v", err)
	}

	found := cfg.Find("test-db")
	if found == nil {
		t.Fatal("Find returned nil")
	}
	if found.DB.Host != "localhost" {
		t.Errorf("host = %q, want localhost", found.DB.Host)
	}
}

func TestAddDuplicate(t *testing.T) {
	cfg := &Config{}

	conn := Connection{Name: "dup", DB: DBConfig{Host: "localhost"}}
	if err := cfg.Add(conn); err != nil {
		t.Fatalf("first Add: %v", err)
	}

	err := cfg.Add(conn)
	if err == nil {
		t.Error("expected error for duplicate name, got nil")
	}
}

func TestRemove(t *testing.T) {
	cfg := &Config{
		Connections: []Connection{
			{Name: "a", DB: DBConfig{Host: "a.example.com"}},
			{Name: "b", DB: DBConfig{Host: "b.example.com"}},
		},
	}

	if err := cfg.Remove("a"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if cfg.Find("a") != nil {
		t.Error("connection 'a' should have been removed")
	}
	if cfg.Find("b") == nil {
		t.Error("connection 'b' should still exist")
	}
}

func TestRemoveNotFound(t *testing.T) {
	cfg := &Config{}
	err := cfg.Remove("nonexistent")
	if err == nil {
		t.Error("expected error for removing nonexistent, got nil")
	}
}

func TestSaveAndLoad(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		Connections: []Connection{
			{
				Name: "prod",
				SSH: &SSHConfig{
					Host: "bastion.example.com",
					Port: 22,
					User: "deploy",
				},
				DB: DBConfig{
					Host:     "127.0.0.1",
					Port:     3306,
					User:     "app",
					Database: "production",
					Password: "encrypted-data-here",
				},
			},
			{
				Name: "local",
				DB: DBConfig{
					Host:     "localhost",
					Port:     3306,
					User:     "root",
					Database: "dev",
				},
			},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Connections) != 2 {
		t.Fatalf("expected 2 connections, got %d", len(loaded.Connections))
	}

	prod := loaded.Find("prod")
	if prod == nil {
		t.Fatal("prod not found")
	}
	if prod.SSH == nil {
		t.Fatal("prod SSH config is nil")
	}
	if prod.SSH.Host != "bastion.example.com" {
		t.Errorf("SSH host = %q, want bastion.example.com", prod.SSH.Host)
	}
	if prod.DB.Password != "encrypted-data-here" {
		t.Errorf("password = %q, want encrypted-data-here", prod.DB.Password)
	}

	local := loaded.Find("local")
	if local == nil {
		t.Fatal("local not found")
	}
	if local.SSH != nil {
		t.Error("local should not have SSH config")
	}
}

func TestFindNotFound(t *testing.T) {
	cfg := &Config{}
	if cfg.Find("nope") != nil {
		t.Error("expected nil for missing connection")
	}
}

func TestHasMaskConfig(t *testing.T) {
	tests := []struct {
		name string
		conn Connection
		want bool
	}{
		{"nil mask", Connection{}, false},
		{"empty mask", Connection{Mask: &MaskConfig{}}, false},
		{"with columns", Connection{Mask: &MaskConfig{Columns: []string{"email"}}}, true},
		{"with patterns", Connection{Mask: &MaskConfig{Patterns: []string{"*phone*"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conn.HasMaskConfig(); got != tt.want {
				t.Errorf("HasMaskConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEffectiveDriver(t *testing.T) {
	tests := []struct {
		name   string
		driver string
		want   string
	}{
		{"empty defaults to cli", "", DriverCLI},
		{"cli returns cli", DriverCLI, DriverCLI},
		{"native returns native", DriverNative, DriverNative},
		{"unknown falls back to cli", "invalid", DriverCLI},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &DBConfig{Driver: tt.driver}
			if got := db.EffectiveDriver(); got != tt.want {
				t.Errorf("EffectiveDriver() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveAndLoadWithDriver(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	cfg := &Config{
		Connections: []Connection{
			{
				Name: "legacy",
				DB: DBConfig{
					Host:   "10.0.0.5",
					Port:   3306,
					User:   "app",
					Driver: DriverNative,
				},
			},
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	legacy := loaded.Find("legacy")
	if legacy == nil {
		t.Fatal("legacy not found")
	}
	if legacy.DB.Driver != DriverNative {
		t.Errorf("driver = %q, want %q", legacy.DB.Driver, DriverNative)
	}
}

func TestShouldMask(t *testing.T) {
	tests := []struct {
		name   string
		conn   Connection
		isTTY  bool
		want   bool
	}{
		{
			name: "production non-TTY with mask rules",
			conn: Connection{
				Env:  "production",
				Mask: &MaskConfig{Columns: []string{"email"}},
			},
			isTTY: false,
			want:  true,
		},
		{
			name: "production TTY with mask rules",
			conn: Connection{
				Env:  "production",
				Mask: &MaskConfig{Columns: []string{"email"}},
			},
			isTTY: true,
			want:  true,
		},
		{
			name: "development non-TTY with mask rules",
			conn: Connection{
				Env:  "development",
				Mask: &MaskConfig{Columns: []string{"email"}},
			},
			isTTY: false,
			want:  false,
		},
		{
			name: "production non-TTY without mask rules",
			conn: Connection{
				Env: "production",
			},
			isTTY: false,
			want:  false,
		},
		{
			name: "staging non-TTY with mask rules",
			conn: Connection{
				Env:  "staging",
				Mask: &MaskConfig{Columns: []string{"email"}},
			},
			isTTY: false,
			want:  true,
		},
		{
			name: "staging TTY with mask rules",
			conn: Connection{
				Env:  "staging",
				Mask: &MaskConfig{Columns: []string{"email"}},
			},
			isTTY: true,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.conn.ShouldMask(tt.isTTY)
			if got != tt.want {
				t.Errorf("ShouldMask(%v) = %v, want %v", tt.isTTY, got, tt.want)
			}
		})
	}
}

func TestMaskConfigWarnings(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		wantSubs []string
		wantLen  int
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantLen: 0,
		},
		{
			name: "no mask",
			cfg: &Config{Connections: []Connection{
				{Name: "db1"},
			}},
			wantLen: 0,
		},
		{
			name: "clean config",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns:  []string{"email", "phone"},
					Patterns: []string{"*email*"},
				}},
			}},
			wantLen: 0,
		},
		{
			name: "internal whitespace in pattern",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Patterns: []string{"* name"},
				}},
			}},
			wantSubs: []string{`"db1"`, "patterns", `"* name"`, "whitespace"},
			wantLen:  1,
		},
		{
			name: "internal whitespace in column",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns: []string{"user email"},
				}},
			}},
			wantSubs: []string{`"db1"`, "columns", `"user email"`},
			wantLen:  1,
		},
		{
			name: "leading/trailing whitespace is tolerated at match but still reported",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns: []string{"  email"},
				}},
			}},
			wantLen: 0,
		},
		{
			name: "empty entry reported as ignored",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns: []string{""},
				}},
			}},
			wantSubs: []string{"empty entry", "ignored"},
			wantLen:  1,
		},
		{
			name: "whitespace-only entry reported as ignored",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Patterns: []string{"   "},
				}},
			}},
			wantSubs: []string{"whitespace-only entry", "ignored"},
			wantLen:  1,
		},
		{
			name: "trailing whitespace only is tolerated silently",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns: []string{"email  "},
				}},
			}},
			wantLen: 0,
		},
		{
			name: "zero-width space inside entry is detected",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Patterns: []string{"email\u200bname"},
				}},
			}},
			wantSubs: []string{"patterns", "whitespace"},
			wantLen:  1,
		},
		{
			name: "full-width space inside entry is detected",
			cfg: &Config{Connections: []Connection{
				{Name: "db1", Mask: &MaskConfig{
					Columns: []string{"user\u3000email"},
				}},
			}},
			wantSubs: []string{"columns", "whitespace"},
			wantLen:  1,
		},
		{
			name: "multiple connections with mixed issues",
			cfg: &Config{Connections: []Connection{
				{Name: "conn-a", Mask: &MaskConfig{Columns: []string{"user email"}}},
				{Name: "conn-b", Mask: &MaskConfig{Patterns: []string{"* name", "   "}}},
				{Name: "conn-c"},
			}},
			wantSubs: []string{"conn-a", "conn-b", "whitespace-only entry"},
			wantLen:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaskConfigWarnings(tt.cfg)
			if len(got) != tt.wantLen {
				t.Fatalf("warnings count = %d, want %d; warnings=%v", len(got), tt.wantLen, got)
			}
			if tt.wantLen == 0 {
				return
			}
			joined := strings.Join(got, "\n")
			for _, sub := range tt.wantSubs {
				if !strings.Contains(joined, sub) {
					t.Errorf("warnings %q missing substring %q", joined, sub)
				}
			}
		})
	}
}

// TestLoadEmitsMaskWarnings verifies the Load -> stderr wiring, not just
// MaskConfigWarnings in isolation. It captures the warnings writer and
// writes a real config file to disk.
func TestLoadEmitsMaskWarnings(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	origWriter := maskWarningsWriter
	var buf bytes.Buffer
	maskWarningsWriter = &buf
	t.Cleanup(func() { maskWarningsWriter = origWriter })

	cfg := &Config{Connections: []Connection{{
		Name: "prod-db",
		DB:   DBConfig{Host: "db.example", User: "u", Database: "d"},
		Mask: &MaskConfig{Patterns: []string{"* name"}},
	}}}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "prod-db") || !strings.Contains(out, "* name") {
		t.Errorf("expected warning about %q to be written to stderr writer; got %q", "* name", out)
	}
}

