package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/mask"
	"github.com/atani/mysh/internal/mysql"
	"github.com/atani/mysh/internal/sqldump"
)

func RunSlice(args []string) error {
	var where string
	forceRaw := false
	outputFile := ""

	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--where":
			if i+1 < len(args) {
				i++
				where = args[i]
			} else {
				return fmt.Errorf("--where requires a value")
			}
		case "--raw":
			forceRaw = true
		case "-o", "--output":
			if i+1 < len(args) {
				i++
				outputFile = args[i]
			} else {
				return fmt.Errorf("-o requires a file path")
			}
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) < 2 {
		return fmt.Errorf("usage: mysh slice <name> <table> --where \"condition\"")
	}
	connName := positional[0]
	tableName := positional[1]

	if where == "" {
		return fmt.Errorf("--where is required")
	}

	if strings.ContainsRune(tableName, '`') {
		return fmt.Errorf("table name must not contain backtick characters")
	}

	// Reject semicolons in WHERE to prevent statement stacking in CLI path
	if strings.ContainsRune(where, ';') {
		return fmt.Errorf("WHERE clause must not contain semicolons")
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

	// Use environment-based masking policy consistent with run command
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	shouldMask := conn.ShouldMask(isTTY)
	if forceRaw && shouldMask {
		stdinTTY := term.IsTerminal(int(os.Stdin.Fd()))
		if conn.Env == "production" {
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
		} else {
			fmt.Fprintf(os.Stderr, "[mysh] --raw: masking disabled for connection %q\n", conn.Name)
		}
		shouldMask = false
	}

	if rc.isNative() {
		return runSliceNative(rc, conn, tableName, where, shouldMask, outputFile)
	}
	return runSliceCLI(rc, conn, tableName, where, shouldMask, outputFile)
}

func runSliceNative(rc *resolvedConn, conn *config.Connection, tableName, where string, shouldMask bool, outputFile string) error {
	dbConn, err := rc.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()

	// Prevent mutations via WHERE injection
	if _, err := db.Exec(dbConn, "SET SESSION TRANSACTION READ ONLY"); err != nil {
		return fmt.Errorf("read-only session not supported on this MySQL version; slice requires read-only protection")
	}

	query := fmt.Sprintf("SELECT * FROM `%s` WHERE %s", tableName, where)

	headers, rows, err := db.Query(dbConn, query)
	if err != nil {
		return err
	}

	if headers == nil || len(rows) == 0 {
		fmt.Fprintln(os.Stderr, "[mysh] no rows returned")
		return nil
	}

	result := &mysql.QueryResult{Headers: headers, Rows: rows}

	applyMasking(result, conn, shouldMask)

	return writeSliceOutput(result, tableName, where, outputFile)
}

func runSliceCLI(rc *resolvedConn, conn *config.Connection, tableName, where string, shouldMask bool, outputFile string) error {
	// Use read-only session to prevent accidental mutations via WHERE clause
	query := fmt.Sprintf("SET SESSION TRANSACTION READ ONLY; SELECT * FROM `%s` WHERE %s", tableName, where)

	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return err
	}
	defer cleanup()

	mysqlArgs = append(mysqlArgs, "-e", query)

	mysqlCmd := exec.Command("mysql", mysqlArgs...)
	mysqlCmd.Stderr = os.Stderr

	var buf bytes.Buffer
	mysqlCmd.Stdout = &buf

	if err := mysqlCmd.Run(); err != nil {
		return err
	}

	output := buf.String()
	result := mysql.ParseOutput(output)
	if result == nil {
		fmt.Fprintln(os.Stderr, "[mysh] no rows returned")
		return nil
	}

	applyMasking(result, conn, shouldMask)

	return writeSliceOutput(result, tableName, where, outputFile)
}

func applyMasking(result *mysql.QueryResult, conn *config.Connection, shouldMask bool) {
	if !shouldMask {
		return
	}
	maskedCols := mask.FindMaskColumns(result.Headers, conn.Mask.Columns, conn.Mask.Patterns)
	if len(maskedCols) == 0 {
		return
	}
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

func writeSliceOutput(result *mysql.QueryResult, tableName, where, outputFile string) error {
	dump := sqldump.Generate(tableName, result, sqldump.Options{
		Where:     where,
		Timestamp: time.Now(),
	})

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(dump), 0600); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[mysh] wrote %d rows to %s\n", len(result.Rows), outputFile)
		return nil
	}

	fmt.Print(dump)
	return nil
}
