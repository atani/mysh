package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/format"
	"github.com/atani/mysh/internal/mysql"
)

func TestParseAddFlags(t *testing.T) {
	f, err := parseAddFlags([]string{
		"--name", "prod",
		"--env", "production",
		"--db-host", "db.example.com",
		"--db-port", "3307",
		"--db-user", "app",
		"--db-name", "myapp",
		"--ssh-host", "bastion",
		"--ssh-port", "2222",
		"--ssh-user", "deploy",
		"--ssh-key", "/key",
		"--driver", "native",
		"--mask", "email",
		"--redash-url", "https://redash",
		"--redash-key", "k",
		"--redash-datasource", "5",
	})
	if err != nil {
		t.Fatalf("parseAddFlags: %v", err)
	}
	if f.name != "prod" || f.env != "production" || f.dbHost != "db.example.com" {
		t.Errorf("string flags wrong: %+v", f)
	}
	if f.dbPort != 3307 || f.sshPort != 2222 || f.redashDatasource != 5 {
		t.Errorf("int flags wrong: dbPort=%d sshPort=%d ds=%d", f.dbPort, f.sshPort, f.redashDatasource)
	}
	if f.driver != "native" || f.mask != "email" || f.redashURL != "https://redash" {
		t.Errorf("misc flags wrong: %+v", f)
	}
}

func TestParseAddFlagsDefaults(t *testing.T) {
	f, err := parseAddFlags(nil)
	if err != nil {
		t.Fatalf("parseAddFlags(nil): %v", err)
	}
	// Ports default to -1 to distinguish "unset" from explicit 0.
	if f.dbPort != -1 || f.sshPort != -1 || f.redashDatasource != -1 {
		t.Errorf("expected -1 defaults, got dbPort=%d sshPort=%d ds=%d", f.dbPort, f.sshPort, f.redashDatasource)
	}
}

func TestParseAddFlagsErrors(t *testing.T) {
	cases := [][]string{
		{"--name"},                     // missing value
		{"--db-port", "abc"},           // invalid int
		{"--ssh-port", "x"},            // invalid int
		{"--redash-datasource", "nan"}, // invalid int
		{"--unknown-flag"},             // unknown flag
		{"--db-port"},                  // missing int value
	}
	for _, args := range cases {
		if _, err := parseAddFlags(args); err == nil {
			t.Errorf("parseAddFlags(%v) expected error", args)
		}
	}
}

func TestWriteDefaultsFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &resolvedConn{password: `p@ss'w\ord`}
	path, cleanup, err := rc.writeDefaultsFile()
	if err != nil {
		t.Fatalf("writeDefaultsFile: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read defaults file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "[client]") {
		t.Errorf("missing [client] section:\n%s", content)
	}
	// Backslash and single-quote must be escaped.
	if !strings.Contains(content, `\\`) || !strings.Contains(content, `\'`) {
		t.Errorf("special chars not escaped:\n%s", content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 {
		t.Errorf("defaults file is group/other readable: %o", info.Mode().Perm())
	}

	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("cleanup should remove the defaults file")
	}
}

func TestWriteDefaultsFileNoPassword(t *testing.T) {
	rc := &resolvedConn{password: ""}
	path, cleanup, err := rc.writeDefaultsFile()
	if err != nil {
		t.Fatalf("writeDefaultsFile: %v", err)
	}
	defer cleanup()
	if path != "" {
		t.Errorf("expected empty path when no password, got %q", path)
	}
}

func TestMysqlArgsWithPassword(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	rc := &resolvedConn{host: "h", port: 3306, user: "u", database: "d", password: "secret"}
	args, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		t.Fatalf("mysqlArgsWithPassword: %v", err)
	}
	defer cleanup()
	if len(args) == 0 || !strings.HasPrefix(args[0], "--defaults-extra-file=") {
		t.Errorf("expected --defaults-extra-file first, got %v", args)
	}
}

func TestMysqlArgsWithPasswordNoPassword(t *testing.T) {
	rc := &resolvedConn{host: "h", port: 3306, user: "u", database: "d"}
	args, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		t.Fatalf("mysqlArgsWithPassword: %v", err)
	}
	defer cleanup()
	for _, a := range args {
		if strings.HasPrefix(a, "--defaults-extra-file=") {
			t.Error("no defaults file should be used when password is empty")
		}
	}
}

func TestDecryptRedashAPIKeyEmpty(t *testing.T) {
	conn := &config.Connection{Redash: &config.RedashConfig{APIKey: ""}}
	_, err := decryptRedashAPIKey(conn)
	if err == nil {
		t.Error("expected error for missing API key")
	}
}

func TestDecryptRedashAPIKeyPlaintextFallback(t *testing.T) {
	// A non-encrypted value is treated as a plaintext key (with a warning).
	conn := &config.Connection{Redash: &config.RedashConfig{APIKey: "raw-plaintext-key"}}
	got, err := decryptRedashAPIKey(conn)
	if err != nil {
		t.Fatalf("decryptRedashAPIKey: %v", err)
	}
	if got != "raw-plaintext-key" {
		t.Errorf("got %q, want plaintext key", got)
	}
}

func TestApplyMasking(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"id", "email"},
		Rows:    [][]string{{"1", "alice@example.com"}},
	}
	conn := &config.Connection{Mask: &config.MaskConfig{Columns: []string{"email"}}}

	applyMasking(result, conn, true)

	if result.Rows[0][0] != "1" {
		t.Errorf("id column should be untouched, got %q", result.Rows[0][0])
	}
	if result.Rows[0][1] != "a***@example.com" {
		t.Errorf("email should be masked, got %q", result.Rows[0][1])
	}
}

func TestApplyMaskingDisabled(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"email"},
		Rows:    [][]string{{"alice@example.com"}},
	}
	conn := &config.Connection{Mask: &config.MaskConfig{Columns: []string{"email"}}}

	applyMasking(result, conn, false)

	if result.Rows[0][0] != "alice@example.com" {
		t.Errorf("masking disabled should leave value unchanged, got %q", result.Rows[0][0])
	}
}

func TestApplyMaskingNoMatchingColumns(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"id"},
		Rows:    [][]string{{"1"}},
	}
	conn := &config.Connection{Mask: &config.MaskConfig{Columns: []string{"email"}}}

	applyMasking(result, conn, true)

	if result.Rows[0][0] != "1" {
		t.Errorf("no matching column should leave value unchanged, got %q", result.Rows[0][0])
	}
}

func TestWriteSliceOutputToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "slice.sql")
	result := &mysql.QueryResult{
		Headers: []string{"id", "name"},
		Rows:    [][]string{{"1", "Alice"}, {"2", "Bob"}},
	}

	if err := writeSliceOutput(result, "users", "id > 0", path); err != nil {
		t.Fatalf("writeSliceOutput: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dump := string(data)
	if !strings.Contains(dump, "INSERT INTO") || !strings.Contains(strings.ToLower(dump), "users") {
		t.Errorf("unexpected slice dump:\n%s", dump)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("slice file perm = %o, want 0600", info.Mode().Perm())
	}
}

func TestWriteOutputToFileCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")
	input := "id\tname\n1\tAlice\n"

	if err := writeOutput(input, format.CSV, path); err != nil {
		t.Fatalf("writeOutput: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "id,name") {
		t.Errorf("CSV output unexpected:\n%s", data)
	}
}

func TestWriteOutputToFilePDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.pdf")
	input := "id\tname\n1\tAlice\n"

	if err := writeOutput(input, format.PDF, path); err != nil {
		t.Fatalf("writeOutput PDF: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}

func TestWriteOutputStructuredToFile(t *testing.T) {
	dir := t.TempDir()

	headers := []string{"id", "name"}
	rows := [][]string{{"1", "Alice"}}

	t.Run("csv", func(t *testing.T) {
		path := filepath.Join(dir, "s.csv")
		if err := writeOutputStructured(headers, rows, format.CSV, path); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "id,name") {
			t.Errorf("unexpected csv:\n%s", data)
		}
	})

	t.Run("json", func(t *testing.T) {
		path := filepath.Join(dir, "s.json")
		if err := writeOutputStructured(headers, rows, format.JSON, path); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), `"name"`) {
			t.Errorf("unexpected json:\n%s", data)
		}
	})

	t.Run("plain to file", func(t *testing.T) {
		path := filepath.Join(dir, "s.txt")
		if err := writeOutputStructured(headers, rows, format.Plain, path); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if !strings.Contains(string(data), "Alice") {
			t.Errorf("plain output should contain data:\n%s", data)
		}
	})

	t.Run("pdf", func(t *testing.T) {
		path := filepath.Join(dir, "s.pdf")
		if err := writeOutputStructured(headers, rows, format.PDF, path); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() == 0 {
			t.Error("PDF is empty")
		}
	})
}
