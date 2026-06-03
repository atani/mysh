package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/format"
)

// tablesOptions holds the parsed arguments for the tables command.
type tablesOptions struct {
	connName   string
	outFmt     format.Type
	outputFile string
}

// parseTablesArgs parses and validates the tables command arguments without
// touching any connection, so the parsing logic is unit-testable.
func parseTablesArgs(args []string) (tablesOptions, error) {
	var opts tablesOptions
	formatStr := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--format":
			if i+1 < len(args) {
				i++
				formatStr = args[i]
			} else {
				return opts, fmt.Errorf("--format requires a value (plain, markdown, csv, json, pdf)")
			}
		case "-o", "--output":
			if i+1 < len(args) {
				i++
				opts.outputFile = args[i]
			} else {
				return opts, fmt.Errorf("-o requires a file path")
			}
		default:
			if opts.connName == "" {
				opts.connName = args[i]
			} else {
				return opts, fmt.Errorf("unexpected argument %q", args[i])
			}
		}
	}

	outFmt, err := format.Parse(formatStr)
	if err != nil {
		return opts, err
	}
	opts.outFmt = outFmt

	if outFmt == format.PDF && opts.outputFile == "" {
		return opts, fmt.Errorf("PDF format requires -o <file> to specify output path")
	}

	return opts, nil
}

func RunTables(args []string) error {
	opts, err := parseTablesArgs(args)
	if err != nil {
		return err
	}
	connName := opts.connName
	outFmt := opts.outFmt
	outputFile := opts.outputFile

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

	if outFmt == format.Plain && outputFile == "" {
		fmt.Print(db.FormatTabular(headers, rows))
		return nil
	}

	return writeOutputStructured(headers, rows, outFmt, outputFile)
}

func runTablesCLI(rc *resolvedConn, outFmt format.Type, outputFile string) error {
	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return err
	}
	defer cleanup()

	mysqlArgs = append(mysqlArgs, "-e", "SHOW TABLES")

	c := execCommand("mysql", mysqlArgs...)
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
