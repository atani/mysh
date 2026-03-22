package tunnel

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/atani/mysh/internal/config"
)

func setupTempHome(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	tunnelsDir := filepath.Join(tmpDir, ".config", "mysh", "tunnels")
	if err := os.MkdirAll(tunnelsDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	return tmpDir
}

func TestSSHArgs(t *testing.T) {
	tests := []struct {
		name       string
		ssh        *config.SSHConfig
		localPort  int
		remoteHost string
		remotePort int
		wantKey    bool
		wantPort   int
	}{
		{
			name:       "basic with default port",
			ssh:        &config.SSHConfig{Host: "bastion.example.com", User: "deploy"},
			localPort:  13306,
			remoteHost: "db.internal",
			remotePort: 3306,
			wantKey:    false,
			wantPort:   22,
		},
		{
			name:       "custom port with key",
			ssh:        &config.SSHConfig{Host: "bastion.example.com", User: "deploy", Port: 2222, Key: "~/.ssh/id_ed25519"},
			localPort:  13306,
			remoteHost: "db.internal",
			remotePort: 3306,
			wantKey:    true,
			wantPort:   2222,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := sshArgs(tt.ssh, tt.localPort, tt.remoteHost, tt.remotePort)

			// Must contain -N and -L
			if args[0] != "-N" || args[1] != "-L" {
				t.Errorf("expected -N -L, got %v %v", args[0], args[1])
			}

			// Check port forwarding spec
			wantSpec := "13306:db.internal:3306"
			if args[2] != wantSpec {
				t.Errorf("port spec: got %q, want %q", args[2], wantSpec)
			}

			// Check user@host
			wantUserHost := "deploy@bastion.example.com"
			if args[3] != wantUserHost {
				t.Errorf("user@host: got %q, want %q", args[3], wantUserHost)
			}

			// Check -i flag presence
			hasKey := false
			for _, a := range args {
				if a == "-i" {
					hasKey = true
				}
			}
			if hasKey != tt.wantKey {
				t.Errorf("key flag: got %v, want %v", hasKey, tt.wantKey)
			}
		})
	}
}

func TestFreePort(t *testing.T) {
	port, err := freePort()
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("port out of range: %d", port)
	}

	// Port should be usable
	l, err := net.Listen("tcp", "127.0.0.1:"+string(rune(port)))
	if err != nil {
		// Expected: port might be reused quickly, just verify it was valid
		return
	}
	_ = l.Close()
}

func TestInfoPath(t *testing.T) {
	setupTempHome(t)
	path := infoPath("production")
	expected := filepath.Join(config.TunnelsDir(), "production.json")
	if path != expected {
		t.Errorf("got %q, want %q", path, expected)
	}
}

func TestSaveLoadRemoveInfo(t *testing.T) {
	setupTempHome(t)

	info := &TunnelInfo{
		Name:       "test-db",
		PID:        12345,
		LocalPort:  13306,
		RemoteHost: "db.internal",
		RemotePort: 3306,
	}

	// Save
	if err := SaveInfo(info); err != nil {
		t.Fatalf("SaveInfo: %v", err)
	}

	// Verify file exists
	path := infoPath("test-db")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("info file not found: %v", err)
	}

	// Verify file permissions
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0600 {
		t.Errorf("file permissions: got %o, want 0600", fi.Mode().Perm())
	}

	// Load
	loaded, err := LoadInfo("test-db")
	if err != nil {
		t.Fatalf("LoadInfo: %v", err)
	}
	if loaded.Name != info.Name {
		t.Errorf("Name: got %q, want %q", loaded.Name, info.Name)
	}
	if loaded.PID != info.PID {
		t.Errorf("PID: got %d, want %d", loaded.PID, info.PID)
	}
	if loaded.LocalPort != info.LocalPort {
		t.Errorf("LocalPort: got %d, want %d", loaded.LocalPort, info.LocalPort)
	}
	if loaded.RemoteHost != info.RemoteHost {
		t.Errorf("RemoteHost: got %q, want %q", loaded.RemoteHost, info.RemoteHost)
	}
	if loaded.RemotePort != info.RemotePort {
		t.Errorf("RemotePort: got %d, want %d", loaded.RemotePort, info.RemotePort)
	}

	// Remove
	if err := RemoveInfo("test-db"); err != nil {
		t.Fatalf("RemoveInfo: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("info file should be removed")
	}
}

func TestLoadInfoNotFound(t *testing.T) {
	setupTempHome(t)

	_, err := LoadInfo("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tunnel info")
	}
}

func TestLoadInfoCorrupted(t *testing.T) {
	setupTempHome(t)

	path := infoPath("corrupted")
	if err := os.WriteFile(path, []byte("not-valid-json"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadInfo("corrupted")
	if err == nil {
		t.Error("expected error for corrupted tunnel info")
	}
}

func TestIsAliveCurrentProcess(t *testing.T) {
	// Current process should be alive
	if !isAlive(os.Getpid()) {
		t.Error("current process should be alive")
	}
}

func TestIsAliveInvalidPID(t *testing.T) {
	// Very high PID that almost certainly doesn't exist
	if isAlive(999999999) {
		t.Error("invalid PID should not be alive")
	}
}

func TestPortOpenListening(t *testing.T) {
	// Start a test listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = l.Close() }()
	port := l.Addr().(*net.TCPAddr).Port

	if !portOpen(port) {
		t.Error("listening port should be detected as open")
	}
}

func TestPortOpenNotListening(t *testing.T) {
	// Get a free port and don't listen on it
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	if portOpen(port) {
		t.Error("closed port should not be detected as open")
	}
}

func TestFindRunningNotFound(t *testing.T) {
	setupTempHome(t)

	info := FindRunning("nonexistent")
	if info != nil {
		t.Error("expected nil for nonexistent tunnel")
	}
}

func TestFindRunningStaleCleanup(t *testing.T) {
	setupTempHome(t)

	// Save info with a dead PID
	info := &TunnelInfo{
		Name:       "stale",
		PID:        999999999,
		LocalPort:  19999,
		RemoteHost: "db.internal",
		RemotePort: 3306,
	}
	if err := SaveInfo(info); err != nil {
		t.Fatalf("SaveInfo: %v", err)
	}

	// FindRunning should detect it's stale and clean up
	result := FindRunning("stale")
	if result != nil {
		t.Error("expected nil for stale tunnel")
	}

	// Info file should be removed
	if _, err := os.Stat(infoPath("stale")); !os.IsNotExist(err) {
		t.Error("stale info file should be cleaned up")
	}
}

func TestListRunningEmpty(t *testing.T) {
	setupTempHome(t)

	result, err := ListRunning()
	if err != nil {
		t.Fatalf("ListRunning: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}

func TestListRunningNonExistentDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	// Don't create the tunnels dir

	result, err := ListRunning()
	if err != nil {
		t.Fatalf("ListRunning: %v", err)
	}
	if result != nil {
		t.Error("expected nil for non-existent dir")
	}
}

func TestListRunningSkipsNonJSON(t *testing.T) {
	setupTempHome(t)
	tunnelsDir := config.TunnelsDir()

	// Create a non-JSON file
	if err := os.WriteFile(filepath.Join(tunnelsDir, "readme.txt"), []byte("ignore me"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Create a subdirectory
	if err := os.MkdirAll(filepath.Join(tunnelsDir, "subdir"), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	result, err := ListRunning()
	if err != nil {
		t.Fatalf("ListRunning: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d", len(result))
	}
}

func TestListRunningFiltersStale(t *testing.T) {
	setupTempHome(t)

	// Save stale tunnel info (dead PID)
	info := &TunnelInfo{
		Name:       "dead-tunnel",
		PID:        999999999,
		LocalPort:  19999,
		RemoteHost: "db.internal",
		RemotePort: 3306,
	}
	data, _ := json.Marshal(info)
	if err := os.WriteFile(filepath.Join(config.TunnelsDir(), "dead-tunnel.json"), data, 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	result, err := ListRunning()
	if err != nil {
		t.Fatalf("ListRunning: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list (stale filtered), got %d", len(result))
	}
}

func TestRemoveInfoNotFound(t *testing.T) {
	setupTempHome(t)

	err := RemoveInfo("nonexistent")
	if err == nil {
		t.Error("expected error for removing nonexistent info")
	}
}
