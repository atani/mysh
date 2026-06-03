package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/i18n"
	"github.com/atani/mysh/internal/keychain"
)

type addFlags struct {
	name            string
	env             string
	mask            string
	driver          string
	dbHost          string
	dbPort          int
	dbUser          string
	dbName          string
	sshHost         string
	sshPort         int
	sshUser         string
	sshKey          string
	redashURL       string
	redashKey       string
	redashDatasource int
}

func parseAddFlags(args []string) (*addFlags, error) {
	f := &addFlags{dbPort: -1, sshPort: -1, redashDatasource: -1}
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
		case "--driver":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--driver requires a value (cli or native)")
			}
			i++
			f.driver = args[i]
		case "--redash-url":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--redash-url requires a value")
			}
			i++
			f.redashURL = args[i]
		case "--redash-key":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--redash-key requires a value")
			}
			i++
			f.redashKey = args[i]
		case "--redash-datasource":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--redash-datasource requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("--redash-datasource: invalid number %q", args[i])
			}
			f.redashDatasource = n
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

	// Redash mode: skip DB/SSH setup entirely
	if flags.redashURL != "" {
		return addRedashConnection(r, cfg, flags)
	}

	// SSH settings
	useSSH := flags.sshHost != ""
	if !useSSH {
		useSSH = askYesNo(r, i18n.T(i18n.AddUseSSH), false)
	}

	var sshHost, sshUser, sshKey string
	var sshPort int
	if useSSH {
		sshHost = askIfEmpty(r, flags.sshHost, i18n.T(i18n.AddSSHHost), "")
		if flags.sshPort >= 0 {
			sshPort = flags.sshPort
		} else {
			sshPort = askInt(r, i18n.T(i18n.AddSSHPort), 22)
		}
		sshUser = askIfEmpty(r, flags.sshUser, i18n.T(i18n.AddSSHUser), "")
		if flags.sshKey != "" {
			sshKey = flags.sshKey
		} else {
			sshKey = ask(r, i18n.T(i18n.AddSSHKeyDefault), "")
		}
	}

	// DB settings
	defaultHost := "127.0.0.1"
	if !useSSH {
		defaultHost = "localhost"
	}
	dbHost := askIfEmptyDefault(r, flags.dbHost, i18n.T(i18n.AddMySQLHost), defaultHost)
	var dbPort int
	if flags.dbPort >= 0 {
		dbPort = flags.dbPort
	} else {
		dbPort = askInt(r, i18n.T(i18n.AddMySQLPort), 3306)
	}
	dbUser := askIfEmpty(r, flags.dbUser, i18n.T(i18n.AddMySQLUser), "")

	fmt.Fprint(os.Stderr, i18n.T(i18n.AddMySQLPassword))
	dbPass, err := crypto.ReadPassword()
	if err != nil {
		return err
	}

	dbName := askIfEmpty(r, flags.dbName, i18n.T(i18n.AddDatabaseName), "")

	// Environment
	var env string
	if flags.env != "" {
		env = normalizeEnv(flags.env)
		if env == "" {
			return fmt.Errorf("invalid environment %q: must be production/prod, staging/stg, or development/dev", flags.env)
		}
	} else {
		env = askEnv(r, "development")
	}
	if env == "production" {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddProdMaskNotice))
		if !askYesNo(r, i18n.T(i18n.AddProdConfirm), true) {
			env = askEnv(r, "development")
		}
	}

	// Mask settings (for non-development)
	var maskCfg *config.MaskConfig
	if env != "development" {
		if flags.mask != "" {
			maskCfg = parseMaskInput(flags.mask)
		} else {
			defaultMask := "email,phone,*password*,*secret*,*token*,*address*"
			maskStr := ask(r, i18n.T(i18n.AddMaskColumns), defaultMask)
			maskCfg = parseMaskInput(maskStr)
		}
	}

	// Driver selection
	var driver string
	if flags.driver != "" {
		switch flags.driver {
		case config.DriverCLI, config.DriverNative:
			driver = flags.driver
		default:
			return fmt.Errorf("invalid driver %q: must be cli or native", flags.driver)
		}
	} else {
		driver = askDriver(r)
	}
	if driver == config.DriverNative {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.NativeDriverWarning1))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.NativeDriverWarning2))
	}

	// Connection name
	var name string
	if flags.name != "" {
		if cfg.Find(flags.name) != nil {
			return fmt.Errorf(i18n.T(i18n.ErrConnExists), flags.name)
		}
		name = flags.name
	} else {
		name = askValidated(r, i18n.T(i18n.AddConnName), func(s string) error {
			if s == "" {
				return errors.New(i18n.T(i18n.ErrNameRequired))
			}
			if cfg.Find(s) != nil {
				return fmt.Errorf(i18n.T(i18n.ErrConnExists), s)
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
			Driver:   driver,
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

	fmt.Fprintf(os.Stderr, i18n.T(i18n.AddConnAdded)+"\n", conn.Name)

	// Connection test loop
	if !askYesNo(r, i18n.T(i18n.AddTestConn), true) {
		return nil
	}

	for {
		if err := testConnection(&conn); err != nil {
			fmt.Fprintf(os.Stderr, "\n"+i18n.T(i18n.AddConnFailed)+"\n", err)
			choice := askFixChoice(r, conn.SSH != nil)
			if choice == "skip" {
				fmt.Fprintln(os.Stderr, i18n.T(i18n.AddSkippedFix))
				return nil
			}
			if err := applyFix(r, &conn, choice); err != nil {
				return err
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, i18n.T(i18n.AddRetesting))
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

	if rc.isNative() {
		dbConn, err := rc.openDB()
		if err != nil {
			return err
		}
		defer func() { _ = dbConn.Close() }()

		if err := db.Ping(dbConn); err != nil {
			return err
		}
	} else {
		mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
		if err != nil {
			return err
		}
		defer cleanup()

		mysqlArgs = append(mysqlArgs, "-e", "SELECT 1")

		c := exec.Command("mysql", mysqlArgs...)
		c.Stdout = nil
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return err
		}
	}

	elapsed := time.Since(start)
	fmt.Fprintf(os.Stderr, i18n.T(i18n.AddConnOK)+"\n", conn.Name, elapsed.Round(time.Millisecond))
	return nil
}

func askFixChoice(r *bufio.Reader, hasSSH bool) string {
	fmt.Fprintln(os.Stderr, "\n"+i18n.T(i18n.AddFixTitle))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixMySQLHostPort))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixMySQLUserPass))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixDBName))
	if hasSSH {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixSSH))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixSkip5))
	} else {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddFixSkip4))
	}

	for {
		choice := ask(r, i18n.T(i18n.AddChoice), "")
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
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddInvalidChoice))
	}
}

func applyFix(r *bufio.Reader, conn *config.Connection, choice string) error {
	switch choice {
	case "db-host":
		conn.DB.Host = askEdit(r, i18n.T(i18n.AddMySQLHost), conn.DB.Host)
		conn.DB.Port = askIntEdit(r, i18n.T(i18n.AddMySQLPort), conn.DB.Port)
	case "db-auth":
		conn.DB.User = askEdit(r, i18n.T(i18n.AddMySQLUser), conn.DB.User)
		fmt.Fprint(os.Stderr, i18n.T(i18n.AddMySQLPasswordKeep))
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
		conn.DB.Database = askEdit(r, i18n.T(i18n.AddDatabaseName), conn.DB.Database)
	case "ssh":
		if conn.SSH != nil {
			conn.SSH.Host = askEdit(r, i18n.T(i18n.AddSSHHost), conn.SSH.Host)
			conn.SSH.Port = askIntEdit(r, i18n.T(i18n.AddSSHPort), conn.SSH.Port)
			conn.SSH.User = askEdit(r, i18n.T(i18n.AddSSHUser), conn.SSH.User)
			conn.SSH.Key = askEdit(r, i18n.T(i18n.AddSSHKey), conn.SSH.Key)
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
	// Try the OS credential store first (macOS Keychain / Windows Credential
	// Manager; silently ignored on unsupported platforms).
	if cached, err := keychain.Get(); err == nil && cached != "" {
		if err := crypto.VerifyMasterPassword([]byte(cached)); err == nil {
			return []byte(cached), nil
		}
		// Cached password is invalid; fall through to prompt
	}

	// Try environment variable (useful for non-TTY contexts like AI assistants)
	if envPass := os.Getenv("MYSH_MASTER_PASSWORD"); envPass != "" {
		if err := crypto.VerifyMasterPassword([]byte(envPass)); err == nil {
			return []byte(envPass), nil
		}
		// Environment password is invalid; fall through to prompt
	}

	// In non-interactive contexts (e.g. the MCP server) stdin carries protocol
	// data, so we must never fall through to an interactive password prompt.
	// Fail fast with an actionable message instead.
	if nonInteractive {
		return nil, errors.New(i18n.T(i18n.ErrMasterUnavailable))
	}

	if !crypto.MasterPasswordInitialized() {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddMasterSetupTitle))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddMasterSetupDesc))
		fmt.Fprint(os.Stderr, i18n.T(i18n.AddMasterPrompt))
		pass, err := crypto.ReadPassword()
		if err != nil {
			return nil, err
		}
		if pass == "" {
			return nil, errors.New(i18n.T(i18n.ErrMasterEmpty))
		}
		fmt.Fprint(os.Stderr, i18n.T(i18n.AddMasterConfirm))
		confirm, err := crypto.ReadPassword()
		if err != nil {
			return nil, err
		}
		if pass != confirm {
			return nil, errors.New(i18n.T(i18n.ErrPasswordMismatch))
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

	fmt.Fprint(os.Stderr, i18n.T(i18n.AddMasterPrompt))
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
		fmt.Fprintf(os.Stderr, i18n.T(i18n.AddMasterSaved)+"\n", keychain.Name())
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
		fmt.Fprintf(os.Stderr, i18n.T(i18n.AddRequiredSuffix)+"\n", prompt)
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

func normalizeEnv(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "production", "prod":
		return "production"
	case "staging", "stg":
		return "staging"
	case "development", "dev":
		return "development"
	default:
		return ""
	}
}

func askEnv(r *bufio.Reader, defaultVal string) string {
	defaultNum := "3"
	for i, v := range config.Environments {
		if v == defaultVal {
			defaultNum = fmt.Sprintf("%d", i+1)
		}
	}
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddEnvTitle))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddEnvProd))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddEnvStg))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddEnvDev))
	for {
		choice := ask(r, i18n.T(i18n.AddChoice), defaultNum)
		switch choice {
		case "1", "production", "prod":
			return "production"
		case "2", "staging", "stg":
			return "staging"
		case "3", "development", "dev":
			return "development"
		}
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddEnvInvalid))
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

func askDriver(r *bufio.Reader) string {
	return askDriverEdit(r, config.DriverCLI)
}

func askDriverEdit(r *bufio.Reader, current string) string {
	defaultNum := "1"
	if current == config.DriverNative {
		defaultNum = "2"
	}
	fmt.Fprintln(os.Stderr, i18n.T(i18n.DriverMenuTitle))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.DriverMenuCLI))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.DriverMenuNative))
	for {
		choice := ask(r, i18n.T(i18n.AddChoice), defaultNum)
		switch choice {
		case "1", "cli":
			return config.DriverCLI
		case "2", "native":
			return config.DriverNative
		}
		fmt.Fprintln(os.Stderr, i18n.T(i18n.DriverMenuInvalid))
	}
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

func addRedashConnection(r *bufio.Reader, cfg *config.Config, flags *addFlags) error {
	name := flags.name
	if name == "" {
		name = askRequired(r, i18n.T(i18n.AddConnName))
	}
	if cfg.Find(name) != nil {
		return fmt.Errorf(i18n.T(i18n.ErrConnExists), name)
	}

	redashURL := flags.redashURL
	apiKey := flags.redashKey
	if apiKey == "" {
		fmt.Fprint(os.Stderr, i18n.T(i18n.AddRedashAPIKey))
		var err error
		apiKey, err = crypto.ReadPassword()
		if err != nil {
			return err
		}
		if apiKey == "" {
			return errors.New(i18n.T(i18n.ErrAPIKeyRequired))
		}
	}

	dataSourceID := flags.redashDatasource
	if dataSourceID < 0 {
		dataSourceID = askInt(r, i18n.T(i18n.AddDataSourceID), 1)
	}

	env := flags.env
	if env == "" {
		env = askEnv(r, "production")
	}

	maskInput := flags.mask
	if maskInput == "" {
		defaultMask := "email,phone,*password*,*secret*,*token*,*address*"
		fmt.Fprintf(os.Stderr, i18n.T(i18n.AddMaskColumnsBracket), defaultMask)
		line, _ := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			maskInput = defaultMask
		} else {
			maskInput = line
		}
	}

	// Encrypt API key
	masterPass, err := getMasterPassword()
	if err != nil {
		return err
	}
	enc, err := crypto.Encrypt([]byte(apiKey), masterPass)
	if err != nil {
		return fmt.Errorf("encrypting API key: %w", err)
	}
	encAPIKey, err := crypto.MarshalEncrypted(enc)
	if err != nil {
		return fmt.Errorf("encoding encrypted API key: %w", err)
	}

	conn := config.Connection{
		Name: name,
		Env:  env,
		Redash: &config.RedashConfig{
			URL:          redashURL,
			APIKey:       encAPIKey,
			DataSourceID: dataSourceID,
		},
		Mask: parseMaskInput(maskInput),
	}

	if err := cfg.Add(conn); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, i18n.T(i18n.AddRedashAdded)+"\n", name)
	return nil
}
