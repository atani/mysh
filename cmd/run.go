package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/mask"
)

func RunRun(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mysh run <name> [-e \"SQL\" | <file.sql>] [--mask|--raw]")
	}

	connName := args[0]
	rest := args[1:]

	var sqlExpr string
	var sqlFile string
	forceMask := false
	forceRaw := false

	// Parse flags
	var remaining []string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--mask":
			forceMask = true
		case "--raw":
			forceRaw = true
		default:
			remaining = append(remaining, rest[i])
		}
	}

	if len(remaining) == 0 {
		return fmt.Errorf("usage: mysh run <name> [-e \"SQL\" | <file.sql>]")
	}

	if remaining[0] == "-e" {
		if len(remaining) < 2 {
			return fmt.Errorf("usage: mysh run <name> -e \"SQL\"")
		}
		sqlExpr = remaining[1]
	} else {
		sqlFile = remaining[0]
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

	// Determine masking
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	shouldMask := conn.ShouldMask(isTTY)
	if forceMask {
		shouldMask = true
	}
	if forceRaw {
		shouldMask = false
	}

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if !shouldMask {
		c.Stdout = os.Stdout
		return c.Run()
	}

	// Capture output for masking
	var buf bytes.Buffer
	c.Stdout = &buf

	if err := c.Run(); err != nil {
		return err
	}

	output := buf.String()

	// Parse header to determine which columns to mask
	headers := parseHeaders(output)
	maskedCols := conn.MaskColumns(headers)

	if len(maskedCols) == 0 {
		fmt.Print(output)
		return nil
	}

	fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(maskedColNames(headers, maskedCols), ", "))
	fmt.Print(mask.TabularOutput(output, maskedCols))
	return nil
}

// parseHeaders extracts column names from mysql output (TSV or tabular).
func parseHeaders(output string) []string {
	lines := strings.SplitN(output, "\n", 3)
	if len(lines) < 2 {
		return nil
	}

	// Tabular format: first line is +---+---+, second is | col1 | col2 |
	if strings.HasPrefix(lines[0], "+") && strings.HasPrefix(lines[1], "|") {
		raw := strings.Trim(lines[1], "| ")
		parts := strings.Split(raw, "|")
		var headers []string
		for _, p := range parts {
			headers = append(headers, strings.TrimSpace(p))
		}
		return headers
	}

	// TSV format: first line is headers
	return strings.Split(lines[0], "\t")
}

func maskedColNames(headers []string, maskedCols map[int]bool) []string {
	var names []string
	for i, h := range headers {
		if maskedCols[i] {
			names = append(names, h)
		}
	}
	return names
}
