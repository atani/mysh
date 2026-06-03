package cmd

import (
	"net"
	"os"
	"testing"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/tunnel"
)

func currentPID() int { return os.Getpid() }

func TestResolveConnectionPlain(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	conn := &config.Connection{
		Name: "p",
		DB:   config.DBConfig{Host: "db", Port: 0, User: "u", Database: "d"},
	}
	rc, err := resolveConnection(conn)
	if err != nil {
		t.Fatalf("resolveConnection: %v", err)
	}
	defer rc.cleanup()
	// Port 0 should default to 3306.
	if rc.port != 3306 {
		t.Errorf("port = %d, want 3306", rc.port)
	}
	if rc.host != "db" || rc.user != "u" {
		t.Errorf("unexpected resolved conn: %+v", rc)
	}
}

func TestResolveConnectionDecryptsPassword(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	withFakePassword(t, "")
	enc := encryptForTest(t, "s3cret")
	conn := &config.Connection{
		Name: "p",
		DB:   config.DBConfig{Host: "db", Port: 3306, User: "u", Database: "d", Password: enc},
	}
	rc, err := resolveConnection(conn)
	if err != nil {
		t.Fatalf("resolveConnection: %v", err)
	}
	defer rc.cleanup()
	if rc.password != "s3cret" {
		t.Errorf("password not decrypted, got %q", rc.password)
	}
}

func TestResolveConnectionReusesBackgroundTunnel(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := config.EnsureDir(); err != nil {
		t.Fatal(err)
	}

	// Start a live listener that stands in for a running background tunnel.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()
	port := l.Addr().(*net.TCPAddr).Port

	info := &tunnel.TunnelInfo{
		Name:       "withssh",
		PID:        currentPID(),
		LocalPort:  port,
		RemoteHost: "db.internal",
		RemotePort: 3306,
	}
	if err := tunnel.SaveInfo(info); err != nil {
		t.Fatalf("SaveInfo: %v", err)
	}

	conn := &config.Connection{
		Name: "withssh",
		SSH:  &config.SSHConfig{Host: "bastion", User: "deploy"},
		DB:   config.DBConfig{Host: "db.internal", Port: 3306, User: "u", Database: "d"},
	}
	rc, err := resolveConnection(conn)
	if err != nil {
		t.Fatalf("resolveConnection: %v", err)
	}
	defer rc.cleanup()

	// Should reuse the running tunnel: host rewritten to localhost, port = listener port.
	if rc.host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", rc.host)
	}
	if rc.port != port {
		t.Errorf("port = %d, want %d (reused tunnel)", rc.port, port)
	}

	// Sanity: the info file path is where we expect.
	if _, err := tunnel.LoadInfo("withssh"); err != nil {
		t.Errorf("tunnel info missing: %v", err)
	}
}

// --- CLI execution error paths (mysql binary not invoked successfully) ---

func TestRunQueryCLIMissingBinary(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	// CLI driver, capture path forces exec of "mysql". With PATH cleared the
	// command fails, exercising runQueryCLI's error handling.
	t.Setenv("PATH", "")
	err := RunQuery([]string{"cli", "-e", "SELECT 1", "--format", "csv"})
	if err == nil {
		t.Error("expected error when mysql binary is unavailable")
	}
}

func TestRunTablesCLIMissingBinary(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	t.Setenv("PATH", "")
	if err := RunTables([]string{"cli", "--format", "csv"}); err == nil {
		t.Error("expected error when mysql binary is unavailable")
	}
}

func TestRunSliceCLIMissingBinary(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	t.Setenv("PATH", "")
	if err := RunSlice([]string{"cli", "users", "--where", "id > 0"}); err == nil {
		t.Error("expected error when mysql binary is unavailable")
	}
}

func TestRunConnectCLIMissingBinary(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	t.Setenv("PATH", "")
	if err := RunConnect([]string{"cli"}); err == nil {
		t.Error("expected error when neither mycli nor mysql is available")
	}
}

func TestRunPingCLIMissingBinary(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	t.Setenv("PATH", "")
	if err := RunPing([]string{"cli"}); err == nil {
		t.Error("expected error when mysql binary is unavailable")
	}
}
