package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/atani/mysh/internal/db"
)

func RunConnect(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	_, conn, err := findConnection(name)
	if err != nil {
		return err
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	if rc.isNative() {
		return runConnectNative(rc)
	}
	return runConnectCLI(rc)
}

func runConnectCLI(rc *resolvedConn) error {
	client := "mycli"
	if _, err := lookPath("mycli"); err != nil {
		client = "mysql"
		if _, err := lookPath("mysql"); err != nil {
			return fmt.Errorf("neither mycli nor mysql found in PATH")
		}
	}

	args, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return err
	}
	defer cleanup()

	c := execCommand(client, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func runConnectNative(rc *resolvedConn) error {
	dbConn, err := rc.openDB()
	if err != nil {
		return err
	}
	defer func() { _ = dbConn.Close() }()

	if err := db.Ping(dbConn); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	database := rc.database
	if database == "" {
		database = "(none)"
	}
	fmt.Fprintf(os.Stderr, "Connected to %s:%d as %s (database: %s)\n", rc.host, rc.port, rc.user, database)
	fmt.Fprintln(os.Stderr, "Type SQL statements, or 'quit' to exit.")

	isTTY := stdinIsTTY()
	queryFn := func(stmt string) ([]string, [][]string, error) {
		return db.Query(dbConn, stmt)
	}
	return runREPL(os.Stdin, os.Stdout, os.Stderr, isTTY, queryFn)
}

// replQueryFunc executes a single SQL statement and returns the result set.
// A nil headers slice means the statement produced no rows (e.g. an OK status).
type replQueryFunc func(stmt string) (headers []string, rows [][]string, err error)

// runREPL drives the interactive native-driver prompt loop. It is factored out
// of runConnectNative so the line-buffering, multi-statement, and quit logic
// can be tested without a live database connection.
func runREPL(in io.Reader, out, errOut io.Writer, isTTY bool, query replQueryFunc) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 1024*1024) // 1MB max line
	var pending strings.Builder

	for {
		if isTTY {
			if pending.Len() == 0 {
				fmt.Fprint(errOut, "mysql> ")
			} else {
				fmt.Fprint(errOut, "    -> ")
			}
		}

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if pending.Len() == 0 {
			lower := strings.ToLower(trimmed)
			if lower == "quit" || lower == "exit" || lower == "\\q" {
				fmt.Fprintln(errOut, "Bye")
				return nil
			}
			if trimmed == "" {
				continue
			}
		}

		if pending.Len() > 0 {
			pending.WriteByte(' ')
		}
		pending.WriteString(line)

		full := strings.TrimSpace(pending.String())
		if !strings.HasSuffix(full, ";") {
			continue
		}

		stmt := strings.TrimSuffix(full, ";")
		stmt = strings.TrimSpace(stmt)
		pending.Reset()

		if stmt == "" {
			continue
		}

		headers, rows, err := query(stmt)
		if err != nil {
			fmt.Fprintf(errOut, "ERROR: %v\n", err)
			continue
		}

		if headers == nil {
			fmt.Fprintln(errOut, "Query OK")
			continue
		}

		output := db.FormatTabular(headers, rows)
		fmt.Fprint(out, output)
		fmt.Fprintf(errOut, "%d rows in set\n", len(rows))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	return nil
}
