package cmd

import (
	"errors"
	"testing"

	"github.com/atani/mysh/internal/config"
)

var errPingFail = errors.New("ping failed")

func TestTestConnectionNativeSuccess(t *testing.T) {
	setupConfig(t)
	_, mock := withMockDB(t)
	mock.ExpectPing()

	conn := nativeConn("tc")
	if err := testConnection(&conn); err != nil {
		t.Fatalf("testConnection: %v", err)
	}
}

func TestTestConnectionNativeFailure(t *testing.T) {
	setupConfig(t)
	_, mock := withMockDB(t)
	mock.ExpectPing().WillReturnError(errPingFail)

	conn := nativeConn("tc")
	if err := testConnection(&conn); err == nil {
		t.Fatal("expected ping failure")
	}
}

// TestRunAddNativeWithSuccessfulTest drives the full add flow including the
// "Test connection now?" loop, with a mock DB that pings successfully.
func TestRunAddNativeWithSuccessfulTest(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	_, mock := withMockDB(t)
	mock.ExpectPing()

	// "Use SSH?" -> n, "Test connection?" -> y (default).
	withStdin(t, "n\ny\n")

	err := RunAdd([]string{
		"--name", "nt", "--env", "development",
		"--db-host", "127.0.0.1", "--db-port", "3306",
		"--db-user", "root", "--db-name", "app",
		"--driver", "native",
	})
	if err != nil {
		t.Fatalf("RunAdd native test: %v", err)
	}
	cfg, _ := config.Load()
	if cfg.Find("nt") == nil {
		t.Error("connection not added")
	}
}

// TestRunAddTestFailureThenSkip drives the fix-choice loop: the connection test
// fails (mock ping error), then the user chooses to skip fixing.
func TestRunAddTestFailureThenSkip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	_, mock := withMockDB(t)
	mock.ExpectPing().WillReturnError(errPingFail)

	// "Use SSH?" -> n, "Test connection?" -> y, then fix-choice (no SSH):
	// option 4 = skip.
	withStdin(t, "n\ny\n4\n")

	err := RunAdd([]string{
		"--name", "ntf", "--env", "development",
		"--db-host", "127.0.0.1", "--db-port", "3306",
		"--db-user", "root", "--db-name", "app",
		"--driver", "native",
	})
	if err != nil {
		t.Fatalf("RunAdd test-fail-skip: %v", err)
	}
	cfg, _ := config.Load()
	if cfg.Find("ntf") == nil {
		t.Error("connection should still be saved")
	}
}

func TestApplyFix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")

	conn := &config.Connection{
		Name: "f",
		DB:   config.DBConfig{Host: "old", Port: 3306, User: "u", Database: "d"},
		SSH:  &config.SSHConfig{Host: "bastion", Port: 22, User: "deploy"},
	}

	t.Run("db-host", func(t *testing.T) {
		withStdin(t, "")
		// askEdit reads host then port via the supplied reader.
		if err := applyFix(reader("newhost\n5306\n"), conn, "db-host"); err != nil {
			t.Fatalf("applyFix: %v", err)
		}
		if conn.DB.Host != "newhost" || conn.DB.Port != 5306 {
			t.Errorf("db-host not applied: %+v", conn.DB)
		}
	})

	t.Run("db-name", func(t *testing.T) {
		if err := applyFix(reader("newdb\n"), conn, "db-name"); err != nil {
			t.Fatalf("applyFix: %v", err)
		}
		if conn.DB.Database != "newdb" {
			t.Errorf("db-name not applied: %q", conn.DB.Database)
		}
	})

	t.Run("ssh", func(t *testing.T) {
		if err := applyFix(reader("newbastion\n2222\nadmin\n/k\n"), conn, "ssh"); err != nil {
			t.Fatalf("applyFix: %v", err)
		}
		if conn.SSH.Host != "newbastion" || conn.SSH.User != "admin" {
			t.Errorf("ssh not applied: %+v", conn.SSH)
		}
	})

	t.Run("db-auth empty password keeps", func(t *testing.T) {
		withFakePassword(t, "") // empty -> password unchanged
		if err := applyFix(reader("newuser\n"), conn, "db-auth"); err != nil {
			t.Fatalf("applyFix db-auth: %v", err)
		}
		if conn.DB.User != "newuser" {
			t.Errorf("db-auth user not applied: %q", conn.DB.User)
		}
	})
}
