package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
)

func RunEdit(args []string) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}

	cfg, conn, err := findConnection(name)
	if err != nil {
		return err
	}

	r := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "Editing connection %q (press Enter to keep current value)\n\n", conn.Name)

	// SSH settings
	if conn.SSH != nil {
		conn.SSH.Host = askEdit(r, "SSH host", conn.SSH.Host)
		conn.SSH.Port = askIntEdit(r, "SSH port", conn.SSH.Port)
		conn.SSH.User = askEdit(r, "SSH user", conn.SSH.User)
		conn.SSH.Key = askEdit(r, "SSH key path", conn.SSH.Key)
	}

	// DB settings
	conn.DB.Host = askEdit(r, "MySQL host", conn.DB.Host)
	conn.DB.Port = askIntEdit(r, "MySQL port", conn.DB.Port)
	conn.DB.User = askEdit(r, "MySQL user", conn.DB.User)
	conn.DB.Database = askEdit(r, "Database name", conn.DB.Database)

	// Password
	fmt.Fprint(os.Stderr, "MySQL password (Enter to keep, 'clear' to remove): ")
	newPass, err := crypto.ReadPassword()
	if err != nil {
		return err
	}
	if newPass == "clear" {
		conn.DB.Password = ""
	} else if newPass != "" {
		masterPass, err := getMasterPassword()
		if err != nil {
			return err
		}
		enc, err := crypto.Encrypt([]byte(newPass), masterPass)
		if err != nil {
			return fmt.Errorf("encrypting password: %w", err)
		}
		conn.DB.Password, err = crypto.MarshalEncrypted(enc)
		if err != nil {
			return fmt.Errorf("encoding encrypted password: %w", err)
		}
	}

	// Environment
	conn.Env = askEdit(r, "Environment (production/staging/development)", conn.Env)

	// Mask settings
	if conn.Env != "development" {
		var currentCols, currentPatterns string
		if conn.Mask != nil {
			currentCols = strings.Join(conn.Mask.Columns, ",")
			currentPatterns = strings.Join(conn.Mask.Patterns, ",")
		}
		colsStr := askEdit(r, "Columns to mask (comma-separated)", currentCols)
		patternsStr := askEdit(r, "Column patterns to mask (comma-separated)", currentPatterns)

		var cols, patterns []string
		if colsStr != "" {
			for _, c := range strings.Split(colsStr, ",") {
				if v := strings.TrimSpace(c); v != "" {
					cols = append(cols, v)
				}
			}
		}
		if patternsStr != "" {
			for _, p := range strings.Split(patternsStr, ",") {
				if v := strings.TrimSpace(p); v != "" {
					patterns = append(patterns, v)
				}
			}
		}
		if len(cols) > 0 || len(patterns) > 0 {
			conn.Mask = &config.MaskConfig{Columns: cols, Patterns: patterns}
		} else {
			conn.Mask = nil
		}
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Connection %q updated.\n", conn.Name)
	return nil
}

func askEdit(r *bufio.Reader, prompt, current string) string {
	if current != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, current)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return current
	}
	return line
}

func askIntEdit(r *bufio.Reader, prompt string, current int) int {
	s := askEdit(r, prompt, fmt.Sprintf("%d", current))
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return current
	}
	return n
}
