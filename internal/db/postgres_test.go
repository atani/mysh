package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// hostileEnv sets every libpq environment variable pgx reads, plus a HOME with
// ~/.pgpass, a service file and a root certificate, all pointing somewhere
// other than the profile under test.
func hostileEnv(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	pgpass := filepath.Join(home, ".pgpass")
	if err := os.WriteFile(pgpass, []byte("*:*:*:*:pgpass-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".postgresql"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".postgresql", "root.crt"), []byte("not a cert"), 0600); err != nil {
		t.Fatal(err)
	}
	serviceFile := filepath.Join(home, ".pg_service.conf")
	service := "[evil]\nhost=evil.example.com\nport=6666\nuser=evil\npassword=evil\ndbname=evil\nsslmode=disable\noptions=-c search_path=evil\n"
	if err := os.WriteFile(serviceFile, []byte(service), 0600); err != nil {
		t.Fatal(err)
	}

	env := map[string]string{
		"PGHOST":               "env.example.com",
		"PGPORT":               "7777",
		"PGDATABASE":           "envdb",
		"PGUSER":               "envuser",
		"PGPASSWORD":           "env-password",
		"PGPASSFILE":           pgpass,
		"PGAPPNAME":            "env-app",
		"PGCONNECT_TIMEOUT":    "1",
		"PGSSLMODE":            "disable",
		"PGSSLKEY":             filepath.Join(home, "missing.key"),
		"PGSSLCERT":            filepath.Join(home, "missing.crt"),
		"PGSSLCERTMODE":        "require",
		"PGSSLSNI":             "0",
		"PGSSLROOTCERT":        filepath.Join(home, ".postgresql", "root.crt"),
		"PGSSLPASSWORD":        "env-ssl-password",
		"PGSSLNEGOTIATION":     "direct",
		"PGTARGETSESSIONATTRS": "read-only",
		"PGSERVICE":            "evil",
		"PGSERVICEFILE":        serviceFile,
		"PGTZ":                 "Asia/Tokyo",
		"PGOPTIONS":            "-c search_path=env",
		"PGMINPROTOCOLVERSION": "3.2",
		"PGMAXPROTOCOLVERSION": "3.2",
		"PGCHANNELBINDING":     "require",
		"PGREQUIREAUTH":        "md5",
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestPostgresConfigIgnoresLibpqEnvironment(t *testing.T) {
	hostileEnv(t)

	for _, tc := range []struct {
		name     string
		password string
	}{
		{"with password", "pro'file\\pass word"},
		{"empty password", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := postgresConfig(PostgresParams{
				Host: "db.example.com", Port: 5432, User: "app", Password: tc.password,
				Database: "appdb", SSLMode: "require",
			})
			if err != nil {
				t.Fatalf("postgresConfig: %v", err)
			}
			if cfg.Host != "db.example.com" || cfg.Port != 5432 {
				t.Errorf("target = %s:%d, want db.example.com:5432", cfg.Host, cfg.Port)
			}
			if cfg.User != "app" || cfg.Database != "appdb" {
				t.Errorf("user/database = %q/%q", cfg.User, cfg.Database)
			}
			if cfg.Password != tc.password {
				t.Errorf("password = %q, want %q (PGPASSWORD/.pgpass must not apply)", cfg.Password, tc.password)
			}
			// sslmode=require: TLS without verification, no plaintext fallback,
			// no client certificate and no root CA from the environment.
			if cfg.TLSConfig == nil || !cfg.TLSConfig.InsecureSkipVerify || cfg.TLSConfig.RootCAs != nil {
				t.Errorf("unexpected TLS config: %+v", cfg.TLSConfig)
			}
			if cfg.TLSConfig.ServerName != "db.example.com" {
				t.Errorf("SNI = %q, want db.example.com (PGSSLSNI must not apply)", cfg.TLSConfig.ServerName)
			}
			if len(cfg.Fallbacks) != 0 {
				t.Errorf("require must not have fallbacks, got %d", len(cfg.Fallbacks))
			}
			if cfg.SSLNegotiation != "postgres" {
				t.Errorf("SSLNegotiation = %q, want postgres", cfg.SSLNegotiation)
			}
			if cfg.ValidateConnect != nil {
				t.Error("target_session_attrs from PGTARGETSESSIONATTRS must not apply")
			}
			if len(cfg.RuntimeParams) != 1 || cfg.RuntimeParams["application_name"] != "mysh" {
				t.Errorf("RuntimeParams = %v, want only application_name=mysh", cfg.RuntimeParams)
			}
			if cfg.ConnectTimeout != postgresConnectTimeout*time.Second {
				t.Errorf("ConnectTimeout = %v", cfg.ConnectTimeout)
			}
			if cfg.MinProtocolVersion != "3.0" || cfg.MaxProtocolVersion != "3.0" {
				t.Errorf("protocol versions = %s/%s", cfg.MinProtocolVersion, cfg.MaxProtocolVersion)
			}
			if cfg.ChannelBinding != "prefer" || cfg.RequireAuth != "" {
				t.Errorf("channel_binding/require_auth = %q/%q", cfg.ChannelBinding, cfg.RequireAuth)
			}
			if cfg.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
				t.Errorf("DefaultQueryExecMode = %v, want simple protocol", cfg.DefaultQueryExecMode)
			}
		})
	}
}

// PGSERVICE cannot be neutralized from the connection string. When it names a
// service that does not exist, pgx refuses to build a config; mysh fails closed
// with an explicit error instead of connecting somewhere else.
func TestPostgresConfigMissingServiceFailsClosed(t *testing.T) {
	hostileEnv(t)
	t.Setenv("PGSERVICE", "does-not-exist")
	_, err := postgresConfig(PostgresParams{Host: "db.example.com", Port: 5432, User: "u", Password: "top-secret", Database: "d"})
	if err == nil {
		t.Fatal("expected an error for a missing PGSERVICE entry")
	}
	if !strings.Contains(err.Error(), "service") || strings.Contains(err.Error(), "top-secret") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPostgresConfigSSLModes(t *testing.T) {
	hostileEnv(t)
	base := PostgresParams{Host: "db.example.com", Port: 5432, User: "u", Database: "d"}

	t.Run("disable", func(t *testing.T) {
		p := base
		p.SSLMode = "disable"
		cfg, err := postgresConfig(p)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.TLSConfig != nil || len(cfg.Fallbacks) != 0 {
			t.Errorf("disable: TLS=%v fallbacks=%d", cfg.TLSConfig, len(cfg.Fallbacks))
		}
	})

	t.Run("prefer falls back to plaintext on the same target", func(t *testing.T) {
		p := base
		p.SSLMode = "" // default
		cfg, err := postgresConfig(p)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.TLSConfig == nil {
			t.Fatal("prefer must try TLS first")
		}
		if len(cfg.Fallbacks) != 1 || cfg.Fallbacks[0].TLSConfig != nil {
			t.Fatalf("prefer must have one plaintext fallback, got %+v", cfg.Fallbacks)
		}
		if cfg.Fallbacks[0].Host != "db.example.com" || cfg.Fallbacks[0].Port != 5432 {
			t.Errorf("fallback target = %s:%d", cfg.Fallbacks[0].Host, cfg.Fallbacks[0].Port)
		}
	})

	t.Run("verify-full checks the host name", func(t *testing.T) {
		p := base
		p.SSLMode = "verify-full"
		cfg, err := postgresConfig(p)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify || cfg.TLSConfig.ServerName != "db.example.com" {
			t.Errorf("verify-full TLS config: %+v", cfg.TLSConfig)
		}
	})

	t.Run("verify-full through a tunnel verifies the original host", func(t *testing.T) {
		p := base
		p.Host = "127.0.0.1"
		p.Port = 40000
		p.SSLMode = "verify-full"
		p.TLSServerName = "db.example.com"
		cfg, err := postgresConfig(p)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Host != "127.0.0.1" || cfg.Port != 40000 {
			t.Errorf("dial target = %s:%d, want the tunnel endpoint", cfg.Host, cfg.Port)
		}
		if cfg.TLSConfig == nil || cfg.TLSConfig.InsecureSkipVerify || cfg.TLSConfig.ServerName != "db.example.com" {
			t.Errorf("TLS must verify db.example.com, got %+v", cfg.TLSConfig)
		}
	})

	t.Run("invalid sslmode", func(t *testing.T) {
		p := base
		p.SSLMode = "bogus"
		p.Password = "top-secret"
		_, err := postgresConfig(p)
		if err == nil {
			t.Fatal("expected error")
		}
		if strings.Contains(err.Error(), "top-secret") {
			t.Errorf("error leaks the password: %v", err)
		}
	})
}

func TestQuoteConnValue(t *testing.T) {
	tests := map[string]string{
		"":          "''",
		"plain":     "'plain'",
		"it's":      `'it\'s'`,
		`back\nope`: `'back\\nope'`,
		"a b=c":     "'a b=c'",
	}
	for in, want := range tests {
		if got := quoteConnValue(in); got != want {
			t.Errorf("quoteConnValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	jst := time.FixedZone("JST", 9*3600)
	ts := time.Date(2024, 1, 2, 3, 4, 5, 123000000, time.UTC)
	tests := []struct {
		name     string
		val      interface{}
		typeName string
		want     string
	}{
		{"nil", nil, "TEXT", "NULL"},
		{"mysql bytes", []byte("hello"), "VARCHAR", "hello"},
		{"mysql blob stays raw", []byte{0x41, 0x42}, "BLOB", "AB"},
		{"bytea as hex", []byte{0xde, 0xad, 0xbe, 0xef}, "BYTEA", `\xdeadbeef`},
		{"jsonb bytes", []byte(`{"a": 1}`), "JSONB", `{"a": 1}`},
		{"date", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), "DATE", "2024-01-02"},
		{"timestamp", ts, "TIMESTAMP", "2024-01-02 03:04:05.123"},
		{"timestamptz", ts.In(jst), "TIMESTAMPTZ", "2024-01-02 12:04:05.123+09:00"},
		{"int64", int64(42), "INT8", "42"},
		{"bool", true, "BOOL", "true"},
		{"string", "{1,2}", "_INT4", "{1,2}"},
	}
	for _, tt := range tests {
		if got := formatValue(tt.val, tt.typeName); got != tt.want {
			t.Errorf("%s: formatValue = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestSplitStatementsPostgres(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"two statements", "SELECT 1; SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"backslash is literal in standard strings", `SELECT '\'; SELECT 2`, []string{`SELECT '\'`, "SELECT 2"}},
		{"doubled quote", "SELECT 'it''s;'; SELECT 2", []string{"SELECT 'it''s;'", "SELECT 2"}},
		{"E string escapes", `SELECT E'\';'; SELECT 2`, []string{`SELECT E'\';'`, "SELECT 2"}},
		{"lowercase e string", `SELECT e'a\'b;c'`, []string{`SELECT e'a\'b;c'`}},
		{"identifier ending in e is not an E string", `SELECT name'\'; SELECT 2`, []string{`SELECT name'\'`, "SELECT 2"}},
		{"double-quoted identifier", `SELECT "a;b" FROM t; SELECT 2`, []string{`SELECT "a;b" FROM t`, "SELECT 2"}},
		{"backslash in identifier", `SELECT "a\"; SELECT 2`, []string{`SELECT "a\"`, "SELECT 2"}},
		{"backtick is not a quote", "SELECT `a; SELECT 2`", []string{"SELECT `a", "SELECT 2`"}},
		{
			"dollar quoted function body",
			"CREATE FUNCTION f() RETURNS int AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql; SELECT f()",
			[]string{"CREATE FUNCTION f() RETURNS int AS $$ BEGIN RETURN 1; END; $$ LANGUAGE plpgsql", "SELECT f()"},
		},
		{
			"tagged dollar quote with nested tag",
			"DO $outer$ BEGIN EXECUTE $inner$SELECT 1;$inner$; END $outer$; SELECT 2",
			[]string{"DO $outer$ BEGIN EXECUTE $inner$SELECT 1;$inner$; END $outer$", "SELECT 2"},
		},
		{"positional parameter is not a dollar quote", "PREPARE p AS SELECT $1; SELECT 2", []string{"PREPARE p AS SELECT $1", "SELECT 2"}},
		{"dollar inside identifier", "SELECT a$b$c FROM t; SELECT 2", []string{"SELECT a$b$c FROM t", "SELECT 2"}},
		{"line comment", "SELECT 1 -- a;b\n; SELECT 2", []string{"SELECT 1 -- a;b", "SELECT 2"}},
		{"nested block comment", "SELECT /* a /* b; */ c; */ 1; SELECT 2", []string{"SELECT /* a /* b; */ c; */ 1", "SELECT 2"}},
		{"unterminated dollar quote keeps the rest", "SELECT $$a;b", []string{"SELECT $$a;b"}},
		{"multibyte text", "SELECT 'あ;い'; SELECT 'う'", []string{"SELECT 'あ;い'", "SELECT 'う'"}},
		{"empty statements skipped", " ; SELECT 1;; ", []string{"SELECT 1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitStatements(tt.input, DialectPostgres)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d statements %q, want %d %q", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("stmt[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// The MySQL dialect keeps treating backslash as an escape, unlike PostgreSQL.
func TestSplitStatementsDialectsDiffer(t *testing.T) {
	input := `SELECT 'a\'; SELECT 2'`
	if got := SplitStatements(input, DialectMySQL); len(got) != 1 {
		t.Errorf("MySQL: got %q, want one statement", got)
	}
	if got := SplitStatements(input, DialectPostgres); len(got) != 2 {
		t.Errorf("Postgres: got %q, want two statements", got)
	}
}
