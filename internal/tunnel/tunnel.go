package tunnel

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/atani/mysh/internal/config"
)

type Tunnel struct {
	LocalPort int
	cmd       *exec.Cmd
}

// TunnelInfo is persisted to disk for background tunnels.
type TunnelInfo struct {
	Name       string `json:"name"`
	PID        int    `json:"pid"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func sshArgs(ssh *config.SSHConfig, localPort int, remoteHost string, remotePort int) []string {
	sshPort := ssh.Port
	if sshPort == 0 {
		sshPort = 22
	}

	args := []string{
		"-N", "-L",
		fmt.Sprintf("%d:%s:%d", localPort, remoteHost, remotePort),
		fmt.Sprintf("%s@%s", ssh.User, ssh.Host),
		"-p", strconv.Itoa(sshPort),
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ServerAliveInterval=60",
	}

	if ssh.Key != "" {
		args = append(args, "-i", ssh.Key)
	}

	return args
}

func waitReady(port int) error {
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("tunnel did not become ready within 5s")
}

// Open starts an SSH tunnel as a child process (foreground, closes when parent exits).
func Open(ssh *config.SSHConfig, remoteHost string, remotePort int) (*Tunnel, error) {
	localPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port: %w", err)
	}

	cmd := exec.Command("ssh", sshArgs(ssh, localPort, remoteHost, remotePort)...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting SSH tunnel: %w", err)
	}

	if err := waitReady(localPort); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	return &Tunnel{LocalPort: localPort, cmd: cmd}, nil
}

func (t *Tunnel) Close() {
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_ = t.cmd.Wait()
	}
}

// OpenBackground starts an SSH tunnel as a detached background process and saves its info.
func OpenBackground(name string, ssh *config.SSHConfig, remoteHost string, remotePort int) (*TunnelInfo, error) {
	// Check if already running
	if info, err := LoadInfo(name); err == nil && info != nil {
		if isAlive(info.PID) && portOpen(info.LocalPort) {
			return info, nil
		}
		// Stale, clean up
		_ = RemoveInfo(name)
	}

	localPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port: %w", err)
	}

	cmd := exec.Command("ssh", sshArgs(ssh, localPort, remoteHost, remotePort)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting SSH tunnel: %w", err)
	}

	// Detach: release the child so it survives after mysh exits
	go func() { _ = cmd.Wait() }()

	if err := waitReady(localPort); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	info := &TunnelInfo{
		Name:       name,
		PID:        cmd.Process.Pid,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
	}

	if err := SaveInfo(info); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("saving tunnel info: %w", err)
	}

	return info, nil
}

// StopBackground kills a background tunnel by name.
func StopBackground(name string) error {
	info, err := LoadInfo(name)
	if err != nil {
		return fmt.Errorf("tunnel %q is not running", name)
	}

	if isAlive(info.PID) {
		proc, err := os.FindProcess(info.PID)
		if err == nil {
			_ = proc.Kill()
		}
	}

	return RemoveInfo(name)
}

// FindRunning returns the TunnelInfo for a running background tunnel, or nil.
func FindRunning(name string) *TunnelInfo {
	info, err := LoadInfo(name)
	if err != nil {
		return nil
	}
	if !isAlive(info.PID) || !portOpen(info.LocalPort) {
		_ = RemoveInfo(name)
		return nil
	}
	return info
}

// ListRunning returns all active background tunnels.
func ListRunning() ([]*TunnelInfo, error) {
	dir := config.TunnelsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var result []*TunnelInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		name := e.Name()[:len(e.Name())-5] // strip .json
		if info := FindRunning(name); info != nil {
			result = append(result, info)
		}
	}
	return result, nil
}

func infoPath(name string) string {
	return filepath.Join(config.TunnelsDir(), name+".json")
}

func SaveInfo(info *TunnelInfo) error {
	if err := os.MkdirAll(config.TunnelsDir(), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return os.WriteFile(infoPath(info.Name), data, 0600)
}

func LoadInfo(name string) (*TunnelInfo, error) {
	data, err := os.ReadFile(infoPath(name))
	if err != nil {
		return nil, err
	}
	var info TunnelInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func RemoveInfo(name string) error {
	return os.Remove(infoPath(name))
}

func isAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if process exists without killing it
	return proc.Signal(syscall.Signal(0)) == nil
}

func portOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
