package tunnel

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"

	"github.com/atani/mysh/internal/config"
)

type Tunnel struct {
	LocalPort int
	cmd       *exec.Cmd
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func Open(ssh *config.SSHConfig, remoteHost string, remotePort int) (*Tunnel, error) {
	localPort, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("finding free port: %w", err)
	}

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

	cmd := exec.Command("ssh", args...)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting SSH tunnel: %w", err)
	}

	// wait for tunnel to be ready
	for i := 0; i < 50; i++ {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return &Tunnel{LocalPort: localPort, cmd: cmd}, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	cmd.Process.Kill()
	return nil, fmt.Errorf("SSH tunnel did not become ready within 5s")
}

func (t *Tunnel) Close() {
	if t.cmd != nil && t.cmd.Process != nil {
		t.cmd.Process.Kill()
		t.cmd.Wait()
	}
}
