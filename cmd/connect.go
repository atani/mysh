package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/atani/mysh/internal/config"
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

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	return execMySQL(rc.host, rc.port, rc.user, rc.password, rc.database)
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
