package config

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDBConfigTypeDefaults(t *testing.T) {
	tests := []struct {
		name       string
		db         DBConfig
		wantDriver string
		wantPort   int
		wantPg     bool
	}{
		{"legacy empty type is MySQL cli", DBConfig{}, DriverCLI, 3306, false},
		{"explicit mysql native", DBConfig{Type: "mysql", Driver: DriverNative}, DriverNative, 3306, false},
		{"postgres always native", DBConfig{Type: DBTypePostgres}, DriverNative, 5432, true},
		{"postgres keeps explicit port", DBConfig{Type: DBTypePostgres, Port: 6543}, DriverNative, 6543, true},
		{"mysql keeps explicit port", DBConfig{Port: 3307}, DriverCLI, 3307, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.db.EffectiveDriver(); got != tt.wantDriver {
				t.Errorf("EffectiveDriver = %q, want %q", got, tt.wantDriver)
			}
			if got := tt.db.EffectivePort(); got != tt.wantPort {
				t.Errorf("EffectivePort = %d, want %d", got, tt.wantPort)
			}
			if got := tt.db.IsPostgres(); got != tt.wantPg {
				t.Errorf("IsPostgres = %v, want %v", got, tt.wantPg)
			}
		})
	}
}

func TestDBConfigEffectiveSSLMode(t *testing.T) {
	if got := (&DBConfig{Type: DBTypePostgres}).EffectiveSSLMode(); got != "prefer" {
		t.Errorf("default sslmode = %q, want prefer", got)
	}
	if got := (&DBConfig{Type: DBTypePostgres, SSLMode: "require"}).EffectiveSSLMode(); got != "require" {
		t.Errorf("sslmode = %q, want require", got)
	}
}

func TestDBConfigValidate(t *testing.T) {
	valid := []DBConfig{
		{},
		{Type: "mysql", Driver: DriverCLI},
		{Type: DBTypePostgres},
		{Type: DBTypePostgres, Driver: DriverNative, SSLMode: "verify-full"},
		{Type: DBTypePostgres, Host: "db.example.com"},
		{Type: DBTypePostgres, Host: "2001:db8::1"},
	}
	for _, d := range valid {
		if err := d.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", d, err)
		}
	}
	invalid := []struct {
		db   DBConfig
		want string
	}{
		{DBConfig{Type: "supabase"}, "unknown database type"},
		{DBConfig{Type: "postgresql"}, "unknown database type"},
		{DBConfig{Type: DBTypePostgres, Driver: DriverCLI}, "not supported for PostgreSQL"},
		{DBConfig{Type: DBTypePostgres, SSLMode: "required"}, "invalid sslmode"},
		{DBConfig{SSLMode: "require"}, "only supported for PostgreSQL"},
		// pgx treats a comma-separated host as a fallback list and a path as a
		// Unix socket (no TLS), which would not match the configured sslmode.
		{DBConfig{Type: DBTypePostgres, Host: "a.example.com,b.example.com"}, "multiple hosts"},
		{DBConfig{Type: DBTypePostgres, Host: "/var/run/postgresql", SSLMode: "require"}, "Unix socket"},
		{DBConfig{Type: DBTypePostgres, Host: `C:\pg`}, "Unix socket"},
	}
	for _, tt := range invalid {
		err := tt.db.Validate()
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Validate(%+v) = %v, want error containing %q", tt.db, err, tt.want)
		}
	}
}

func TestTypeLabelAndDefaultPort(t *testing.T) {
	if TypeLabel("") != "MySQL" || TypeLabel(DBTypeMySQL) != "MySQL" || TypeLabel(DBTypePostgres) != "PostgreSQL" {
		t.Error("unexpected type labels")
	}
	if DefaultPort("") != 3306 || DefaultPort(DBTypePostgres) != 5432 {
		t.Error("unexpected default ports")
	}
}

// A config written before PostgreSQL support must load unchanged and be saved
// back byte-for-byte: the new fields are omitempty.
func TestLegacyYAMLRoundTripUnchanged(t *testing.T) {
	legacy := `version: 1
connections:
    - name: prod
      env: production
      ssh:
        host: bastion
        port: 22
        user: deploy
      db:
        host: 127.0.0.1
        port: 3306
        user: app
        database: main
        password: encrypted-blob
        driver: native
      mask:
        columns:
            - email
`
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c := cfg.Find("prod")
	if c == nil || c.DB.Type != "" || c.DB.SSLMode != "" || c.DB.IsPostgres() {
		t.Fatalf("legacy connection should load as MySQL: %+v", c)
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != legacy {
		t.Errorf("re-saved YAML differs from original.\n--- got ---\n%s\n--- want ---\n%s", got, legacy)
	}
}

func TestPostgresYAMLRoundTrip(t *testing.T) {
	in := Config{Version: 1, Connections: []Connection{{
		Name: "sb",
		DB:   DBConfig{Host: "aws-0-ap-northeast-1.pooler.supabase.com", Port: 6543, User: "postgres.abc", Database: "postgres", Type: DBTypePostgres, SSLMode: "require"},
	}}}
	data, err := yaml.Marshal(&in)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "type: postgres") || !strings.Contains(s, "sslmode: require") {
		t.Errorf("type/sslmode not serialized:\n%s", s)
	}
	var out Config
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out.Connections[0].DB != in.Connections[0].DB {
		t.Errorf("round trip mismatch: %+v vs %+v", out.Connections[0].DB, in.Connections[0].DB)
	}
}
