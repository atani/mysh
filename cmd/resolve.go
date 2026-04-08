package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/redash"
	"github.com/atani/mysh/internal/tunnel"
)

type resolvedConn struct {
	host     string
	port     int
	user     string
	password string
	database string
	driver   string // "cli" or "native"
	cleanup  func() // call when done (closes ad-hoc tunnel if any)
}

// isNative returns true if this connection uses the native Go driver.
func (rc *resolvedConn) isNative() bool {
	return rc.driver == config.DriverNative
}

// openDB opens a database connection using the native Go driver.
// AllowOldPasswords is enabled only for native driver connections.
func (rc *resolvedConn) openDB() (*sql.DB, error) {
	return db.Open(rc.host, rc.port, rc.user, rc.password, rc.database, rc.isNative())
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
		driver:   conn.DB.EffectiveDriver(),
		cleanup:  cleanup,
	}, nil
}

// findConnection loads the config and finds the connection by name.
// If name is empty and there is exactly one connection, it returns that one.
func findConnection(name string) (*config.Config, *config.Connection, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	if name == "" {
		switch len(cfg.Connections) {
		case 0:
			return nil, nil, fmt.Errorf("no connections configured. Run `mysh add` to add one")
		case 1:
			return cfg, &cfg.Connections[0], nil
		default:
			return nil, nil, fmt.Errorf("multiple connections exist. Specify a name. Run `mysh list` to see available connections")
		}
	}

	conn := cfg.Find(name)
	if conn == nil {
		return nil, nil, fmt.Errorf("connection %q not found. Run `mysh list` to see available connections", name)
	}
	return cfg, conn, nil
}

// mysqlArgs builds the common mysql command-line arguments for this connection.
// Password is passed via a temporary defaults file to avoid exposure in both
// the process argument list and the environment.
func (rc *resolvedConn) mysqlArgs() []string {
	args := []string{
		"-h", rc.host,
		"-P", strconv.Itoa(rc.port),
		"-u", rc.user,
	}
	if rc.database != "" {
		args = append(args, rc.database)
	}
	return args
}

// writeDefaultsFile creates a temporary MySQL defaults file with the password.
// Returns the file path and a cleanup function. The caller must call cleanup
// when done to remove the temporary file.
func (rc *resolvedConn) writeDefaultsFile() (string, func(), error) {
	if rc.password == "" {
		return "", func() {}, nil
	}

	dir := config.Dir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, fmt.Errorf("creating config directory: %w", err)
	}

	f, err := os.CreateTemp(dir, ".mysql_defaults_tmp_*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp defaults file: %w", err)
	}

	// Quote the password to handle special characters (#, newlines, backslashes)
	escaped := strings.ReplaceAll(rc.password, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	content := fmt.Sprintf("[client]\npassword='%s'\n", escaped)

	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("writing defaults file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", nil, fmt.Errorf("closing defaults file: %w", err)
	}

	path := f.Name()
	cleanup := func() { _ = os.Remove(path) }
	return path, cleanup, nil
}

// mysqlArgsWithPassword returns mysql args with --defaults-extra-file prepended
// if a password is set, and a cleanup function to remove the temp file.
func (rc *resolvedConn) mysqlArgsWithPassword() ([]string, func(), error) {
	defaultsPath, cleanup, err := rc.writeDefaultsFile()
	if err != nil {
		return nil, nil, err
	}

	args := rc.mysqlArgs()
	if defaultsPath != "" {
		// --defaults-extra-file must be the first argument
		args = append([]string{"--defaults-extra-file=" + defaultsPath}, args...)
	}
	return args, cleanup, nil
}

// decryptRedashAPIKey returns the plaintext API key for a Redash connection.
func decryptRedashAPIKey(conn *config.Connection) (string, error) {
	apiKey := conn.Redash.APIKey
	if apiKey == "" {
		return "", fmt.Errorf("redash API key is not configured")
	}

	enc, err := crypto.UnmarshalEncrypted(apiKey)
	if err != nil {
		// Not a valid encrypted payload — treat as plaintext API key.
		// This path is used during testing or manual config editing.
		fmt.Fprintln(os.Stderr, "[mysh] warning: API key is stored unencrypted")
		return apiKey, nil
	}

	masterPass, err := getMasterPassword()
	if err != nil {
		return "", err
	}
	plain, err := crypto.Decrypt(enc, masterPass)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// resolveRedashClient creates a Redash API client for the connection.
func resolveRedashClient(conn *config.Connection) (*redash.Client, error) {
	apiKey, err := decryptRedashAPIKey(conn)
	if err != nil {
		return nil, err
	}
	return redash.NewClient(conn.Redash.URL, apiKey), nil
}
