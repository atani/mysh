package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/format"
	"github.com/atani/mysh/internal/mask"
)

func RunQuery(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: mysh run <name> [-e \"SQL\" | <file.sql>] [--mask|--raw] [--format plain|markdown|csv|pdf] [-o <file>]")
	}

	connName := args[0]
	rest := args[1:]

	var sqlExpr string
	var sqlFile string
	forceMask := false
	forceRaw := false
	formatStr := ""
	outputFile := ""

	// Parse flags
	var remaining []string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--mask":
			forceMask = true
		case "--raw":
			forceRaw = true
		case "--format":
			if i+1 < len(rest) {
				i++
				formatStr = rest[i]
			} else {
				return fmt.Errorf("--format requires a value (plain, markdown, csv, pdf)")
			}
		case "-o", "--output":
			if i+1 < len(rest) {
				i++
				outputFile = rest[i]
			} else {
				return fmt.Errorf("-o requires a file path")
			}
		default:
			remaining = append(remaining, rest[i])
		}
	}

	outFmt, err := format.Parse(formatStr)
	if err != nil {
		return err
	}

	if outFmt == format.PDF && outputFile == "" {
		return fmt.Errorf("PDF format requires -o <file> to specify output path")
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

	mysqlArgs := rc.mysqlArgs()

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

	// When format or output file is specified, always capture output
	needCapture := shouldMask || outFmt != format.Plain || outputFile != ""

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdin = os.Stdin
	c.Stderr = os.Stderr

	if !needCapture {
		c.Stdout = os.Stdout
		return c.Run()
	}

	// Capture output
	var buf bytes.Buffer
	c.Stdout = &buf

	if err := c.Run(); err != nil {
		return err
	}

	output := buf.String()

	// Apply masking
	if shouldMask && conn.Mask != nil {
		masked, colNames := mask.ApplyToOutput(output, conn.Mask.Columns, conn.Mask.Patterns)
		if len(colNames) > 0 {
			fmt.Fprintf(os.Stderr, "[mysh] masking columns: %s\n", strings.Join(colNames, ", "))
			output = masked
		}
	}

	// Apply format conversion
	return writeOutput(output, outFmt, outputFile)
}
