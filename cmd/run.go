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
				return fmt.Errorf("--format requires a value (plain, markdown, csv, pdf)")
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

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

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

	if rc.isNative() {
		return runQueryNative(rc, conn, sqlExpr, sqlFile, shouldMask, outFmt, outputFile)
	}
	return runQueryCLI(rc, conn, sqlExpr, sqlFile, shouldMask, outFmt, outputFile)
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

	headers, rows, err := db.Query(dbConn, sqlExpr)
	if err != nil {
		return err
	}

	if headers == nil {
		fmt.Fprintln(os.Stderr, "Query OK")
		return nil
	}

	output := db.FormatTabular(headers, rows)

	if shouldMask {
		masked, colNames := mask.ApplyToOutput(output, conn.Mask.Columns, conn.Mask.Patterns)
		if len(colNames) > 0 {
			fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(colNames, ", "))
			output = masked
		}
	}

	return writeOutput(output, outFmt, outputFile)
}

func runQueryCLI(rc *resolvedConn, conn *config.Connection, sqlExpr, sqlFile string, shouldMask bool, outFmt format.Type, outputFile string) error {
	mysqlArgs := rc.mysqlArgs()

	if sqlExpr != "" {
		mysqlArgs = append(mysqlArgs, "-e", sqlExpr)
	} else {
		mysqlArgs = append(mysqlArgs, "-e", fmt.Sprintf("source %s", sqlFile))
	}

	needCapture := shouldMask || outFmt != format.Plain || outputFile != ""

	c := exec.Command("mysql", mysqlArgs...)
	c.Env = rc.mysqlEnv()
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if !needCapture {
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
