package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/keychain"
)

type addFlags struct {
	name    string
	env     string
	mask    string
	dbHost  string
	dbPort  int
	dbUser  string
	dbName  string
	sshHost string
	sshPort int
	sshUser string
	sshKey  string
}

func parseAddFlags(args []string) (*addFlags, error) {
	f := &addFlags{dbPort: -1, sshPort: -1}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--name requires a value")
			}
			i++
			f.name = args[i]
		case "--env":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--env requires a value")
			}
			i++
			f.env = args[i]
		case "--mask":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--mask requires a value")
			}
			i++
			f.mask = args[i]
		case "--db-host":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--db-host requires a value")
			}
			i++
			f.dbHost = args[i]
		case "--db-port":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--db-port requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("--db-port: invalid number %q", args[i])
			}
			f.dbPort = n
		case "--db-user":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--db-user requires a value")
			}
			i++
			f.dbUser = args[i]
		case "--db-name":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--db-name requires a value")
			}
			i++
			f.dbName = args[i]
		case "--ssh-host":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--ssh-host requires a value")
			}
			i++
			f.sshHost = args[i]
		case "--ssh-port":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--ssh-port requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("--ssh-port: invalid number %q", args[i])
			}
			f.sshPort = n
		case "--ssh-user":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--ssh-user requires a value")
			}
			i++
			f.sshUser = args[i]
		case "--ssh-key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--ssh-key requires a value")
			}
			i++
			f.sshKey = args[i]
		default:
			return nil, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	return f, nil
}

func RunAdd(args []string) error {
	flags, err := parseAddFlags(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	r := bufio.NewReader(os.Stdin)

	// SSH settings
	useSSH := flags.sshHost != ""
	if !useSSH && flags.sshHost == "" {
		useSSH = askYesNo(r, "Use SSH tunnel?", false)
	}

	var sshHost, sshUser, sshKey string
	var sshPort int
	if useSSH {
		sshHost = askIfEmpty(r, flags.sshHost, "SSH host", "")
		if flags.sshPort >= 0 {
			sshPort = flags.sshPort
		} else {
			sshPort = askInt(r, "SSH port", 22)
		}
		sshUser = askIfEmpty(r, flags.sshUser, "SSH user", "")
		if flags.sshKey != "" {
			sshKey = flags.sshKey
		} else {
			sshKey = ask(r, "SSH key path (empty for default)", "")
		}
	}

	// DB settings
	defaultHost := "127.0.0.1"
	if !useSSH {
		defaultHost = "localhost"
	}
	dbHost := askIfEmptyDefault(r, flags.dbHost, "MySQL host", defaultHost)
	var dbPort int
	if flags.dbPort >= 0 {
		dbPort = flags.dbPort
	} else {
		dbPort = askInt(r, "MySQL port", 3306)
	}
	dbUser := askIfEmpty(r, flags.dbUser, "MySQL user", "")

	fmt.Fprint(os.Stderr, "MySQL password: ")
	dbPass, err := crypto.ReadPassword()
	if err != nil {
		return err
	}

	dbName := askIfEmpty(r, flags.dbName, "Database name", "")

	// Environment
	var env string
	if flags.env != "" {
		if !isValidEnv(flags.env) {
			return fmt.Errorf("invalid environment %q: must be production, staging, or development", flags.env)
		}
		env = flags.env
	} else {
		env = askEnv(r, "development")
	}

	// Mask settings (for non-development)
	var maskCfg *config.MaskConfig
	if env != "development" {
		if flags.mask != "" {
			maskCfg = parseMaskInput(flags.mask)
		} else {
			defaultMask := "email,phone,*password*,*secret*,*token*,*address*"
			maskStr := ask(r, "Columns to mask (comma-separated, wildcards OK)", defaultMask)
			maskCfg = parseMaskInput(maskStr)
		}
	}

	// Connection name
	var name string
	if flags.name != "" {
		if cfg.Find(flags.name) != nil {
			return fmt.Errorf("connection %q already exists", flags.name)
		}
		name = flags.name
	} else {
		name = askValidated(r, "Connection name", func(s string) error {
			if s == "" {
				return fmt.Errorf("name is required")
			}
			if cfg.Find(s) != nil {
				return fmt.Errorf("connection %q already exists", s)
			}
			return nil
		})
	}

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
		Env:  env,
		Mask: maskCfg,
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

	fmt.Fprintf(os.Stderr, "Connection %q added.\n", conn.Name)

	// Connection test loop
	if !askYesNo(r, "Test connection?", true) {
		return nil
	}

	for {
		if err := testConnection(&conn); err != nil {
			fmt.Fprintf(os.Stderr, "\nConnection failed: %v\n", err)
			choice := askFixChoice(r, conn.SSH != nil)
			if choice == "skip" {
				fmt.Fprintln(os.Stderr, "Skipped. You can fix settings later with `mysh edit`.")
				return nil
			}
			if err := applyFix(r, &conn, choice); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Retesting...")
			continue
		}
		return nil
	}
}

func testConnection(conn *config.Connection) error {
	rc, err := resolveConnection(conn)
	if err != nil {
		return err
	}
	defer rc.cleanup()

	start := time.Now()
	mysqlArgs := rc.mysqlArgs()
	mysqlArgs = append(mysqlArgs, "-e", "SELECT 1")

	c := exec.Command("mysql", mysqlArgs...)
	c.Stdout = nil
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		return err
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, "Connection %q: OK (%s)\n", conn.Name, elapsed.Round(time.Millisecond))
	return nil
}

func askFixChoice(r *bufio.Reader, hasSSH bool) string {
	fmt.Fprintln(os.Stderr, "\nWhat would you like to fix?")
	fmt.Fprintln(os.Stderr, "  1) MySQL host/port")
	fmt.Fprintln(os.Stderr, "  2) MySQL user/password")
	fmt.Fprintln(os.Stderr, "  3) Database name")
	if hasSSH {
		fmt.Fprintln(os.Stderr, "  4) SSH settings")
		fmt.Fprintln(os.Stderr, "  5) Skip")
	} else {
		fmt.Fprintln(os.Stderr, "  4) Skip")
	}

	for {
		choice := ask(r, "Choice", "")
		switch choice {
		case "1":
			return "db-host"
		case "2":
			return "db-auth"
		case "3":
			return "db-name"
		case "4":
			if hasSSH {
				return "ssh"
			}
			return "skip"
		case "5":
			if hasSSH {
				return "skip"
			}
		}
		fmt.Fprintln(os.Stderr, "  Invalid choice.")
	}
}

func applyFix(r *bufio.Reader, conn *config.Connection, choice string) error {
	switch choice {
	case "db-host":
		conn.DB.Host = askEdit(r, "MySQL host", conn.DB.Host)
		conn.DB.Port = askIntEdit(r, "MySQL port", conn.DB.Port)
	case "db-auth":
		conn.DB.User = askEdit(r, "MySQL user", conn.DB.User)
		fmt.Fprint(os.Stderr, "MySQL password (Enter to keep): ")
		newPass, err := crypto.ReadPassword()
		if err != nil {
			return err
		}
		if newPass != "" {
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
	case "db-name":
		conn.DB.Database = askEdit(r, "Database name", conn.DB.Database)
	case "ssh":
		if conn.SSH != nil {
			conn.SSH.Host = askEdit(r, "SSH host", conn.SSH.Host)
			conn.SSH.Port = askIntEdit(r, "SSH port", conn.SSH.Port)
			conn.SSH.User = askEdit(r, "SSH user", conn.SSH.User)
			conn.SSH.Key = askEdit(r, "SSH key path", conn.SSH.Key)
		}
	}
	return nil
}

// askIfEmpty prompts only if the flag value is empty. Required field.
func askIfEmpty(r *bufio.Reader, flagVal, prompt, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	if defaultVal != "" {
		return ask(r, prompt, defaultVal)
	}
	return askRequired(r, prompt)
}

// askIfEmptyDefault prompts only if the flag value is empty. Has a default.
func askIfEmptyDefault(r *bufio.Reader, flagVal, prompt, defaultVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return ask(r, prompt, defaultVal)
}

func getMasterPassword() ([]byte, error) {
	// Try keychain first (macOS only, silently ignored on other platforms)
	if cached, err := keychain.Get(); err == nil && cached != "" {
		if err := crypto.VerifyMasterPassword([]byte(cached)); err == nil {
			return []byte(cached), nil
		}
		// Cached password is invalid; fall through to prompt
	}

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
		saveToKeychain(pass)
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
	saveToKeychain(pass)
	return []byte(pass), nil
}

func saveToKeychain(password string) {
	if err := keychain.Set(password); err == nil {
		fmt.Fprintln(os.Stderr, "Master password saved to keychain.")
	}
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

var validEnvs = []string{"production", "staging", "development"}

func isValidEnv(env string) bool {
	for _, v := range validEnvs {
		if env == v {
			return true
		}
	}
	return false
}

func askEnv(r *bufio.Reader, defaultVal string) string {
	for {
		val := ask(r, "Environment (production/staging/development)", defaultVal)
		if isValidEnv(val) {
			return val
		}
		fmt.Fprintln(os.Stderr, "  Must be production, staging, or development.")
	}
}

// parseMaskInput splits a comma-separated input into columns (exact match)
// and patterns (wildcard match) based on whether the value contains "*".
func parseMaskInput(input string) *config.MaskConfig {
	if input == "" {
		return nil
	}
	var cols, patterns []string
	for _, s := range strings.Split(input, ",") {
		v := strings.TrimSpace(s)
		if v == "" {
			continue
		}
		if strings.Contains(v, "*") {
			patterns = append(patterns, v)
		} else {
			cols = append(cols, v)
		}
	}
	if len(cols) == 0 && len(patterns) == 0 {
		return nil
	}
	return &config.MaskConfig{Columns: cols, Patterns: patterns}
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
