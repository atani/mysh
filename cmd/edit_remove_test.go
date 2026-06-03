package cmd

import (
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
)

// encryptForTest encrypts plaintext using the test master password ("master-pw"
// from withFakePassword) and returns the marshaled encrypted string.
func encryptForTest(t *testing.T, plaintext string) string {
	t.Helper()
	enc, err := crypto.Encrypt([]byte(plaintext), []byte("master-pw"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	s, err := crypto.MarshalEncrypted(enc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return s
}

func TestRunEditKeepsValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "") // empty -> keep existing password
	conn := config.Connection{
		Name: "e1",
		Env:  "development",
		DB:   config.DBConfig{Host: "h", Port: 3306, User: "u", Database: "d", Driver: config.DriverCLI},
	}
	if err := config.Save(&config.Config{Connections: []config.Connection{conn}}); err != nil {
		t.Fatal(err)
	}

	// "Use SSH? [y/N]" -> n (default), then keep host/port/user/db (blank lines),
	// then password (empty via readPassword), then driver keep (blank),
	// then env keep (blank). Development env skips the mask prompt.
	withStdin(t, "n\n\n\n\n\n\n\n")

	if err := RunEdit([]string{"e1"}); err != nil {
		t.Fatalf("RunEdit: %v", err)
	}

	cfg, _ := config.Load()
	got := cfg.Find("e1")
	if got == nil || got.DB.Host != "h" || got.DB.User != "u" {
		t.Errorf("values not kept: %+v", got)
	}
}

func TestRunEditAddsSSH(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	conn := config.Connection{
		Name: "e2",
		Env:  "development",
		DB:   config.DBConfig{Host: "h", Port: 3306, User: "u", Database: "d", Driver: config.DriverCLI},
	}
	if err := config.Save(&config.Config{Connections: []config.Connection{conn}}); err != nil {
		t.Fatal(err)
	}

	// "Use SSH?" -> y, ssh host/port/user/key, then DB host/port/user/db (keep),
	// password (empty), driver (keep), env (keep).
	withStdin(t, "y\nbastion\n22\ndeploy\n/key\n\n\n\n\n\n\n")

	if err := RunEdit([]string{"e2"}); err != nil {
		t.Fatalf("RunEdit add ssh: %v", err)
	}

	cfg, _ := config.Load()
	got := cfg.Find("e2")
	if got == nil || got.SSH == nil || got.SSH.Host != "bastion" || got.SSH.User != "deploy" {
		t.Errorf("ssh not added: %+v", got.SSH)
	}
}

func TestRunEditClearsPassword(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "clear") // "clear" sentinel removes the password
	conn := config.Connection{
		Name: "e3",
		Env:  "development",
		DB:   config.DBConfig{Host: "h", Port: 3306, User: "u", Database: "d", Password: "ENCRYPTED", Driver: config.DriverCLI},
	}
	if err := config.Save(&config.Config{Connections: []config.Connection{conn}}); err != nil {
		t.Fatal(err)
	}

	withStdin(t, "n\n\n\n\n\n\n\n")

	if err := RunEdit([]string{"e3"}); err != nil {
		t.Fatalf("RunEdit clear: %v", err)
	}
	cfg, _ := config.Load()
	if got := cfg.Find("e3"); got == nil || got.DB.Password != "" {
		t.Errorf("password should be cleared, got %q", got.DB.Password)
	}
}

func TestRunRemoveConfirmed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Save(&config.Config{Connections: []config.Connection{dbConn("r1")}}); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "y\n")
	if err := RunRemove([]string{"r1"}); err != nil {
		t.Fatalf("RunRemove: %v", err)
	}
	cfg, _ := config.Load()
	if cfg.Find("r1") != nil {
		t.Error("connection should be removed")
	}
}

func TestRunRemoveAborted(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.Save(&config.Config{Connections: []config.Connection{dbConn("r2")}}); err != nil {
		t.Fatal(err)
	}
	withStdin(t, "n\n")
	if err := RunRemove([]string{"r2"}); err != nil {
		t.Fatalf("RunRemove abort: %v", err)
	}
	cfg, _ := config.Load()
	if cfg.Find("r2") == nil {
		t.Error("connection should NOT be removed when aborted")
	}
}

func TestRunConnectNativeREPL(t *testing.T) {
	setupConfig(t, nativeConn("c1"))
	_, mock := withMockDB(t)
	mock.ExpectPing()
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"n"}).AddRow("1"))
	// Drive the REPL: one query then quit.
	withStdin(t, "SELECT 1;\nquit\n")

	if err := RunConnect([]string{"c1"}); err != nil {
		t.Fatalf("RunConnect native: %v", err)
	}
}

func TestDecryptRedashAPIKeyEncrypted(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	// Encrypt a key with the master password, then decrypt it back.
	enc := encryptForTest(t, "real-api-key")
	conn := &config.Connection{Redash: &config.RedashConfig{URL: "https://r", APIKey: enc}}
	got, err := decryptRedashAPIKey(conn)
	if err != nil {
		t.Fatalf("decryptRedashAPIKey: %v", err)
	}
	if got != "real-api-key" {
		t.Errorf("got %q, want real-api-key", got)
	}
	if !strings.HasPrefix(got, "real") {
		t.Errorf("unexpected: %q", got)
	}
}
