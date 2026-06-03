package cmd

import (
	"errors"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
)

// initMaster ensures the config dir exists and initializes the master password.
func initMaster(t *testing.T, pw string) {
	t.Helper()
	if err := config.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if err := crypto.InitMasterPassword([]byte(pw)); err != nil {
		t.Fatalf("InitMasterPassword: %v", err)
	}
}

// withKeychainAndReader installs stub keychain accessors and a fixed password
// reader, and clears MYSH_MASTER_PASSWORD so getMasterPassword falls through to
// the interactive setup/verify branches.
func withMasterPasswordStubs(t *testing.T, pw string) {
	t.Helper()
	origRead := readPassword
	origGet := keychainGet
	origSet := keychainSet
	readPassword = func() (string, error) { return pw, nil }
	keychainGet = func() (string, error) { return "", errors.New("not found") }
	keychainSet = func(string) error { return nil }
	t.Setenv("MYSH_MASTER_PASSWORD", "")
	t.Cleanup(func() {
		readPassword = origRead
		keychainGet = origGet
		keychainSet = origSet
	})
}

func TestGetMasterPasswordFirstTimeSetup(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withMasterPasswordStubs(t, "new-master")

	// No master password initialized yet -> setup branch. The reader returns the
	// same value for both the prompt and the confirmation, so they match.
	got, err := getMasterPassword()
	if err != nil {
		t.Fatalf("getMasterPassword setup: %v", err)
	}
	if string(got) != "new-master" {
		t.Errorf("got %q, want new-master", got)
	}
	// The check file should now exist so the password verifies.
	if !crypto.MasterPasswordInitialized() {
		t.Error("master password should be initialized after setup")
	}
}

func TestGetMasterPasswordEmptyRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withMasterPasswordStubs(t, "") // empty password during setup -> error
	if _, err := getMasterPassword(); err == nil {
		t.Error("expected error for empty master password")
	}
}

func TestGetMasterPasswordVerifyExisting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	// Initialize a master password directly, then verify via the existing-password
	// branch (MasterPasswordInitialized() == true).
	initMaster(t, "existing-pw")
	withMasterPasswordStubs(t, "existing-pw")

	got, err := getMasterPassword()
	if err != nil {
		t.Fatalf("getMasterPassword verify: %v", err)
	}
	if string(got) != "existing-pw" {
		t.Errorf("got %q, want existing-pw", got)
	}
}

func TestGetMasterPasswordVerifyWrong(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	initMaster(t, "right-pw")
	withMasterPasswordStubs(t, "wrong-pw")

	if _, err := getMasterPassword(); err == nil {
		t.Error("expected verification failure for wrong password")
	}
}

func TestGetMasterPasswordFromEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	initMaster(t, "env-pw")
	origGet := keychainGet
	keychainGet = func() (string, error) { return "", errors.New("not found") }
	t.Cleanup(func() { keychainGet = origGet })
	t.Setenv("MYSH_MASTER_PASSWORD", "env-pw")

	got, err := getMasterPassword()
	if err != nil {
		t.Fatalf("getMasterPassword env: %v", err)
	}
	if string(got) != "env-pw" {
		t.Errorf("got %q, want env-pw", got)
	}
}

func TestGetMasterPasswordFromKeychain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	initMaster(t, "kc-pw")
	origGet := keychainGet
	keychainGet = func() (string, error) { return "kc-pw", nil }
	t.Cleanup(func() { keychainGet = origGet })
	t.Setenv("MYSH_MASTER_PASSWORD", "")

	got, err := getMasterPassword()
	if err != nil {
		t.Fatalf("getMasterPassword keychain: %v", err)
	}
	if string(got) != "kc-pw" {
		t.Errorf("got %q, want kc-pw", got)
	}
}
