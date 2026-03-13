package cmd

import (
	"fmt"
	"os"
	"os/exec"

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

	client := "mycli"
	if _, err := exec.LookPath("mycli"); err != nil {
		client = "mysql"
		if _, err := exec.LookPath("mysql"); err != nil {
			return fmt.Errorf("neither mycli nor mysql found in PATH")
		}
	}

	c := exec.Command(client, rc.mysqlArgs()...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
