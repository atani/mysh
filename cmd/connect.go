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
	prompt := "mysql> "
	if rc.isPostgres() {
		prompt = "postgres> "
	}
	return runREPL(os.Stdin, os.Stdout, os.Stderr, isTTY, prompt, rc.dialect(), queryFn)
}

// replQueryFunc executes a single SQL statement and returns the result set.
// A nil headers slice means the statement produced no rows (e.g. an OK status).
type replQueryFunc func(stmt string) (headers []string, rows [][]string, err error)

// runREPL drives the interactive native-driver prompt loop. It is factored out
// of runConnectNative so the line-buffering, multi-statement, and quit logic
// can be tested without a live database connection.
//
// Input is buffered until it ends with a ';' outside quotes, dollar quotes and
// comments of the given dialect. The buffered input is then split like
// `mysh run` does and each statement is executed in turn; the remaining
// statements are skipped after an error.
func runREPL(in io.Reader, out, errOut io.Writer, isTTY bool, prompt string, dialect db.Dialect, query replQueryFunc) error {
	continuation := strings.Repeat(" ", max(len(prompt)-3, 0)) + "-> "
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 1024*1024) // 1MB max line
	var pending strings.Builder

	for {
		if isTTY {
			if pending.Len() == 0 {
				fmt.Fprint(errOut, prompt)
			} else {
				fmt.Fprint(errOut, continuation)
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

		// Lines are joined with newlines so a line comment ends at its line.
		if pending.Len() > 0 {
			pending.WriteByte('\n')
		}
		pending.WriteString(line)

		full := pending.String()
		if !db.EndsWithTerminator(full, dialect) {
			continue
		}
		pending.Reset()

		for _, stmt := range db.SplitStatements(full, dialect) {
			if !runREPLStatement(out, errOut, query, stmt) {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	return nil
}

// runREPLStatement executes one statement and prints its result. It returns
// false when the statement failed.
func runREPLStatement(out, errOut io.Writer, query replQueryFunc, stmt string) bool {
	headers, rows, err := query(stmt)
	if err != nil {
		fmt.Fprintf(errOut, "ERROR: %v\n", err)
		return false
	}

	if headers == nil {
		fmt.Fprintln(errOut, "Query OK")
		return true
	}

	output := db.FormatTabular(headers, rows)
	fmt.Fprint(out, output)
	fmt.Fprintf(errOut, "%d rows in set\n", len(rows))
	return true
}
