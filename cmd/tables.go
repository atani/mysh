package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/format"
)

func RunTables(args []string) error {
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
				return fmt.Errorf("--format requires a value (plain, markdown, csv, json, pdf)")
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
			} else {
				return fmt.Errorf("unexpected argument %q", args[i])
			}
		}
	}

	outFmt, err := format.Parse(formatStr)
	if err != nil {
		return err
	}

	if outFmt == format.PDF && outputFile == "" {
		return fmt.Errorf("PDF format requires -o <file> to specify output path")
	}

	_, conn, err := findConnection(connName)
	if err != nil {
		return err
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	if rc.isNative() {
		return runTablesNative(rc, outFmt, outputFile)
	}
	return runTablesCLI(rc, outFmt, outputFile)
}

func runTablesNative(rc *resolvedConn, outFmt format.Type, outputFile string) error {
	dbConn, err := rc.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()

	headers, rows, err := db.Query(dbConn, "SHOW TABLES")
	if err != nil {
		return err
	}

	output := db.FormatTabular(headers, rows)

	if outFmt == format.Plain && outputFile == "" {
		fmt.Print(output)
		return nil
	}

	return writeOutput(output, outFmt, outputFile)
}

func runTablesCLI(rc *resolvedConn, outFmt format.Type, outputFile string) error {
	mysqlArgs := rc.mysqlArgs()
	mysqlArgs = append(mysqlArgs, "-e", "SHOW TABLES")

	c := exec.Command("mysql", mysqlArgs...)
	c.Env = rc.mysqlEnv()
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
