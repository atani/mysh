package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
)

// writeFile is a small test helper that writes content to path.
func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0600)
}

// setupConfig points the config package at a fresh temp directory and writes the
// given connections to it. It returns the config dir.
func setupConfig(t *testing.T, conns ...config.Connection) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := &config.Config{Connections: conns}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return filepath.Join(dir, "mysh")
}

func dbConn(name string) config.Connection {
	return config.Connection{
		Name: name,
		Env:  "development",
		DB:   config.DBConfig{Host: "127.0.0.1", Port: 3306, User: "root", Database: "test", Driver: config.DriverCLI},
	}
}

func TestFindConnectionNoConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, _, err := findConnection("")
	if err == nil || !strings.Contains(err.Error(), "no connections") {
		t.Errorf("expected no-connections error, got %v", err)
	}
}

func TestFindConnectionSingleAutoSelect(t *testing.T) {
	setupConfig(t, dbConn("only"))
	_, conn, err := findConnection("")
	if err != nil {
		t.Fatalf("findConnection: %v", err)
	}
	if conn.Name != "only" {
		t.Errorf("got %q, want only", conn.Name)
	}
}

func TestFindConnectionMultipleRequiresName(t *testing.T) {
	setupConfig(t, dbConn("a"), dbConn("b"))
	_, _, err := findConnection("")
	if err == nil || !strings.Contains(err.Error(), "multiple connections") {
		t.Errorf("expected multiple-connections error, got %v", err)
	}
}

func TestFindConnectionNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	_, _, err := findConnection("nope")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestFindConnectionByName(t *testing.T) {
	setupConfig(t, dbConn("a"), dbConn("b"))
	_, conn, err := findConnection("b")
	if err != nil {
		t.Fatalf("findConnection: %v", err)
	}
	if conn.Name != "b" {
		t.Errorf("got %q, want b", conn.Name)
	}
}

// --- Entry-function argument/validation paths (no DB required) ---

func TestRunQueryParseErrors(t *testing.T) {
	if err := RunQuery([]string{"--format"}); err == nil {
		t.Error("expected error")
	}
	// Valid parse but unknown connection.
	setupConfig(t, dbConn("a"))
	if err := RunQuery([]string{"nope", "-e", "SELECT 1"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunSliceParseErrors(t *testing.T) {
	if err := RunSlice([]string{"prod", "us`ers", "--where", "x"}); err == nil {
		t.Error("expected backtick error")
	}
	setupConfig(t, dbConn("a"))
	if err := RunSlice([]string{"nope", "users", "--where", "id>0"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunTablesParseErrors(t *testing.T) {
	if err := RunTables([]string{"a", "b"}); err == nil {
		t.Error("expected unexpected-argument error")
	}
	setupConfig(t, dbConn("a"))
	if err := RunTables([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunPingNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunPing([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunConnectNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunConnect([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunEditNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunEdit([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunRemoveNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunRemove([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunExport(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := RunExport(nil); err == nil || !strings.Contains(err.Error(), "no connections") {
		t.Errorf("expected no-connections error, got %v", err)
	}

	setupConfig(t, dbConn("a"))
	if err := RunExport(nil); err != nil {
		t.Errorf("RunExport: %v", err)
	}
	if err := RunExport([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
	if err := RunExport([]string{"a"}); err != nil {
		t.Errorf("RunExport single: %v", err)
	}
}

func TestRunListEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := RunList(nil); err != nil {
		t.Errorf("RunList empty: %v", err)
	}
}

func TestRunList(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunList(nil); err != nil {
		t.Errorf("RunList: %v", err)
	}
}

func TestRunQueriesNoDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := RunQueries(nil); err != nil {
		t.Errorf("RunQueries no dir: %v", err)
	}
}

func TestRunQueriesWithFiles(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	qdir := filepath.Join(dir, "mysh", "queries")
	if err := os.MkdirAll(qdir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(t, filepath.Join(qdir, "a.sql"), "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(t, filepath.Join(qdir, "ignore.txt"), "x"); err != nil {
		t.Fatal(err)
	}
	if err := RunQueries(nil); err != nil {
		t.Errorf("RunQueries: %v", err)
	}
}

func TestRunQueriesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	qdir := filepath.Join(dir, "mysh", "queries")
	if err := os.MkdirAll(qdir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := RunQueries(nil); err != nil {
		t.Errorf("RunQueries empty: %v", err)
	}
}

func TestRunTunnelList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// No active tunnels: should print "No active tunnels." and succeed.
	if err := RunTunnel(nil); err != nil {
		t.Errorf("RunTunnel list: %v", err)
	}
	if err := RunTunnel([]string{"list"}); err != nil {
		t.Errorf("RunTunnel list explicit: %v", err)
	}
}

func TestRunTunnelOpenNoSSH(t *testing.T) {
	setupConfig(t, dbConn("a"))
	// Connection without SSH config should error.
	if err := RunTunnel([]string{"a"}); err == nil || !strings.Contains(err.Error(), "no SSH config") {
		t.Errorf("expected no-SSH error, got %v", err)
	}
}

func TestRunTunnelOpenNotFound(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunTunnel([]string{"nope"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found, got %v", err)
	}
}

func TestRunTunnelStopNotRunning(t *testing.T) {
	setupConfig(t, dbConn("a"))
	if err := RunTunnel([]string{"stop", "a"}); err == nil {
		t.Error("expected error stopping a non-running tunnel")
	}
}

func TestUsage(t *testing.T) {
	// Usage writes to stdout; just ensure it runs without panic.
	Usage()
}

// --- REPL loop ---

func TestRunREPL(t *testing.T) {
	type queryCall struct{ stmt string }
	var calls []queryCall
	query := func(stmt string) ([]string, [][]string, error) {
		calls = append(calls, queryCall{stmt})
		switch {
		case strings.HasPrefix(stmt, "SELECT"):
			return []string{"id"}, [][]string{{"1"}, {"2"}}, nil
		case strings.HasPrefix(stmt, "BAD"):
			return nil, nil, errors.New("syntax error")
		default:
			return nil, nil, nil // OK, no rows
		}
	}

	input := strings.Join([]string{
		"",                 // blank line skipped
		"SELECT * FROM t;", // multi-row result
		"UPDATE t SET x=1", // continuation (no semicolon)
		";",                // completes the UPDATE -> Query OK
		";",                // bare semicolon -> empty statement, skipped
		"BAD SQL;",         // error path
		"quit",             // exit
	}, "\n") + "\n"

	var out, errOut bytes.Buffer
	if err := runREPL(strings.NewReader(input), &out, &errOut, true, query); err != nil {
		t.Fatalf("runREPL: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("expected 3 query calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].stmt != "SELECT * FROM t" {
		t.Errorf("first stmt = %q", calls[0].stmt)
	}
	if calls[1].stmt != "UPDATE t SET x=1" {
		t.Errorf("second stmt = %q (continuation join)", calls[1].stmt)
	}
	if !strings.Contains(out.String(), "1") {
		t.Errorf("output missing rows: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "Query OK") {
		t.Errorf("errOut missing Query OK: %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "ERROR: syntax error") {
		t.Errorf("errOut missing ERROR: %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "Bye") {
		t.Errorf("errOut missing Bye: %q", errOut.String())
	}
	if !strings.Contains(errOut.String(), "mysql>") {
		t.Errorf("TTY prompt missing: %q", errOut.String())
	}
}

func TestRunREPLEOF(t *testing.T) {
	query := func(string) ([]string, [][]string, error) { return nil, nil, nil }
	var out, errOut bytes.Buffer
	// Non-TTY, ends with EOF (no quit).
	if err := runREPL(strings.NewReader("SELECT 1;\n"), &out, &errOut, false, query); err != nil {
		t.Fatalf("runREPL: %v", err)
	}
	if strings.Contains(errOut.String(), "mysql>") {
		t.Error("non-TTY should not print prompt")
	}
}

// errReader fails on the first Read so runREPL's scanner.Err() branch is hit.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failure") }

func TestRunREPLScannerError(t *testing.T) {
	query := func(string) ([]string, [][]string, error) { return nil, nil, nil }
	var out, errOut bytes.Buffer
	err := runREPL(errReader{}, &out, &errOut, false, query)
	if err == nil || !strings.Contains(err.Error(), "reading input") {
		t.Errorf("expected reading-input error, got %v", err)
	}
}
