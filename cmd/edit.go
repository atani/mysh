package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/i18n"
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
	hasSSH := conn.SSH != nil
	useSSH := askYesNo(r, "Use SSH tunnel?", hasSSH)
	if useSSH {
		if conn.SSH == nil {
			conn.SSH = &config.SSHConfig{Port: 22}
		}
		conn.SSH.Host = askEdit(r, "SSH host", conn.SSH.Host)
		conn.SSH.Port = askIntEdit(r, "SSH port", conn.SSH.Port)
		conn.SSH.User = askEdit(r, "SSH user", conn.SSH.User)
		conn.SSH.Key = askEdit(r, "SSH key path", conn.SSH.Key)
	} else {
		conn.SSH = nil
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

	// Driver
	conn.DB.Driver = askDriverEdit(r, conn.DB.EffectiveDriver())
	if conn.DB.Driver == config.DriverNative {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.NativeDriverWarning1))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.NativeDriverWarning2))
	}

	// Environment
	conn.Env = askEnv(r, conn.Env)

	// Mask settings
	if conn.Env != "development" {
		var currentMask string
		if conn.Mask != nil {
			parts := append(conn.Mask.Columns, conn.Mask.Patterns...)
			currentMask = strings.Join(parts, ",")
		}
		maskStr := askEdit(r, "Columns to mask (comma-separated, wildcards OK)", currentMask)
		conn.Mask = parseMaskInput(maskStr)
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
