package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/format"
	"github.com/atani/mysh/internal/mask"
)

func RunQuery(args []string) error {
	var connName string
	var sqlExpr string
	var sqlFile string
	forceMask := false
	forceRaw := false
	formatStr := ""
	outputFile := ""

	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mask":
			forceMask = true
		case "--raw":
			forceRaw = true
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
		case "-e":
			if i+1 < len(args) {
				i++
				sqlExpr = args[i]
			} else {
				return fmt.Errorf("usage: mysh run [name] -e \"SQL\"")
			}
		default:
			positional = append(positional, args[i])
		}
	}

	outFmt, err := format.Parse(formatStr)
	if err != nil {
		return err
	}

	if outFmt == format.PDF && outputFile == "" {
		return fmt.Errorf("PDF format requires -o <file> to specify output path")
	}

	switch len(positional) {
	case 0:
	case 1:
		if sqlExpr == "" {
			if _, statErr := os.Stat(positional[0]); statErr == nil {
				sqlFile = positional[0]
			} else {
				connName = positional[0]
			}
		} else {
			connName = positional[0]
		}
	case 2:
		connName = positional[0]
		sqlFile = positional[1]
	default:
		return fmt.Errorf("usage: mysh run [name] [-e \"SQL\" | <file.sql>]")
	}

	if sqlExpr == "" && sqlFile == "" {
		return fmt.Errorf("usage: mysh run [name] [-e \"SQL\" | <file.sql>]")
	}

	if sqlFile != "" {
		if _, err := os.Stat(sqlFile); err != nil {
			return fmt.Errorf("SQL file not found: %s", sqlFile)
		}
	}

	_, conn, err := findConnection(connName)
	if err != nil {
		return err
	}

	// Determine masking
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	shouldMask := conn.ShouldMask(isTTY)
	if forceMask {
		shouldMask = true
	}
	if forceRaw && shouldMask {
		if conn.Env == "production" {
			stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
			if !stdinTTY {
				return fmt.Errorf("--raw on production requires interactive confirmation (TTY)")
			}
			fmt.Fprintf(os.Stderr, "⚠ Raw output requested for production connection %q.\n", conn.Name)
			fmt.Fprint(os.Stderr, "  Masking will be disabled. Continue? [y/N]: ")
			var answer string
			if _, err := fmt.Fscanln(os.Stdin, &answer); err != nil {
				return fmt.Errorf("failed to read confirmation: %w", err)
			}
			if answer != "y" && answer != "Y" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
		}
		shouldMask = false
	}

	// Redash connections don't need SSH tunnels or DB credentials
	if conn.IsRedash() {
		return runQueryRedash(conn, sqlExpr, sqlFile, shouldMask, outFmt, outputFile)
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	if rc.isNative() {
		return runQueryNative(rc, conn, sqlExpr, sqlFile, shouldMask, outFmt, outputFile)
	}
	return runQueryCLI(rc, conn, sqlExpr, sqlFile, shouldMask, outFmt, outputFile)
}

func runQueryRedash(conn *config.Connection, sqlExpr, sqlFile string, shouldMask bool, outFmt format.Type, outputFile string) error {
	if sqlFile != "" {
		data, err := os.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("reading SQL file: %w", err)
		}
		sqlExpr = string(data)
	}

	client, err := resolveRedashClient(conn)
	if err != nil {
		return err
	}
	result, err := client.Query(sqlExpr, conn.Redash.DataSourceID)
	if err != nil {
		return err
	}

	if result.Headers == nil {
		fmt.Fprintln(os.Stderr, "Query OK")
		return nil
	}

	if shouldMask && conn.HasMaskConfig() {
		maskedCols := mask.FindMaskColumns(result.Headers, conn.Mask.Columns, conn.Mask.Patterns)
		if len(maskedCols) > 0 {
			var colNames []string
			for i, h := range result.Headers {
				if maskedCols[i] {
					colNames = append(colNames, h)
				}
			}
			fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(colNames, ", "))
			for _, row := range result.Rows {
				for idx := range row {
					if maskedCols[idx] {
						row[idx] = mask.Value(row[idx])
					}
				}
			}
		}
	}

	return writeOutputStructured(result.Headers, result.Rows, outFmt, outputFile)
}

func runQueryNative(rc *resolvedConn, conn *config.Connection, sqlExpr, sqlFile string, shouldMask bool, outFmt format.Type, outputFile string) error {
	if sqlFile != "" {
		data, err := os.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("reading SQL file: %w", err)
		}
		sqlExpr = string(data)
	}

	dbConn, err := rc.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()

	// Split multi-statement SQL and execute each statement
	stmts := db.SplitStatements(sqlExpr)
	var lastHeaders []string
	var lastRows [][]string

	for _, stmt := range stmts {
		headers, rows, err := db.Query(dbConn, stmt)
		if err != nil {
			return err
		}
		if headers != nil {
			lastHeaders = headers
			lastRows = rows
		}
	}

	if lastHeaders == nil {
		fmt.Fprintln(os.Stderr, "Query OK")
		return nil
	}

	if shouldMask {
		maskedCols := mask.FindMaskColumns(lastHeaders, conn.Mask.Columns, conn.Mask.Patterns)
		if len(maskedCols) > 0 {
			var colNames []string
			for i, h := range lastHeaders {
				if maskedCols[i] {
					colNames = append(colNames, h)
				}
			}
			fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(colNames, ", "))
			for _, row := range lastRows {
				for idx := range row {
					if maskedCols[idx] {
						row[idx] = mask.Value(row[idx])
					}
				}
			}
		}
	}

	return writeOutputStructured(lastHeaders, lastRows, outFmt, outputFile)
}

func runQueryCLI(rc *resolvedConn, conn *config.Connection, sqlExpr, sqlFile string, shouldMask bool, outFmt format.Type, outputFile string) error {
	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return err
	}
	defer cleanup()

	if sqlExpr != "" {
		mysqlArgs = append(mysqlArgs, "-e", sqlExpr)
	} else {
		mysqlArgs = append(mysqlArgs, "-e", fmt.Sprintf("source %s", sqlFile))
	}

	captureOutput := shouldMask || outFmt != format.Plain || outputFile != ""

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if !captureOutput {
		c.Stdout = os.Stdout
		return c.Run()
	}

	var buf bytes.Buffer
	c.Stdout = &buf

	if err := c.Run(); err != nil {
		return err
	}

	output := buf.String()

	if shouldMask {
		masked, colNames := mask.ApplyToOutput(output, conn.Mask.Columns, conn.Mask.Patterns)
		if len(colNames) > 0 {
			fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(colNames, ", "))
			output = masked
		}
	}

	return writeOutput(output, outFmt, outputFile)
}
