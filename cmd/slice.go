package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/mask"
	"github.com/atani/mysh/internal/mysql"
	"github.com/atani/mysh/internal/sqldump"
)

func RunSlice(args []string) error {
	var where string
	forceMask := false
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
		case "--mask":
			forceMask = true
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

	sql := fmt.Sprintf("SELECT * FROM `%s` WHERE %s", tableName, where)

	mysqlArgs := rc.mysqlArgs()
	mysqlArgs = append(mysqlArgs, "-e", sql)

	c := exec.Command("mysql", mysqlArgs...)
	c.Stderr = os.Stderr

	var buf bytes.Buffer
	c.Stdout = &buf

	if err := c.Run(); err != nil {
		return err
	}

	output := buf.String()
	result := mysql.ParseOutput(output)
	if result == nil {
		fmt.Fprintln(os.Stderr, "[mysh] no rows returned")
		return nil
	}

	// Apply masking at data level
	if shouldMask && conn.Mask != nil {
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

	dump := sqldump.Generate(tableName, result, sqldump.Options{
		Where:     where,
		Timestamp: time.Now(),
	})

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(dump), 0644); err != nil {
			return fmt.Errorf("writing output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[mysh] wrote %d rows to %s\n", len(result.Rows), outputFile)
		return nil
	}

	fmt.Print(dump)
	return nil
}
