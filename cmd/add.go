package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
)

func RunAdd(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	r := bufio.NewReader(os.Stdin)

	// SSH settings
	useSSH := askYesNo(r, "Use SSH tunnel?", false)

	var sshHost, sshUser, sshKey string
	var sshPort int
	if useSSH {
		sshHost = askRequired(r, "SSH host")
		sshPort = askInt(r, "SSH port", 22)
		sshUser = askRequired(r, "SSH user")
		sshKey = ask(r, "SSH key path (empty for default)", "")
	}

	// DB settings
	defaultHost := "127.0.0.1"
	if !useSSH {
		defaultHost = "localhost"
	}
	dbHost := ask(r, "MySQL host", defaultHost)
	dbPort := askInt(r, "MySQL port", 3306)
	dbUser := askRequired(r, "MySQL user")

	fmt.Fprint(os.Stderr, "MySQL password: ")
	dbPass, err := crypto.ReadPassword()
	if err != nil {
		return err
	}

	dbName := askRequired(r, "Database name")

	// Connection name last
	name := askValidated(r, "Connection name", func(s string) error {
		if s == "" {
			return fmt.Errorf("name is required")
		}
		if cfg.Find(s) != nil {
			return fmt.Errorf("connection %q already exists", s)
		}
		return nil
	})

	// Encrypt password
	var encryptedPassword string
	if dbPass != "" {
		masterPass, err := getMasterPassword()
		if err != nil {
			return err
		}
		enc, err := crypto.Encrypt([]byte(dbPass), masterPass)
		if err != nil {
			return fmt.Errorf("encrypting password: %w", err)
		}
		encryptedPassword, err = crypto.MarshalEncrypted(enc)
		if err != nil {
			return fmt.Errorf("encoding encrypted password: %w", err)
		}
	}

	conn := config.Connection{
		Name: name,
		DB: config.DBConfig{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Database: dbName,
			Password: encryptedPassword,
		},
	}

	if useSSH {
		conn.SSH = &config.SSHConfig{
			Host: sshHost,
			Port: sshPort,
			User: sshUser,
			Key:  sshKey,
		}
	}

	if err := cfg.Add(conn); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Connection %q added.\n", name)
	return nil
}

func getMasterPassword() ([]byte, error) {
	if !crypto.MasterPasswordInitialized() {
		fmt.Fprintln(os.Stderr, "Setting up master password for the first time.")
		fmt.Fprintln(os.Stderr, "This password protects your stored database credentials.")
		fmt.Fprint(os.Stderr, "Master password: ")
		pass, err := crypto.ReadPassword()
		if err != nil {
			return nil, err
		}
		if pass == "" {
			return nil, fmt.Errorf("master password cannot be empty")
		}
		fmt.Fprint(os.Stderr, "Confirm master password: ")
		confirm, err := crypto.ReadPassword()
		if err != nil {
			return nil, err
		}
		if pass != confirm {
			return nil, fmt.Errorf("passwords do not match")
		}
		if err := config.EnsureDir(); err != nil {
			return nil, err
		}
		if err := crypto.InitMasterPassword([]byte(pass)); err != nil {
			return nil, err
		}
		return []byte(pass), nil
	}

	fmt.Fprint(os.Stderr, "Master password: ")
	pass, err := crypto.ReadPassword()
	if err != nil {
		return nil, err
	}
	if err := crypto.VerifyMasterPassword([]byte(pass)); err != nil {
		return nil, err
	}
	return []byte(pass), nil
}

func ask(r *bufio.Reader, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", prompt)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func askRequired(r *bufio.Reader, prompt string) string {
	for {
		val := ask(r, prompt, "")
		if val != "" {
			return val
		}
		fmt.Fprintf(os.Stderr, "  %s is required.\n", prompt)
	}
}

func askValidated(r *bufio.Reader, prompt string, validate func(string) error) string {
	for {
		val := ask(r, prompt, "")
		if err := validate(val); err != nil {
			fmt.Fprintf(os.Stderr, "  %v\n", err)
			continue
		}
		return val
	}
}

func askInt(r *bufio.Reader, prompt string, defaultVal int) int {
	s := ask(r, prompt, strconv.Itoa(defaultVal))
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}

func askYesNo(r *bufio.Reader, prompt string, defaultVal bool) bool {
	hint := "y/N"
	if defaultVal {
		hint = "Y/n"
	}
	fmt.Fprintf(os.Stderr, "%s [%s]: ", prompt, hint)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	switch line {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	default:
		return defaultVal
	}
}
