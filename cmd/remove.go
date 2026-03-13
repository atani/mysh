package cmd

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/config"
)

func RunRemove(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	cfg, conn, err := findConnection(name)
	if err != nil {
		return err
	}

	if err := cfg.Remove(conn.Name); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Connection %q removed.\n", conn.Name)
	return nil
}
