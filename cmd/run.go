package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/atani/mysh/internal/config"
)

func RunRun(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mysh run <name> [-e \"SQL\" | <file.sql>]")
	}

	connName := args[0]
	rest := args[1:]

	var sqlExpr string
	var sqlFile string

	if rest[0] == "-e" {
		if len(rest) < 2 {
			return fmt.Errorf("usage: mysh run <name> -e \"SQL\"")
		}
		sqlExpr = rest[1]
	} else {
		sqlFile = rest[0]
		if _, err := os.Stat(sqlFile); err != nil {
			return fmt.Errorf("SQL file not found: %s", sqlFile)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	conn := cfg.Find(connName)
	if conn == nil {
		return fmt.Errorf("connection %q not found", connName)
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

	if sqlExpr != "" {
		mysqlArgs = append(mysqlArgs, "-e", sqlExpr)
	} else {
		mysqlArgs = append(mysqlArgs, "-e", fmt.Sprintf("source %s", sqlFile))
	}

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
