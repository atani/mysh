package cmd

import (
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/tunnel"
	"gopkg.in/yaml.v3"
)

// pgConn returns a PostgreSQL connection profile. Driver is left empty so the
// tests prove that the database type alone routes to the native driver.
func pgConn(name string) config.Connection {
	return config.Connection{
		Name: name,
		Env:  "development",
		DB:   config.DBConfig{Host: "127.0.0.1", Port: 5432, User: "postgres", Database: "app", Type: config.DBTypePostgres},
	}
}

// withPostgresMockDB installs a sqlmock database as the native opener and
// fails the test if the opener is reached for anything but a PostgreSQL
// connection.
func withPostgresMockDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	orig := dbOpen
	dbOpen = func(rc *resolvedConn) (*sql.DB, error) {
		if !rc.isPostgres() || !rc.isNative() {
			t.Errorf("dbOpen reached with dbType=%q driver=%q, want native postgres", rc.dbType, rc.driver)
		}
		return mockDB, nil
	}
	t.Cleanup(func() {
		dbOpen = orig
		_ = mockDB.Close()
	})
	return mock
}

func TestParsePostgresURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgConnString
	}{
		{
			"supabase direct with placeholder",
			"postgresql://postgres:[YOUR-PASSWORD]@db.abcdefghij.supabase.co:5432/postgres",
			pgConnString{host: "db.abcdefghij.supabase.co", port: 5432, user: "postgres", database: "postgres"},
		},
		{
			"supabase session pooler with dotted user",
			"postgresql://postgres.abcdefghij:[YOUR-PASSWORD]@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres",
			pgConnString{host: "aws-0-ap-northeast-1.pooler.supabase.com", port: 5432, user: "postgres.abcdefghij", database: "postgres"},
		},
		{
			"supabase transaction pooler",
			"postgresql://postgres.abcdefghij:[YOUR-PASSWORD]@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres",
			pgConnString{host: "aws-0-ap-northeast-1.pooler.supabase.com", port: 6543, user: "postgres.abcdefghij", database: "postgres"},
		},
		{
			"percent-encoded password",
			"postgresql://postgres.ref:p%40ss%3Aw%2Ford%25@pooler.example.com:6543/postgres",
			pgConnString{host: "pooler.example.com", port: 6543, user: "postgres.ref", password: "p@ss:w/ord%", hasPassword: true, database: "postgres"},
		},
		{
			"unencoded special characters in password",
			"postgres://app:p@ss/w?rd:x%@db.example.com/appdb?sslmode=verify-full",
			pgConnString{host: "db.example.com", port: 5432, user: "app", password: "p@ss/w?rd:x%", hasPassword: true, database: "appdb", sslmode: "verify-full"},
		},
		{
			"default port and query parameters",
			"postgresql://u:pw@db.example.com/mydb?sslmode=require&application_name=x&pgbouncer=true",
			pgConnString{host: "db.example.com", port: 5432, user: "u", password: "pw", hasPassword: true, database: "mydb", sslmode: "require", ignored: []string{"application_name", "pgbouncer"}},
		},
		{
			"ipv6 host",
			"postgresql://u@[2001:db8::1]:5433/db",
			pgConnString{host: "2001:db8::1", port: 5433, user: "u", database: "db"},
		},
		{
			"no user info",
			"postgresql://db.example.com",
			pgConnString{host: "db.example.com", port: 5432},
		},
		{
			"empty password",
			"postgresql://u:@db.example.com/db",
			pgConnString{host: "db.example.com", port: 5432, user: "u", database: "db"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePostgresURL(tt.in)
			if err != nil {
				t.Fatalf("parsePostgresURL: %v", err)
			}
			if got.host != tt.want.host || got.port != tt.want.port || got.user != tt.want.user ||
				got.password != tt.want.password || got.hasPassword != tt.want.hasPassword ||
				got.database != tt.want.database || got.sslmode != tt.want.sslmode {
				t.Errorf("got %+v, want %+v", *got, tt.want)
			}
			if strings.Join(sortedCopy(got.ignored), ",") != strings.Join(tt.want.ignored, ",") {
				t.Errorf("ignored = %v, want %v", got.ignored, tt.want.ignored)
			}
		})
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func TestParsePostgresURLErrorsDoNotLeakPassword(t *testing.T) {
	inputs := []string{
		"mysql://u:SECRETPW@h/db",
		"postgresql://u:SECRETPW@h:notaport/db",
		"postgresql://u:SECRETPW@h1,h2/db",
		"postgresql://u:SECRETPW@/db",
		"postgresql://u:SECRETPW@h/db?sslmode=bogus",
		"postgresql://u:SECRETPW@[::1/db",
		"postgresql://u@h/db?password=SECRETPW",
	}
	for _, in := range inputs {
		_, err := parsePostgresURL(in)
		if err == nil {
			t.Errorf("parsePostgresURL(%q): expected error", in)
			continue
		}
		if strings.Contains(err.Error(), "SECRETPW") {
			t.Errorf("error leaks password: %v", err)
		}
	}
}

func TestParseAddFlagsPostgres(t *testing.T) {
	f, err := parseAddFlags([]string{"--type", "supabase", "--sslmode", "verify-full", "--url", "postgresql://u@h/db"})
	if err != nil {
		t.Fatalf("parseAddFlags: %v", err)
	}
	if f.dbType != "supabase" || f.sslmode != "verify-full" || f.url != "postgresql://u@h/db" {
		t.Errorf("flags = %+v", f)
	}

	f, err = parseAddFlags([]string{"--url", "postgresql://u@h/db"})
	if err != nil {
		t.Fatalf("parseAddFlags --url: %v", err)
	}
	if f.dbType != addKindPostgres {
		t.Errorf("--url alone should imply postgres, got %q", f.dbType)
	}

	f, err = parseAddFlags([]string{"--type", "postgresql"})
	if err != nil || f.dbType != addKindPostgres {
		t.Errorf("--type postgresql: %v %q", err, f.dbType)
	}

	errCases := []struct {
		args []string
		want string
	}{
		{[]string{"--type", "oracle"}, "invalid --type"},
		{[]string{"--type"}, "requires a value"},
		{[]string{"--sslmode"}, "requires a value"},
		{[]string{"--url"}, "requires a value"},
		{[]string{"--type", "postgres", "--driver", "cli"}, "not supported for PostgreSQL"},
		{[]string{"--type", "supabase", "--driver", "cli"}, "not supported for PostgreSQL"},
		{[]string{"--url", "postgresql://h/db", "--driver", "cli"}, "not supported for PostgreSQL"},
		{[]string{"--sslmode", "require"}, "only supported for PostgreSQL"},
		{[]string{"--type", "mysql", "--sslmode", "require"}, "only supported for PostgreSQL"},
		{[]string{"--type", "postgres", "--sslmode", "required"}, "invalid --sslmode"},
		{[]string{"--type", "mysql", "--url", "postgresql://h/db"}, "only supported for PostgreSQL"},
		{[]string{"--redash-url", "https://r", "--type", "postgres"}, "cannot be combined"},
	}
	for _, tc := range errCases {
		_, err := parseAddFlags(tc.args)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("parseAddFlags(%v) = %v, want error containing %q", tc.args, err, tc.want)
		}
	}
}

func decryptForTest(t *testing.T, s string) string {
	t.Helper()
	enc, err := crypto.UnmarshalEncrypted(s)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	plain, err := crypto.Decrypt(enc, []byte("master-pw"))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	return string(plain)
}

func TestRunAddPostgresFlags(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "pgpass")
	// "Use SSH?" -> n, port prompt -> Enter (5432), "Test connection?" -> n.
	withStdin(t, "n\n\nn\n")

	err := RunAdd([]string{
		"--type", "postgres", "--name", "pg", "--env", "development",
		"--db-host", "db.example.com", "--db-user", "app", "--db-name", "appdb",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("pg")
	if c == nil {
		t.Fatal("connection not added")
	}
	want := config.DBConfig{Host: "db.example.com", Port: 5432, User: "app", Database: "appdb", Type: config.DBTypePostgres}
	got := c.DB
	got.Password = ""
	if got != want {
		t.Errorf("DB = %+v, want %+v", got, want)
	}
	if decryptForTest(t, c.DB.Password) != "pgpass" {
		t.Error("password not encrypted with the prompted value")
	}
}

func TestRunAddSupabaseURL(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "unused")
	withStdin(t, "n\nn\n") // "Use SSH?" -> n, "Test connection?" -> n

	err := RunAdd([]string{
		"--type", "supabase", "--name", "sb", "--env", "development",
		"--url", "postgresql://postgres.abcdefghij:p%40ss%25w@aws-0-ap-northeast-1.pooler.supabase.com:6543/postgres",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("sb")
	if c == nil {
		t.Fatal("connection not added")
	}
	if c.DB.Type != config.DBTypePostgres || c.DB.SSLMode != "require" || c.DB.Driver != "" {
		t.Errorf("type/sslmode/driver = %q/%q/%q", c.DB.Type, c.DB.SSLMode, c.DB.Driver)
	}
	if c.DB.Host != "aws-0-ap-northeast-1.pooler.supabase.com" || c.DB.Port != 6543 || c.DB.User != "postgres.abcdefghij" || c.DB.Database != "postgres" {
		t.Errorf("DB = %+v", c.DB)
	}
	if got := decryptForTest(t, c.DB.Password); got != "p@ss%w" {
		t.Errorf("password = %q, want p@ss%%w", got)
	}
	data, _ := os.ReadFile(config.Path())
	if strings.Contains(string(data), "postgresql://") || strings.Contains(string(data), "p@ss") {
		t.Error("the connection string or plaintext password must not be saved")
	}
}

func TestRunAddURLPlaceholderPromptsForPassword(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "typed-password")
	withStdin(t, "n\nn\n")

	err := RunAdd([]string{
		"--type", "supabase", "--name", "sb", "--env", "development",
		"--url", "postgresql://postgres:[YOUR-PASSWORD]@db.abcdefghij.supabase.co:5432/postgres",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("sb")
	if c == nil {
		t.Fatal("connection not added")
	}
	if got := decryptForTest(t, c.DB.Password); got != "typed-password" {
		t.Errorf("password = %q, want the prompted value", got)
	}
}

func TestRunAddURLFlagsOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	withStdin(t, "n\nn\n")

	err := RunAdd([]string{
		"--name", "pg", "--env", "development",
		"--url", "postgresql://urluser:pw@url.example.com:5432/urldb?sslmode=require",
		"--db-port", "5433", "--db-user", "flaguser", "--sslmode", "verify-full",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("pg")
	if c == nil {
		t.Fatal("connection not added")
	}
	if c.DB.Host != "url.example.com" || c.DB.Port != 5433 || c.DB.User != "flaguser" || c.DB.Database != "urldb" {
		t.Errorf("DB = %+v", c.DB)
	}
	if c.DB.SSLMode != "verify-full" || c.DB.Type != config.DBTypePostgres {
		t.Errorf("sslmode/type = %q/%q", c.DB.SSLMode, c.DB.Type)
	}
}

func TestRunAddInteractiveSupabase(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "typed")
	// type=3 (Supabase), an invalid string then a valid one, "Use SSH?" -> n,
	// environment 3 (development), name, "Test connection?" -> n.
	withStdin(t, "3\nnot-a-url\npostgresql://postgres.abcdefghij:[YOUR-PASSWORD]@aws-0-ap-northeast-1.pooler.supabase.com:5432/postgres\nn\n3\nsb\nn\n")

	if err := RunAdd(nil); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("sb")
	if c == nil {
		t.Fatal("connection not added")
	}
	if c.DB.Type != config.DBTypePostgres || c.DB.SSLMode != "require" || c.DB.Port != 5432 || c.DB.User != "postgres.abcdefghij" {
		t.Errorf("DB = %+v", c.DB)
	}
	if decryptForTest(t, c.DB.Password) != "typed" {
		t.Error("password should come from the prompt")
	}
}

func TestRunAddInteractivePostgres(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	// type=2, "Use SSH?" -> n, host, port Enter (5432), user, database,
	// sslmode (invalid then verify-full), environment 3, name, test -> n.
	withStdin(t, "2\nn\ndb.example.com\n\napp\nappdb\nbogus\nverify-full\n3\npg\nn\n")

	if err := RunAdd(nil); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("pg")
	if c == nil {
		t.Fatal("connection not added")
	}
	want := config.DBConfig{Host: "db.example.com", Port: 5432, User: "app", Database: "appdb", Type: config.DBTypePostgres, SSLMode: "verify-full"}
	if c.DB != want {
		t.Errorf("DB = %+v, want %+v", c.DB, want)
	}
}

func TestRunAddInteractiveMySQLUnchanged(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	// type=1, "Use SSH?" -> n, host Enter (localhost), port Enter (3306), user,
	// database, environment 3, driver 1 (cli), name, test -> n.
	withStdin(t, "1\nn\n\n\nroot\napp\n3\n1\nmy\nn\n")

	if err := RunAdd(nil); err != nil {
		t.Fatalf("RunAdd: %v", err)
	}
	cfg, _ := config.Load()
	c := cfg.Find("my")
	if c == nil {
		t.Fatal("connection not added")
	}
	want := config.DBConfig{Host: "localhost", Port: 3306, User: "root", Database: "app", Driver: config.DriverCLI}
	if c.DB != want {
		t.Errorf("DB = %+v, want %+v", c.DB, want)
	}
}

func TestResolveConnectionPostgresDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := pgConn("pg")
	c.DB.Port = 0
	rc, err := resolveConnection(&c)
	if err != nil {
		t.Fatalf("resolveConnection: %v", err)
	}
	defer rc.cleanup()
	if rc.port != 5432 || rc.sslmode != "prefer" || !rc.isPostgres() || !rc.isNative() {
		t.Errorf("resolved = %+v", rc)
	}
	if rc.dialect() != db.DialectPostgres || rc.tlsServerName != "" {
		t.Errorf("dialect/tlsServerName = %v/%q", rc.dialect(), rc.tlsServerName)
	}

	mysqlConn := dbConn("my")
	rc2, err := resolveConnection(&mysqlConn)
	if err != nil {
		t.Fatal(err)
	}
	if rc2.isPostgres() || rc2.sslmode != "" || rc2.dialect() != db.DialectMySQL {
		t.Errorf("MySQL resolved = %+v", rc2)
	}
}

func TestResolveConnectionRejectsInvalidDBConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, d := range []config.DBConfig{
		{Host: "h", Type: "supabase"},
		{Host: "h", Type: config.DBTypePostgres, Driver: config.DriverCLI},
		{Host: "h", Type: config.DBTypePostgres, SSLMode: "bogus"},
	} {
		c := config.Connection{Name: "bad", DB: d}
		if _, err := resolveConnection(&c); err == nil {
			t.Errorf("resolveConnection(%+v): expected error", d)
		}
	}
}

func TestRunQueryPostgresCLIDriverRejected(t *testing.T) {
	c := pgConn("pg")
	c.DB.Driver = config.DriverCLI
	setupConfig(t, c)
	err := RunQuery([]string{"pg", "-e", "SELECT 1"})
	if err == nil || !strings.Contains(err.Error(), "not supported for PostgreSQL") {
		t.Errorf("expected driver rejection, got %v", err)
	}
}

// Both tunnel branches rewrite the host to the tunnel endpoint and keep the
// configured host for TLS verification.
func TestResolveConnectionPostgresTunnelServerName(t *testing.T) {
	t.Run("ad-hoc tunnel", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		orig := openAdHocTunnel
		var gotHost string
		var gotPort int
		openAdHocTunnel = func(_ *config.SSHConfig, host string, port int) (*tunnel.Tunnel, error) {
			gotHost, gotPort = host, port
			return &tunnel.Tunnel{LocalPort: 45678}, nil
		}
		t.Cleanup(func() { openAdHocTunnel = orig })

		c := pgConn("pg")
		c.DB.Host = "db.internal.example.com"
		c.DB.Port = 0
		c.DB.SSLMode = "verify-full"
		c.SSH = &config.SSHConfig{Host: "bastion", User: "deploy"}
		rc, err := resolveConnection(&c)
		if err != nil {
			t.Fatalf("resolveConnection: %v", err)
		}
		defer rc.cleanup()
		if gotHost != "db.internal.example.com" || gotPort != 5432 {
			t.Errorf("tunnel opened to %s:%d, want db.internal.example.com:5432", gotHost, gotPort)
		}
		if rc.host != "127.0.0.1" || rc.port != 45678 || rc.tlsServerName != "db.internal.example.com" || rc.sslmode != "verify-full" {
			t.Errorf("resolved = %+v", rc)
		}
	})

	t.Run("background tunnel", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		if err := config.EnsureDir(); err != nil {
			t.Fatal(err)
		}
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = l.Close() }()
		port := l.Addr().(*net.TCPAddr).Port
		if err := tunnel.SaveInfo(&tunnel.TunnelInfo{Name: "pg", PID: currentPID(), LocalPort: port, RemoteHost: "db.internal.example.com", RemotePort: 5432}); err != nil {
			t.Fatal(err)
		}

		c := pgConn("pg")
		c.DB.Host = "db.internal.example.com"
		c.SSH = &config.SSHConfig{Host: "bastion", User: "deploy"}
		rc, err := resolveConnection(&c)
		if err != nil {
			t.Fatalf("resolveConnection: %v", err)
		}
		defer rc.cleanup()
		if rc.host != "127.0.0.1" || rc.port != port || rc.tlsServerName != "db.internal.example.com" {
			t.Errorf("resolved = %+v", rc)
		}
	})
}

// Masking guarantee: a production PostgreSQL connection with mask rules must
// never print the raw value from `mysh run` (native path), even when stdout is
// a terminal and --mask is not given.
func TestRunQueryPostgresProductionMasked(t *testing.T) {
	c := pgConn("prod")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"email"}}
	setupConfig(t, c)
	mock := withPostgresMockDB(t)

	origTTY := stdoutIsTTY
	stdoutIsTTY = func() bool { return true }
	t.Cleanup(func() { stdoutIsTTY = origTTY })

	// The backslash is literal in a PostgreSQL string, so this is two
	// statements; the MySQL lexer would read it as one.
	mock.ExpectQuery(`SELECT '\\'`).WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(`\`))
	mock.ExpectQuery("SELECT id, email").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email"}).AddRow("1", "alice@example.com"))

	out, err := captureStdout(t, func() error {
		return RunQuery([]string{"prod", "-e", `SELECT '\'; SELECT id, email FROM users`})
	})
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("raw email leaked: %q", out)
	}
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("expected masked email, got %q", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestMCPQueryPostgresProductionMasked(t *testing.T) {
	c := pgConn("prod")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Patterns: []string{"*email*"}}
	setupConfig(t, c)
	mock := withPostgresMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_email"}).AddRow("1", "alice@example.com"))

	out, err := mcpQuery(map[string]any{"connection": "prod", "sql": "SELECT id, user_email FROM users", "format": "json"})
	if err != nil {
		t.Fatalf("mcpQuery: %v", err)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("raw email leaked over MCP: %q", out)
	}
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("expected masked email, got %q", out)
	}
}

func TestMCPListConnectionsPostgresKind(t *testing.T) {
	setupConfig(t, pgConn("pg"), dbConn("my"))
	out, err := mcpListConnections(nil)
	if err != nil {
		t.Fatal(err)
	}
	var infos []map[string]any
	if err := json.Unmarshal([]byte(out), &infos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	kinds := map[string]any{}
	for _, info := range infos {
		kinds[info["name"].(string)] = info["kind"]
	}
	if kinds["pg"] != "postgres" || kinds["my"] != "mysql" {
		t.Errorf("kinds = %v", kinds)
	}
}

func TestTablesPostgresQuery(t *testing.T) {
	setupConfig(t, pgConn("pg"))
	mock := withPostgresMockDB(t)
	rows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"schema", "table"}).AddRow("auth", "users").AddRow("public", "orders")
	}
	mock.ExpectQuery(`FROM information_schema\.tables`).WillReturnRows(rows())

	out, err := captureStdout(t, func() error { return RunTables([]string{"pg"}) })
	if err != nil {
		t.Fatalf("RunTables: %v", err)
	}
	if !strings.Contains(out, "auth") || !strings.Contains(out, "orders") {
		t.Errorf("unexpected tables output: %q", out)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}

	// RunTables closed its connection; mcpTables opens a fresh one.
	mock = withPostgresMockDB(t)
	mock.ExpectQuery(`FROM information_schema\.tables`).WillReturnRows(rows())
	mcpOut, err := mcpTables(map[string]any{"connection": "pg", "format": "csv"})
	if err != nil {
		t.Fatalf("mcpTables: %v", err)
	}
	if !strings.Contains(mcpOut, "schema,table") || !strings.Contains(mcpOut, "public,orders") {
		t.Errorf("unexpected MCP tables output: %q", mcpOut)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations: %v", err)
	}
}

func TestPostgresTablesQueryExcludesSystemSchemas(t *testing.T) {
	for _, s := range []string{"'pg_catalog'", "'information_schema'", `'pg\_toast%'`, `'pg\_temp\_%'`} {
		if !strings.Contains(postgresTablesQuery, s) {
			t.Errorf("tables query should exclude %s", s)
		}
	}
	if nativeTablesQuery(&resolvedConn{dbType: config.DBTypeMySQL}) != "SHOW TABLES" {
		t.Error("MySQL must keep SHOW TABLES")
	}
}

func TestRunSlicePostgresUnsupported(t *testing.T) {
	setupConfig(t, pgConn("pg"))
	err := RunSlice([]string{"pg", "users", "--where", "id = 1"})
	if err == nil || !strings.Contains(err.Error(), "not supported for PostgreSQL") {
		t.Errorf("expected unsupported error, got %v", err)
	}
}

func TestExportImportPostgresRoundTrip(t *testing.T) {
	src := pgConn("sb")
	src.DB.SSLMode = "require"
	src.DB.Password = "never-exported"
	exported := buildExported([]config.Connection{src, dbConn("my")})
	data, err := yaml.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "type: postgres") || !strings.Contains(s, "sslmode: require") {
		t.Errorf("export missing type/sslmode:\n%s", s)
	}
	if strings.Contains(s, "never-exported") {
		t.Error("password must not be exported")
	}
	if strings.Count(s, "type:") != 1 {
		t.Errorf("MySQL connection should not gain a type field:\n%s", s)
	}

	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	withFakePassword(t, "")
	path := filepath.Join(dir, "export.yaml")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	// "Add without password?" -> y for each connection, then decline masking.
	withStdin(t, "y\ny\nn\n")
	if err := RunImport([]string{"--from", "yaml", "--file", path, "--all"}); err != nil {
		t.Fatalf("RunImport: %v", err)
	}
	cfg, _ := config.Load()
	got := cfg.Find("sb")
	if got == nil {
		t.Fatal("postgres connection not imported")
	}
	if got.DB.Type != config.DBTypePostgres || got.DB.SSLMode != "require" || got.DB.Driver != "" || got.DB.Port != 5432 {
		t.Errorf("imported DB = %+v", got.DB)
	}
	if my := cfg.Find("my"); my == nil || my.DB.Type != "" || my.DB.Driver != config.DriverCLI {
		t.Errorf("imported MySQL = %+v", my)
	}
}

func TestImportSkipsInvalidDBType(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	withFakePassword(t, "")
	path := filepath.Join(dir, "bad.yaml")
	content := "- name: bad\n  db:\n    host: h\n    user: u\n    database: d\n    type: supabase\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "")
	if err := RunImport([]string{"--from", "yaml", "--file", path, "--all"}); err != nil {
		t.Fatalf("RunImport: %v", err)
	}
	cfg, _ := config.Load()
	if cfg.Find("bad") != nil {
		t.Error("connection with an unknown type must not be imported")
	}
}

func TestRunEditPostgres(t *testing.T) {
	c := pgConn("pg")
	c.DB.Driver = config.DriverNative
	setupConfig(t, c)
	withFakePassword(t, "")
	// "Use SSH?" -> n, host/port/user/db Enter, sslmode verify-full,
	// environment Enter (development). No driver menu for PostgreSQL.
	withStdin(t, "n\n\n\n\n\nverify-full\n\n")
	if err := RunEdit([]string{"pg"}); err != nil {
		t.Fatalf("RunEdit: %v", err)
	}
	cfg, _ := config.Load()
	got := cfg.Find("pg")
	if got.DB.SSLMode != "verify-full" || got.DB.Driver != "" || got.DB.Type != config.DBTypePostgres || got.DB.Port != 5432 {
		t.Errorf("edited DB = %+v", got.DB)
	}

	// Accepting the default sslmode keeps an unset value unset.
	c2 := pgConn("pg2")
	setupConfig(t, c2)
	withStdin(t, "n\n\n\n\n\n\n\n")
	if err := RunEdit([]string{"pg2"}); err != nil {
		t.Fatalf("RunEdit: %v", err)
	}
	cfg, _ = config.Load()
	if s := cfg.Find("pg2").DB.SSLMode; s != "" {
		t.Errorf("sslmode = %q, want unset", s)
	}
}

func TestREPLPostgresPrompt(t *testing.T) {
	var out, errOut strings.Builder
	query := func(string) ([]string, [][]string, error) { return nil, nil, nil }
	if err := runREPL(strings.NewReader("SELECT\n1;\nquit\n"), &out, &errOut, true, "postgres> ", query); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "postgres> ") || !strings.Contains(errOut.String(), "       -> ") {
		t.Errorf("unexpected prompts: %q", errOut.String())
	}
}
