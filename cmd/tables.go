package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/atani/mysh/internal/config"
)

func RunTables(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mysh tables <name>")
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

	mysqlArgs := []string{
		"-h", rc.host,
		"-P", strconv.Itoa(rc.port),
		"-u", rc.user,
	}

	if rc.password != "" {
		mysqlArgs = append(mysqlArgs, fmt.Sprintf("-p%s", rc.password))
	}

	if rc.database != "" {
		mysqlArgs = append(mysqlArgs, rc.database)
	}

	mysqlArgs = append(mysqlArgs, "-e", "SHOW TABLES")

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
