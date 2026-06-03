package cmd

import (
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
)

func TestRunAddRedashInteractive(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "interactive-key") // supplies the API key prompt
	// Only --redash-url given. Prompts: data source id, env (1=production),
	// mask (Enter -> default). Name comes from --name.
	withStdin(t, "7\n1\n\n")

	err := RunAdd([]string{"--name", "ri", "--redash-url", "https://redash.example.com"})
	if err != nil {
		t.Fatalf("RunAdd redash interactive: %v", err)
	}

	cfg, _ := config.Load()
	conn := cfg.Find("ri")
	if conn == nil || conn.Redash == nil {
		t.Fatal("redash connection not added")
	}
	if conn.Redash.DataSourceID != 7 {
		t.Errorf("data source id = %d, want 7", conn.Redash.DataSourceID)
	}
	if conn.Env != "production" {
		t.Errorf("env = %q, want production", conn.Env)
	}
	if conn.Mask == nil {
		t.Error("default mask should be applied")
	}
}

func TestRunAddRedashDuplicate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "k")
	if err := config.Save(&config.Config{Connections: []config.Connection{
		{Name: "dupr", Redash: &config.RedashConfig{URL: "https://r"}},
	}}); err != nil {
		t.Fatal(err)
	}
	// --redash-url + existing name -> ErrConnExists before any prompt.
	err := RunAdd([]string{
		"--name", "dupr", "--redash-url", "https://r", "--redash-key", "k",
		"--redash-datasource", "1", "--env", "production", "--mask", "email",
	})
	if err == nil || !strings.Contains(err.Error(), "dupr") {
		t.Errorf("expected duplicate error, got %v", err)
	}
}

func TestRunAddParseError(t *testing.T) {
	if err := RunAdd([]string{"--unknown"}); err == nil {
		t.Error("expected parse error")
	}
}

func TestRunAddInvalidEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	withStdin(t, "")
	// --env bad with enough flags to skip prompts up to the env validation.
	err := RunAdd([]string{
		"--name", "x", "--db-host", "h", "--db-port", "3306",
		"--db-user", "u", "--db-name", "d", "--env", "bogus",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid environment") {
		t.Errorf("expected invalid environment error, got %v", err)
	}
}

func TestRunAddInvalidDriver(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	withStdin(t, "")
	err := RunAdd([]string{
		"--name", "x", "--db-host", "h", "--db-port", "3306",
		"--db-user", "u", "--db-name", "d", "--env", "development",
		"--driver", "bogus",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid driver") {
		t.Errorf("expected invalid driver error, got %v", err)
	}
}

func TestRunAddFullFlagsNoTest(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "") // empty DB password -> no encryption needed
	// "Use SSH? [y/N]" -> n, then "Test connection now? [Y/n]" -> n.
	withStdin(t, "n\nn\n")

	err := RunAdd([]string{
		"--name", "prod", "--env", "development",
		"--db-host", "127.0.0.1", "--db-port", "3306",
		"--db-user", "root", "--db-name", "app",
		"--driver", "cli",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	conn := cfg.Find("prod")
	if conn == nil {
		t.Fatal("connection not added")
	}
	if conn.DB.Host != "127.0.0.1" || conn.DB.User != "root" || conn.DB.Driver != config.DriverCLI {
		t.Errorf("unexpected connection: %+v", conn.DB)
	}
}

func TestRunAddWithSSHAndMask(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	withStdin(t, "n\n") // skip connection test

	err := RunAdd([]string{
		"--name", "withssh", "--env", "staging",
		"--db-host", "db.internal", "--db-port", "3306",
		"--db-user", "app", "--db-name", "main",
		"--ssh-host", "bastion", "--ssh-port", "22", "--ssh-user", "deploy", "--ssh-key", "/key",
		"--mask", "email,*token*",
		"--driver", "native",
	})
	if err != nil {
		t.Fatalf("RunAdd ssh: %v", err)
	}

	cfg, _ := config.Load()
	conn := cfg.Find("withssh")
	if conn == nil || conn.SSH == nil {
		t.Fatal("ssh connection not added")
	}
	if conn.SSH.Host != "bastion" || conn.SSH.User != "deploy" {
		t.Errorf("ssh config wrong: %+v", conn.SSH)
	}
	if conn.Mask == nil || len(conn.Mask.Columns) != 1 || len(conn.Mask.Patterns) != 1 {
		t.Errorf("mask config wrong: %+v", conn.Mask)
	}
}

func TestRunAddDuplicateName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	// First add: "Use SSH? [y/N]" -> n, "Test connection? [Y/n]" -> n.
	// Second add: "Use SSH? [y/N]" -> n, then it errors on the duplicate name.
	withStdin(t, "n\nn\nn\n")

	args := []string{
		"--name", "dup", "--env", "development",
		"--db-host", "h", "--db-port", "3306", "--db-user", "u", "--db-name", "d", "--driver", "cli",
	}
	if err := RunAdd(args); err != nil {
		t.Fatalf("first add: %v", err)
	}
	// The duplicate-name error message is localized; assert it references the name.
	if err := RunAdd(args); err == nil || !strings.Contains(err.Error(), "dup") {
		t.Errorf("expected already-exists error for %q, got %v", "dup", err)
	}
}

func TestRunAddRedash(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "redash-key")
	withStdin(t, "")

	err := RunAdd([]string{
		"--name", "ra", "--env", "production",
		"--redash-url", "https://redash.example.com",
		"--redash-key", "redash-key",
		"--redash-datasource", "5",
		"--mask", "email",
	})
	if err != nil {
		t.Fatalf("RunAdd redash: %v", err)
	}

	cfg, _ := config.Load()
	conn := cfg.Find("ra")
	if conn == nil || conn.Redash == nil {
		t.Fatal("redash connection not added")
	}
	if conn.Redash.URL != "https://redash.example.com" || conn.Redash.DataSourceID != 5 {
		t.Errorf("redash config wrong: %+v", conn.Redash)
	}
	if conn.Redash.APIKey == "" || conn.Redash.APIKey == "redash-key" {
		t.Errorf("API key should be encrypted: %q", conn.Redash.APIKey)
	}
}

func TestRunAddWithPasswordEncrypts(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "dbsecret")
	withStdin(t, "n\nn\n") // "Use SSH?" -> n, "Test connection?" -> n

	err := RunAdd([]string{
		"--name", "secured", "--env", "development",
		"--db-host", "h", "--db-port", "3306", "--db-user", "u", "--db-name", "d", "--driver", "cli",
	})
	if err != nil {
		t.Fatalf("RunAdd: %v", err)
	}

	cfg, _ := config.Load()
	conn := cfg.Find("secured")
	if conn == nil {
		t.Fatal("connection not added")
	}
	if conn.DB.Password == "" || conn.DB.Password == "dbsecret" {
		t.Errorf("password should be encrypted, got %q", conn.DB.Password)
	}
}
