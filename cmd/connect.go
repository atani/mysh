package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"

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
	if _, err := exec.LookPath("mycli"); err != nil {
		client = "mysql"
		if _, err := exec.LookPath("mysql"); err != nil {
			return fmt.Errorf("neither mycli nor mysql found in PATH")
		}
	}

	args, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return err
	}
	defer cleanup()

	c := exec.Command(client, args...)
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

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), 1024*1024) // 1MB max line
	var pending strings.Builder

	for {
		if isTTY {
			if pending.Len() == 0 {
				fmt.Fprint(os.Stderr, "mysql> ")
			} else {
				fmt.Fprint(os.Stderr, "    -> ")
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
				fmt.Fprintln(os.Stderr, "Bye")
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

		query := strings.TrimSuffix(full, ";")
		query = strings.TrimSpace(query)
		pending.Reset()

		if query == "" {
			continue
		}

		headers, rows, err := db.Query(dbConn, query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			continue
		}

		if headers == nil {
			fmt.Fprintln(os.Stderr, "Query OK")
			continue
		}

		output := db.FormatTabular(headers, rows)
		fmt.Print(output)
		fmt.Fprintf(os.Stderr, "%d rows in set\n", len(rows))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	return nil
}
