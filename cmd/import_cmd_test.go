package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/importer"
)

// withStdin temporarily replaces os.Stdin with a pipe carrying content, so
// functions that read from os.Stdin directly (e.g. the import selection loop)
// can be driven deterministically.
func withStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(content); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
}

// withFakePassword installs a deterministic password reader and stub keychain
// for the duration of the test, so encryption paths can run without a TTY or
// the real OS credential store.
func withFakePassword(t *testing.T, pw string) {
	t.Helper()
	origRead := readPassword
	origGet := keychainGet
	origSet := keychainSet
	readPassword = func() (string, error) { return pw, nil }
	keychainGet = func() (string, error) { return "", os.ErrNotExist }
	keychainSet = func(string) error { return nil }
	// MYSH_MASTER_PASSWORD lets getMasterPassword skip the interactive setup
	// once a master password is initialized.
	t.Setenv("MYSH_MASTER_PASSWORD", "master-pw")
	t.Cleanup(func() {
		readPassword = origRead
		keychainGet = origGet
		keychainSet = origSet
	})
}

func TestReadSharedDBPasswordDisabled(t *testing.T) {
	// reusePassword off -> returns ("", false, nil) without prompting.
	pw, set, err := readSharedDBPassword(reader(""), "src", nil, importOptions{})
	if err != nil || set || pw != "" {
		t.Errorf("got (%q,%v,%v), want empty/false/nil", pw, set, err)
	}
}

func TestReadSharedDBPasswordEncrypts(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "shared-secret")
	selected := []importer.ImportedConnection{{Name: "db1", DB: config.DBConfig{Host: "h", User: "u"}}}
	pw, set, err := readSharedDBPassword(reader(""), "src", selected, importOptions{reusePassword: true})
	if err != nil {
		t.Fatalf("readSharedDBPassword: %v", err)
	}
	if !set {
		t.Fatal("expected password to be set")
	}
	if pw == "" || pw == "shared-secret" {
		t.Errorf("password should be encrypted, got %q", pw)
	}
}

func TestEncryptImportedSecret(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	enc, err := encryptImportedSecret("secret-value")
	if err != nil {
		t.Fatalf("encryptImportedSecret: %v", err)
	}
	if enc == "" || enc == "secret-value" {
		t.Errorf("expected encrypted value, got %q", enc)
	}
}

func TestSaveToKeychain(t *testing.T) {
	origSet := keychainSet
	called := false
	keychainSet = func(string) error { called = true; return nil }
	defer func() { keychainSet = origSet }()
	saveToKeychain("pw")
	if !called {
		t.Error("keychainSet should have been called")
	}
}

func TestRunImportUnknownSource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := RunImport([]string{"--from", "bogus"}); err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Errorf("expected unknown-source error, got %v", err)
	}
}

func TestRunImportMissingFrom(t *testing.T) {
	if err := RunImport(nil); err == nil || !strings.Contains(err.Error(), "--from is required") {
		t.Errorf("expected --from required, got %v", err)
	}
}

func TestRunImportYAMLNoFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// yaml provider's Discover (no --file) returns an error.
	if err := RunImport([]string{"--from", "yaml"}); err == nil {
		t.Error("expected error: yaml requires a file")
	}
}

func TestRunImportYAMLRedash(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	withFakePassword(t, "api-key-123")

	yamlPath := filepath.Join(dir, "conns.yaml")
	yaml := `- name: analytics
  env: production
  redash:
    url: https://redash.example.com
    data_source_id: 4
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}

	if err := RunImport([]string{"--from", "yaml", "--file", yamlPath, "--all"}); err != nil {
		t.Fatalf("RunImport redash: %v", err)
	}

	// Verify the connection was saved with an encrypted (non-plaintext) API key.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	conn := cfg.Find("analytics")
	if conn == nil || conn.Redash == nil {
		t.Fatal("analytics connection not imported")
	}
	if conn.Redash.APIKey == "" || conn.Redash.APIKey == "api-key-123" {
		t.Errorf("API key should be encrypted, got %q", conn.Redash.APIKey)
	}
}

func TestRunImportYAMLDBConnection(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Empty password so promptForDBPassword skips the testConnection path.
	withFakePassword(t, "")

	yamlPath := filepath.Join(dir, "conns.yaml")
	yaml := `- name: localdb
  env: development
  db:
    host: 127.0.0.1
    port: 3306
    user: root
    database: app
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0600); err != nil {
		t.Fatal(err)
	}

	// Provide "y" on stdin for the "add without password?" and mask prompts.
	withStdin(t, "y\ny\n")

	if err := RunImport([]string{"--from", "yaml", "--file", yamlPath, "--all"}); err != nil {
		t.Fatalf("RunImport db: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Find("localdb") == nil {
		t.Error("localdb connection not imported")
	}
}

func TestRunImportYAMLEmpty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	yamlPath := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(yamlPath, []byte("[]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	// Empty list -> "no connections found" message, no error.
	if err := RunImport([]string{"--from", "yaml", "--file", yamlPath, "--all"}); err != nil {
		t.Errorf("RunImport empty: %v", err)
	}
}
