package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/format"
)

func RunTables(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: mysh tables <name> [--format plain|markdown|csv|pdf] [-o <file>]")
	}

	formatStr := ""
	outputFile := ""
	connName := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--format":
			if i+1 < len(args) {
				i++
				formatStr = args[i]
			} else {
				return fmt.Errorf("--format requires a value (plain, markdown, csv, pdf)")
			}
		case "-o", "--output":
			if i+1 < len(args) {
				i++
				outputFile = args[i]
			} else {
				return fmt.Errorf("-o requires a file path")
			}
		default:
			if connName == "" {
				connName = args[i]
			}
		}
	}

	if connName == "" {
		return fmt.Errorf("usage: mysh tables <name>")
	}

	outFmt, err := format.Parse(formatStr)
	if err != nil {
		return err
	}

	if outFmt == format.PDF && outputFile == "" {
		return fmt.Errorf("PDF format requires -o <file> to specify output path")
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

	mysqlArgs = append(mysqlArgs, "-e", "SHOW TABLES")

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if outFmt == format.Plain && outputFile == "" {
		c.Stdout = os.Stdout
		return c.Run()
	}

	var buf bytes.Buffer
	c.Stdout = &buf

	if err := c.Run(); err != nil {
		return err
	}

	return writeOutput(buf.String(), outFmt, outputFile)
}
