package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
)

// captureStdout redirects os.Stdout to a pipe for the duration of fn and returns
// everything written to it. Export writes to os.Stdout directly, so this lets us
// assert on the marshaled YAML.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	runErr := fn()
	_ = w.Close()
	os.Stdout = orig
	data, _ := io.ReadAll(r)
	_ = r.Close()
	return string(data), runErr
}

// writeSavedQuery writes a .sql file into QueriesDir so export --with-queries can
// pick it up. XDG_CONFIG_HOME must already point at a temp dir (via setupConfig).
func writeSavedQuery(t *testing.T, name, body string) {
	t.Helper()
	if err := os.MkdirAll(config.QueriesDir(), 0700); err != nil {
		t.Fatalf("mkdir queries: %v", err)
	}
	path := filepath.Join(config.QueriesDir(), name+".sql")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write query %s: %v", name, err)
	}
}

func TestParseExportFlags(t *testing.T) {
	// Name only.
	opts, err := parseExportFlags([]string{"prod"})
	if err != nil || opts.name != "prod" || opts.withQueries {
		t.Errorf("name only: got %+v err=%v", opts, err)
	}

	// Flag only.
	opts, err = parseExportFlags([]string{"--with-queries"})
	if err != nil || opts.name != "" || !opts.withQueries {
		t.Errorf("flag only: got %+v err=%v", opts, err)
	}

	// Name and flag in either order.
	for _, args := range [][]string{{"prod", "--with-queries"}, {"--with-queries", "prod"}} {
		opts, err = parseExportFlags(args)
		if err != nil || opts.name != "prod" || !opts.withQueries {
			t.Errorf("args %v: got %+v err=%v", args, opts, err)
		}
	}

	// Errors: unknown flag, duplicate positional.
	if _, err := parseExportFlags([]string{"--bogus"}); err == nil {
		t.Error("expected error for unknown flag")
	}
	if _, err := parseExportFlags([]string{"a", "b"}); err == nil {
		t.Error("expected error for extra positional argument")
	}
}

func TestLoadSavedQueries(t *testing.T) {
	// Missing queries directory -> empty, no error.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := loadSavedQueries()
	if err != nil {
		t.Fatalf("loadSavedQueries (missing dir): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no queries, got %v", got)
	}

	// With saved .sql files (and a non-.sql file that must be ignored).
	setupConfig(t, dbConn("a"))
	writeSavedQuery(t, "active_users", "SELECT * FROM users WHERE active = 1;")
	writeSavedQuery(t, "recent_orders", "SELECT * FROM orders ORDER BY id DESC LIMIT 10;")
	if err := os.WriteFile(filepath.Join(config.QueriesDir(), "notes.txt"), []byte("ignore me"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err = loadSavedQueries()
	if err != nil {
		t.Fatalf("loadSavedQueries: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 queries, got %d: %v", len(got), got)
	}
	byName := map[string]string{}
	for _, q := range got {
		byName[q.Name] = q.Query
	}
	if !strings.Contains(byName["active_users"], "WHERE active = 1") {
		t.Errorf("active_users body wrong: %q", byName["active_users"])
	}
	if !strings.Contains(byName["recent_orders"], "ORDER BY id DESC") {
		t.Errorf("recent_orders body wrong: %q", byName["recent_orders"])
	}
}

func TestRunExportWithQueries(t *testing.T) {
	setupConfig(t, dbConn("a"), dbConn("b"))
	writeSavedQuery(t, "active_users", "SELECT * FROM users WHERE active = 1;")
	writeSavedQuery(t, "recent_orders", "SELECT * FROM orders LIMIT 10;")

	out, err := captureStdout(t, func() error {
		return RunExport([]string{"--with-queries"})
	})
	if err != nil {
		t.Fatalf("RunExport --with-queries: %v", err)
	}

	// Bundle form: both top-level keys present.
	for _, want := range []string{"connections:", "queries:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	// Both connections and both query names/bodies are included.
	for _, want := range []string{
		"name: a", "name: b",
		"name: active_users", "name: recent_orders",
		"WHERE active = 1", "FROM orders LIMIT 10",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestRunExportSingleConnectionWithQueries(t *testing.T) {
	setupConfig(t, dbConn("a"), dbConn("b"))
	writeSavedQuery(t, "active_users", "SELECT 1;")

	out, err := captureStdout(t, func() error {
		return RunExport([]string{"a", "--with-queries"})
	})
	if err != nil {
		t.Fatalf("RunExport a --with-queries: %v", err)
	}

	// Only the requested connection is exported.
	if !strings.Contains(out, "name: a") {
		t.Errorf("expected connection a in output:\n%s", out)
	}
	if strings.Contains(out, "name: b") {
		t.Errorf("connection b should not be exported:\n%s", out)
	}
	// Saved queries have no per-connection association, so they are still
	// included alongside a single-connection export.
	if !strings.Contains(out, "queries:") || !strings.Contains(out, "name: active_users") {
		t.Errorf("expected saved queries in output:\n%s", out)
	}
}

func TestRunExportWithoutQueriesOmitsQueries(t *testing.T) {
	setupConfig(t, dbConn("a"))
	writeSavedQuery(t, "active_users", "SELECT 1;")

	out, err := captureStdout(t, func() error {
		return RunExport(nil)
	})
	if err != nil {
		t.Fatalf("RunExport: %v", err)
	}

	// Backward compatible: bare connection list, no queries: key.
	if strings.Contains(out, "queries:") {
		t.Errorf("plain export must not contain queries:\n%s", out)
	}
	if strings.Contains(out, "connections:") {
		t.Errorf("plain export must stay a bare list (no connections: key):\n%s", out)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "- name: a") {
		t.Errorf("plain export should start with a bare list item:\n%s", out)
	}
}
