package cmd

import (
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/importer"
)

func TestNormalizeEnv(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"production", "production"},
		{"prod", "production"},
		{"staging", "staging"},
		{"stg", "staging"},
		{"development", "development"},
		{"dev", "development"},
		{"PRODUCTION", "production"},
		{"Prod", "production"},
		{"  staging  ", "staging"},
		{"invalid", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeEnv(tt.input)
			if got != tt.want {
				t.Errorf("normalizeEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseMaskInput(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantNil      bool
		wantCols     int
		wantPatterns int
	}{
		{"empty returns nil", "", true, 0, 0},
		{"single column", "email", false, 1, 0},
		{"single pattern", "*password*", false, 0, 1},
		{"mixed", "email,phone,*secret*,*token*", false, 2, 2},
		{"trailing comma", "email,", false, 1, 0},
		{"whitespace entries", " , , ", true, 0, 0},
		{"whitespace around values", " email , *phone* ", false, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseMaskInput(tt.input)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			if len(got.Columns) != tt.wantCols {
				t.Errorf("columns: got %d, want %d", len(got.Columns), tt.wantCols)
			}
			if len(got.Patterns) != tt.wantPatterns {
				t.Errorf("patterns: got %d, want %d", len(got.Patterns), tt.wantPatterns)
			}
		})
	}
}

func TestParseImportFlagsDBCredentialOverrides(t *testing.T) {
	opts, err := parseImportFlags([]string{"--from", "yaml", "--file", "team.yaml", "--all", "--db-user", "alice", "--reuse-password"})
	if err != nil {
		t.Fatalf("parseImportFlags: %v", err)
	}
	if opts.from != "yaml" || opts.file != "team.yaml" || !opts.all {
		t.Fatalf("basic flags parsed incorrectly: %+v", opts)
	}
	if opts.dbUser != "alice" {
		t.Errorf("dbUser = %q, want %q", opts.dbUser, "alice")
	}
	if !opts.reusePassword {
		t.Error("reusePassword should be true")
	}
}

func TestParseImportFlagsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		check   func(t *testing.T, opts importOptions)
	}{
		{
			name:    "ask-user and db-user conflict",
			args:    []string{"--from", "yaml", "--ask-user", "--db-user", "alice"},
			wantErr: true,
		},
		{
			name:    "db-user without value",
			args:    []string{"--from", "yaml", "--db-user"},
			wantErr: true,
		},
		{
			name:    "db-user with whitespace-only value",
			args:    []string{"--from", "yaml", "--db-user", "   "},
			wantErr: true,
		},
		{
			name:    "ask-user alone is valid",
			args:    []string{"--from", "yaml", "--ask-user"},
			wantErr: false,
			check: func(t *testing.T, opts importOptions) {
				if !opts.askUser {
					t.Error("askUser should be true")
				}
				if opts.dbUser != "" {
					t.Errorf("dbUser = %q, want empty", opts.dbUser)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseImportFlags(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for args %v", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseImportFlags(%v): %v", tt.args, err)
			}
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

func TestResolveImportDBUser(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current string
		opts    importOptions
		want    string
	}{
		{"db-user override wins over ask-user", "ignored\n", "readonly", importOptions{dbUser: "alice", askUser: true}, "alice"},
		{"db-user alone (no ask-user)", "", "readonly", importOptions{dbUser: "alice"}, "alice"},
		{"no flags returns current unchanged", "", "readonly", importOptions{}, "readonly"},
		{"ask-user uses entered value", "bob\n", "readonly", importOptions{askUser: true}, "bob"},
		{"ask-user empty input falls back to current", "\n", "readonly", importOptions{askUser: true}, "readonly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveImportDBUser(strings.NewReader(tt.input), "prod", tt.current, tt.opts)
			if got != tt.want {
				t.Errorf("resolveImportDBUser = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasDBConnections(t *testing.T) {
	redash := &importer.ImportedConnection{Redash: &config.RedashConfig{URL: "https://redash.example.com"}}
	redashEmptyURL := &importer.ImportedConnection{Redash: &config.RedashConfig{}}
	db := &importer.ImportedConnection{Name: "prod"}

	tests := []struct {
		name  string
		conns []importer.ImportedConnection
		want  bool
	}{
		{"empty slice", nil, false},
		{"db only", []importer.ImportedConnection{*db}, true},
		{"redash only", []importer.ImportedConnection{*redash}, false},
		{"redash with empty url counts as db", []importer.ImportedConnection{*redashEmptyURL}, true},
		{"redash and db mixed", []importer.ImportedConnection{*redash, *db}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasDBConnections(tt.conns); got != tt.want {
				t.Errorf("hasDBConnections = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindConnectionEmpty(t *testing.T) {
	// When no config exists, findConnection should return a descriptive error
	// We can't easily test this without mocking config.Load, but we can
	// test the Config methods directly

	cfg := &config.Config{}
	if cfg.Find("nonexistent") != nil {
		t.Error("expected nil for nonexistent connection")
	}
}

func TestMysqlArgs(t *testing.T) {
	rc := &resolvedConn{
		host:     "127.0.0.1",
		port:     3306,
		user:     "root",
		database: "testdb",
		driver:   "cli",
	}

	args := rc.mysqlArgs()

	expected := []string{"-h", "127.0.0.1", "-P", "3306", "-u", "root", "testdb"}
	if len(args) != len(expected) {
		t.Fatalf("args length: got %d, want %d", len(args), len(expected))
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("args[%d]: got %q, want %q", i, a, expected[i])
		}
	}
}

func TestMysqlArgsNoDatabase(t *testing.T) {
	rc := &resolvedConn{
		host:   "127.0.0.1",
		port:   3306,
		user:   "root",
		driver: "cli",
	}

	args := rc.mysqlArgs()
	for _, a := range args {
		if a == "" {
			t.Error("args should not contain empty string when database is empty")
		}
	}
	if len(args) != 6 {
		t.Errorf("args length without database: got %d, want 6", len(args))
	}
}

func TestIsNative(t *testing.T) {
	cli := &resolvedConn{driver: config.DriverCLI}
	native := &resolvedConn{driver: config.DriverNative}

	if cli.isNative() {
		t.Error("CLI driver should not be native")
	}
	if !native.isNative() {
		t.Error("native driver should be native")
	}
}
