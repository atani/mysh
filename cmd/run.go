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

func RunRun(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: mysh run <name> <file.sql>")
	}

	connName := args[0]
	sqlFile := args[1]

	if _, err := os.Stat(sqlFile); err != nil {
		return fmt.Errorf("SQL file not found: %s", sqlFile)
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn := cfg.Find(connName)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connName)
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
	}

	mysqlArgs := []string{
		"-h", host,
		"-P", strconv.Itoa(port),
		"-u", conn.DB.User,
	}

	if password != "" {
		mysqlArgs = append(mysqlArgs, fmt.Sprintf("-p%s", password))
	}

	if conn.DB.Database != "" {
		mysqlArgs = append(mysqlArgs, conn.DB.Database)
	}

	mysqlArgs = append(mysqlArgs, "-e", fmt.Sprintf("source %s", sqlFile))

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
