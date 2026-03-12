package cmd

import (
	"fmt"
	"os"

	"github.com/atani/mysh/internal/config"
)

func RunRemove(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: mysh remove <name>")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if err := cfg.Remove(args[0]); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Connection %q removed.\n", args[0])
	return nil
}
