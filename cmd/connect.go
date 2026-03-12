package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/tunnel"
)

func RunConnect(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: mysh connect <name>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn := cfg.Find(args[0])
	if conn == nil {
		return fmt.Errorf("connection %q not found. Run `mysh list` to see available connections", args[0])
	}

	host := conn.DB.Host
	port := conn.DB.Port
	if port == 0 {
		port = 3306
	}

	var password string
	if conn.DB.Password != "" {
		masterPass, err := getMasterPassword()
		if err != nil {
			return err
		}
		enc, err := crypto.UnmarshalEncrypted(conn.DB.Password)
		if err != nil {
			return fmt.Errorf("reading encrypted password: %w", err)
		}
		plain, err := crypto.Decrypt(enc, masterPass)
		if err != nil {
			return err
		}
		password = string(plain)
	}

	var tun *tunnel.Tunnel
	if conn.SSH != nil {
		fmt.Fprintf(os.Stderr, "Opening SSH tunnel via %s@%s...\n", conn.SSH.User, conn.SSH.Host)
		tun, err = tunnel.Open(conn.SSH, host, port)
		if err != nil {
			return fmt.Errorf("SSH tunnel: %w", err)
		}
		defer tun.Close()
		host = "127.0.0.1"
		port = tun.LocalPort
		fmt.Fprintf(os.Stderr, "Tunnel ready on port %d\n", port)
	}

	return execMySQL(host, port, conn.DB.User, password, conn.DB.Database)
}

func execMySQL(host string, port int, user, password, database string) error {
	client := "mycli"
	if _, err := exec.LookPath("mycli"); err != nil {
		client = "mysql"
		if _, err := exec.LookPath("mysql"); err != nil {
			return fmt.Errorf("neither mycli nor mysql found in PATH")
		}
	}

	args := []string{
		"-h", host,
		"-P", strconv.Itoa(port),
		"-u", user,
	}

	if password != "" {
		args = append(args, fmt.Sprintf("-p%s", password))
	}

	if database != "" {
		args = append(args, database)
	}

	c := exec.Command(client, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
