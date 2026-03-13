package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/tunnel"
)

type resolvedConn struct {
	host     string
	port     int
	user     string
	password string
	database string
	cleanup  func() // call when done (closes ad-hoc tunnel if any)
}

// resolveConnection decrypts the password and sets up SSH tunnel if needed.
// It reuses a background tunnel when available, otherwise opens an ad-hoc one.
func resolveConnection(conn *config.Connection) (*resolvedConn, error) {
	host := conn.DB.Host
	port := conn.DB.Port
	if port == 0 {
		port = 3306
	}

	var password string
	if conn.DB.Password != "" {
		masterPass, err := getMasterPassword()
		if err != nil {
			return nil, err
		}
		enc, err := crypto.UnmarshalEncrypted(conn.DB.Password)
		if err != nil {
			return nil, fmt.Errorf("reading encrypted password: %w", err)
		}
		plain, err := crypto.Decrypt(enc, masterPass)
		if err != nil {
			return nil, err
		}
		password = string(plain)
	}

	cleanup := func() {}

	if conn.SSH != nil {
		// Try reusing a background tunnel first
		if info := tunnel.FindRunning(conn.Name); info != nil {
			fmt.Fprintf(os.Stderr, "Reusing background tunnel %q (localhost:%d)\n", conn.Name, info.LocalPort)
			host = "127.0.0.1"
			port = info.LocalPort
		} else {
			// Open an ad-hoc tunnel
			fmt.Fprintf(os.Stderr, "Opening SSH tunnel via %s@%s...\n", conn.SSH.User, conn.SSH.Host)
			tun, err := tunnel.Open(conn.SSH, host, port)
			if err != nil {
				return nil, fmt.Errorf("SSH tunnel: %w", err)
			}
			host = "127.0.0.1"
			port = tun.LocalPort
			cleanup = tun.Close
			fmt.Fprintf(os.Stderr, "Tunnel ready on port %d\n", port)
		}
	}

	return &resolvedConn{
		host:     host,
		port:     port,
		user:     conn.DB.User,
		password: password,
		database: conn.DB.Database,
		cleanup:  cleanup,
	}, nil
}

// mysqlArgs builds the common mysql command-line arguments for this connection.
func (rc *resolvedConn) mysqlArgs() []string {
	args := []string{
		"-h", rc.host,
		"-P", strconv.Itoa(rc.port),
		"-u", rc.user,
	}
	if rc.password != "" {
		args = append(args, fmt.Sprintf("-p%s", rc.password))
	}
	if rc.database != "" {
		args = append(args, rc.database)
	}
	return args
}
