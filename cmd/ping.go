package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/atani/mysh/internal/config"
)

func RunPing(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mysh ping <name>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn := cfg.Find(args[0])
	if conn == nil {
		return fmt.Errorf("connection %q not found", args[0])
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	start := time.Now()

	mysqlArgs := rc.mysqlArgs()
	mysqlArgs = append(mysqlArgs, "-e", "SELECT 1")

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdout = nil
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Connection %q: FAILED (%v)\n", args[0], err)
		return err
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "Connection %q: OK (%s)\n", args[0], elapsed.Round(time.Millisecond))
	return nil
}
