package tunnel

import (
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/atani/mysh/internal/config"
)

// TestHelperProcess is not a real test. It is executed as a subprocess in place
// of the `ssh` binary so the tunnel logic can be exercised without contacting a
// real host. Depending on MYSH_TEST_MODE it either opens the forwarded local
// port (success) or exits immediately (failure).
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	mode := os.Getenv("MYSH_TEST_MODE")
	if mode == "fail" {
		os.Exit(3)
	}

	// Parse the "-L localPort:host:port" argument to discover the local port
	// the parent expects us to listen on.
	args := os.Args
	localPort := 0
	for i, a := range args {
		if a == "-L" && i+1 < len(args) {
			spec := args[i+1]
			parts := strings.SplitN(spec, ":", 2)
			localPort, _ = strconv.Atoi(parts[0])
		}
	}
	if localPort == 0 {
		os.Exit(4)
	}

	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(localPort))
	if err != nil {
		os.Exit(5)
	}
	defer func() { _ = l.Close() }()

	// Stay alive accepting connections until killed by the parent.
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	// Block until killed.
	select {}
}

// stubExecCommand returns an execCommand replacement that runs TestHelperProcess
// instead of the real ssh binary.
func stubExecCommand(mode string) func(name string, args ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "MYSH_TEST_MODE="+mode)
		return cmd
	}
}

func withStubExec(t *testing.T, mode string) {
	t.Helper()
	orig := execCommand
	execCommand = stubExecCommand(mode)
	t.Cleanup(func() { execCommand = orig })
}

func TestOpenSuccess(t *testing.T) {
	withStubExec(t, "ok")
	ssh := &config.SSHConfig{Host: "bastion", User: "deploy"}
	tun, err := Open(ssh, "db.internal", 3306)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if tun.LocalPort <= 0 {
		t.Errorf("expected a local port, got %d", tun.LocalPort)
	}
	if !portOpen(tun.LocalPort) {
		t.Errorf("tunnel local port %d should be open", tun.LocalPort)
	}
	tun.Close()
	// After Close the helper process is killed; the port should free up.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && portOpen(tun.LocalPort) {
		time.Sleep(20 * time.Millisecond)
	}
	if portOpen(tun.LocalPort) {
		t.Errorf("tunnel port %d should be closed after Close()", tun.LocalPort)
	}
}

func TestOpenProcessExitsEarly(t *testing.T) {
	withStubExec(t, "fail")
	ssh := &config.SSHConfig{Host: "bastion", User: "deploy"}
	_, err := Open(ssh, "db.internal", 3306)
	if err == nil {
		t.Fatal("expected error when ssh process exits early")
	}
	if !strings.Contains(err.Error(), "SSH process exited") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCloseNilCmd(t *testing.T) {
	// Close on a zero-value Tunnel must not panic.
	tun := &Tunnel{}
	tun.Close()
}

func TestOpenBackgroundSuccessAndReuse(t *testing.T) {
	setupTempHome(t)
	withStubExec(t, "ok")
	ssh := &config.SSHConfig{Host: "bastion", User: "deploy"}

	info, err := OpenBackground("prod", ssh, "db.internal", 3306)
	if err != nil {
		t.Fatalf("OpenBackground: %v", err)
	}
	defer func() { _ = StopBackground("prod") }()

	if info.Name != "prod" || info.LocalPort <= 0 {
		t.Errorf("unexpected info: %+v", info)
	}
	// Info file must be persisted.
	if _, err := LoadInfo("prod"); err != nil {
		t.Fatalf("info not saved: %v", err)
	}
	// FindRunning should return the live tunnel.
	if got := FindRunning("prod"); got == nil {
		t.Error("FindRunning should find the live background tunnel")
	}
	// ListRunning should include it.
	list, err := ListRunning()
	if err != nil {
		t.Fatalf("ListRunning: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 running tunnel, got %d", len(list))
	}

	// Calling OpenBackground again with the same name should reuse it (same port).
	info2, err := OpenBackground("prod", ssh, "db.internal", 3306)
	if err != nil {
		t.Fatalf("OpenBackground reuse: %v", err)
	}
	if info2.LocalPort != info.LocalPort {
		t.Errorf("reuse should keep same port: got %d, want %d", info2.LocalPort, info.LocalPort)
	}
}

func TestOpenBackgroundFailure(t *testing.T) {
	setupTempHome(t)
	withStubExec(t, "fail")
	ssh := &config.SSHConfig{Host: "bastion", User: "deploy"}
	_, err := OpenBackground("prod", ssh, "db.internal", 3306)
	if err == nil {
		t.Fatal("expected error when ssh exits early")
	}
}

func TestStopBackground(t *testing.T) {
	setupTempHome(t)
	withStubExec(t, "ok")
	ssh := &config.SSHConfig{Host: "bastion", User: "deploy"}

	info, err := OpenBackground("stopme", ssh, "db.internal", 3306)
	if err != nil {
		t.Fatalf("OpenBackground: %v", err)
	}
	if err := StopBackground("stopme"); err != nil {
		t.Fatalf("StopBackground: %v", err)
	}
	// Info file should be gone.
	if _, err := LoadInfo("stopme"); err == nil {
		t.Error("info file should be removed after StopBackground")
	}
	// Port should eventually close.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && portOpen(info.LocalPort) {
		time.Sleep(20 * time.Millisecond)
	}
	if portOpen(info.LocalPort) {
		t.Errorf("port %d should be closed after StopBackground", info.LocalPort)
	}
}

func TestStopBackgroundNotRunning(t *testing.T) {
	setupTempHome(t)
	err := StopBackground("ghost")
	if err == nil {
		t.Error("expected error stopping a non-running tunnel")
	}
}

func TestWaitReadyTimeout(t *testing.T) {
	// A port nobody listens on should time out quickly.
	l, _ := net.Listen("tcp", "127.0.0.1:0")
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	err := waitReady(port, 200*time.Millisecond, nil)
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "did not become ready") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWaitReadySuccess(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()
	port := l.Addr().(*net.TCPAddr).Port

	if err := waitReady(port, 2*time.Second, nil); err != nil {
		t.Errorf("waitReady should succeed on a listening port: %v", err)
	}
}

func TestWaitForCmd(t *testing.T) {
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		// "true" may not exist on some systems; fall back to a no-op.
		t.Skip("/bin/true unavailable")
	}
	ch := waitForCmd(cmd)
	select {
	case err := <-ch:
		if err != nil {
			t.Errorf("waitForCmd: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("waitForCmd did not signal exit")
	}
}
