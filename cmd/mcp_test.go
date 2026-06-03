package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/format"
	"github.com/atani/mysh/internal/i18n"
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
	if out != i18n.T(i18n.McpNoConnections) {
		t.Errorf("expected empty message, got %q", out)
	}
}

func TestMCPQueryRequiresSQL(t *testing.T) {
	_, err := mcpQuery(map[string]any{})
	if err == nil {
		t.Error("expected error when sql argument is missing")
	}
}

// The following tests drive the native-driver query/tables/ping paths through
// a sqlmock database injected via the dbOpen indirection (added in #100).

func TestMCPQueryNativeSuccess(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "alice"))

	out, err := mcpQuery(map[string]any{"connection": "n", "sql": "SELECT id, name FROM users"})
	if err != nil {
		t.Fatalf("mcpQuery native: %v", err)
	}
	if !strings.Contains(out, "alice") || !strings.Contains(out, "id") {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestMCPQueryNativeMasked(t *testing.T) {
	// A production connection with a mask rule must mask sensitive columns in
	// the result returned over MCP.
	c := nativeConn("p")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"email"}}
	setupConfig(t, c)
	_, mock := withMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email"}).AddRow("1", "alice@example.com"))

	out, err := mcpQuery(map[string]any{"connection": "p", "sql": "SELECT id, email FROM users", "format": "csv"})
	if err != nil {
		t.Fatalf("mcpQuery native masked: %v", err)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("email must be masked, got %q", out)
	}
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("expected masked email, got %q", out)
	}
}

func TestMCPQueryNativeQueryOK(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	// A statement that returns no columns yields the "Query OK" path.
	mock.ExpectQuery("UPDATE").WillReturnRows(sqlmock.NewRows(nil))

	out, err := mcpQuery(map[string]any{"connection": "n", "sql": "UPDATE t SET x=1"})
	if err != nil {
		t.Fatalf("mcpQuery native query-ok: %v", err)
	}
	if out != i18n.T(i18n.McpQueryOK) {
		t.Errorf("expected %q, got %q", i18n.T(i18n.McpQueryOK), out)
	}
}

func TestMCPTablesNative(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SHOW TABLES").WillReturnRows(
		sqlmock.NewRows([]string{"Tables_in_test"}).AddRow("users").AddRow("orders"))

	out, err := mcpTables(map[string]any{"connection": "n"})
	if err != nil {
		t.Fatalf("mcpTables native: %v", err)
	}
	if !strings.Contains(out, "users") || !strings.Contains(out, "orders") {
		t.Errorf("unexpected tables output: %q", out)
	}
}

func TestMCPTablesRedashUnsupported(t *testing.T) {
	c := config.Connection{Name: "r", Redash: &config.RedashConfig{URL: "https://redash.example.com"}}
	setupConfig(t, c)
	if _, err := mcpTables(map[string]any{"connection": "r"}); err == nil {
		t.Error("expected error: tables not supported for Redash")
	}
}

func TestMCPPingNative(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectPing()

	out, err := mcpPing(map[string]any{"connection": "n"})
	if err != nil {
		t.Fatalf("mcpPing native: %v", err)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK, got %q", out)
	}
}

// The CLI-driver paths are exercised through stubCLI (added in #100), which
// replaces the external mysql client with a helper process.

func TestMCPQueryCLI(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")

	out, err := mcpQuery(map[string]any{"connection": "cli", "sql": "SELECT 1", "format": "csv"})
	if err != nil {
		t.Fatalf("mcpQuery CLI: %v", err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("unexpected CLI output: %q", out)
	}
}

func TestMCPQueryCLIMasked(t *testing.T) {
	// Production + a mask rule matching the helper's "name" column exercises the
	// masking branch of the CLI path.
	c := dbConn("p")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"name"}}
	setupConfig(t, c)
	stubCLI(t, "ok")

	out, err := mcpQuery(map[string]any{"connection": "p", "sql": "SELECT 1"})
	if err != nil {
		t.Fatalf("mcpQuery CLI masked: %v", err)
	}
	if strings.Contains(out, "alice") {
		t.Errorf("name column should be masked, got %q", out)
	}
}

func TestMCPQueryCLIError(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "fail")
	if _, err := mcpQuery(map[string]any{"connection": "cli", "sql": "SELECT 1"}); err == nil {
		t.Error("expected error when the mysql client fails")
	}
}

func TestMCPTablesCLI(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")

	out, err := mcpTables(map[string]any{"connection": "cli", "format": "csv"})
	if err != nil {
		t.Fatalf("mcpTables CLI: %v", err)
	}
	if !strings.Contains(out, "alice") {
		t.Errorf("unexpected tables output: %q", out)
	}
}

func TestMCPPingCLI(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")

	out, err := mcpPing(map[string]any{"connection": "cli"})
	if err != nil {
		t.Fatalf("mcpPing CLI: %v", err)
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK, got %q", out)
	}
}

func TestMCPPingCLIFail(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "fail")
	if _, err := mcpPing(map[string]any{"connection": "cli"}); err == nil {
		t.Error("expected ping failure")
	}
}

func TestMCPQueryRedashMasked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_result":{"data":{` +
			`"columns":[{"name":"id","type":"integer"},{"name":"email","type":"string"}],` +
			`"rows":[{"id":1,"email":"alice@example.com"}]}}}`))
	}))
	defer server.Close()

	c := config.Connection{
		Name:   "r",
		Env:    "production",
		Redash: &config.RedashConfig{URL: server.URL, APIKey: "plain-key", DataSourceID: 1},
		Mask:   &config.MaskConfig{Columns: []string{"email"}},
	}
	setupConfig(t, c)

	out, err := mcpQuery(map[string]any{"connection": "r", "sql": "SELECT id, email FROM users", "format": "csv"})
	if err != nil {
		t.Fatalf("mcpQuery redash: %v", err)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("email must be masked, got %q", out)
	}
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("expected masked email, got %q", out)
	}
}

// TestRunMCPHandshake drives the full MCP command in-process: it redirects
// stdin/stdout to pipes, sends an initialize request, and verifies the server
// responds and exits cleanly on EOF.
func TestRunMCPHandshake(t *testing.T) {
	setupConfig(t, dbConn("n"))

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origIn, origOut := os.Stdin, os.Stdout
	os.Stdin, os.Stdout = inR, outW
	defer func() {
		os.Stdin, os.Stdout = origIn, origOut
		nonInteractive = false // RunMCP sets this global; reset for other tests
	}()

	if _, err := io.WriteString(inW, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`+"\n"); err != nil {
		t.Fatal(err)
	}
	_ = inW.Close() // EOF makes Serve return

	done := make(chan error, 1)
	go func() { done <- RunMCP(nil) }()
	if err := <-done; err != nil {
		t.Fatalf("RunMCP: %v", err)
	}
	_ = outW.Close()

	data, _ := io.ReadAll(outR)
	if !strings.Contains(string(data), `"serverInfo"`) {
		t.Errorf("expected handshake response, got %q", string(data))
	}
	if !nonInteractive {
		t.Error("RunMCP should set nonInteractive while serving")
	}
}

func TestSetVersion(t *testing.T) {
	orig := mcpVersion
	defer func() { mcpVersion = orig }()

	SetVersion("9.9.9")
	if mcpVersion != "9.9.9" {
		t.Errorf("SetVersion did not apply, got %q", mcpVersion)
	}
	SetVersion("") // empty must not overwrite an existing version
	if mcpVersion != "9.9.9" {
		t.Errorf("empty version should not overwrite, got %q", mcpVersion)
	}
}

// TestNonInteractiveMasterPasswordFailsFast verifies that when running in
// non-interactive (MCP) mode, a missing master password produces an actionable
// error instead of falling through to a stdin password prompt (which would
// corrupt the JSON-RPC stream).
func TestNonInteractiveMasterPasswordFailsFast(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("MYSH_MASTER_PASSWORD", "")

	// Initialize a known master password in the temp config dir. This makes the
	// test hermetic regardless of any real entry in the developer's OS keychain:
	// a keychain password (if present) will fail verification against this
	// verifier, so resolution falls through to the non-interactive guard.
	if err := config.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	if err := crypto.InitMasterPassword([]byte("test-only-master-password-xyz")); err != nil {
		t.Fatal(err)
	}

	prev := nonInteractive
	nonInteractive = true
	defer func() { nonInteractive = prev }()

	_, err := getMasterPassword()
	if err == nil {
		t.Fatal("expected an error when master password is unavailable in non-interactive mode")
	}
	if !strings.Contains(err.Error(), "MYSH_MASTER_PASSWORD") {
		t.Errorf("error should mention how to supply the master password, got: %v", err)
	}
}
