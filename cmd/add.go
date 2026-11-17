package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
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
	name             string
	env              string
	mask             string
	driver           string
	dbType           string
	sslmode          string
	url              string
	dbHost           string
	dbPort           int
	dbUser           string
	dbName           string
	sshHost          string
	sshPort          int
	sshUser          string
	sshKey           string
	redashURL        string
	redashKey        string
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
		case "--type":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--type requires a value (mysql, postgres, or supabase)")
			}
			i++
			f.dbType = args[i]
		case "--sslmode":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--sslmode requires a value (%s)", strings.Join(config.SSLModes, ", "))
			}
			i++
			f.sslmode = args[i]
		case "--url":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--url requires a value (postgresql://...)")
			}
			i++
			f.url = args[i]
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
	if err := f.validateDBType(); err != nil {
		return nil, err
	}
	return f, nil
}

// Database kinds accepted by `mysh add --type` and the interactive menu.
// "supabase" is an input preset only: it is saved as type postgres.
const (
	addKindMySQL    = "mysql"
	addKindPostgres = "postgres"
	addKindSupabase = "supabase"
)

// supabaseSSLMode is the default sslmode for the Supabase preset; Supabase
// accepts only encrypted connections from outside its network.
const supabaseSSLMode = "require"

// validateDBType resolves the --type alias, checks the flag combinations that
// exist only on the command line, and checks --type, --driver and --sslmode
// against the profile rules in config.DBConfig.Validate.
func (f *addFlags) validateDBType() error {
	switch f.dbType {
	case "", addKindMySQL, addKindPostgres, addKindSupabase:
	case "postgresql":
		f.dbType = addKindPostgres
	default:
		return fmt.Errorf("invalid --type %q: must be mysql, postgres, or supabase", f.dbType)
	}
	if f.redashURL != "" && (f.dbType != "" || f.url != "" || f.sslmode != "") {
		return fmt.Errorf("--redash-url cannot be combined with --type, --url, or --sslmode")
	}
	if f.url != "" && f.dbType == addKindMySQL {
		return fmt.Errorf("--url is only supported for PostgreSQL and Supabase connections")
	}
	if f.dbType == "" && f.url != "" {
		f.dbType = addKindPostgres
	}
	partial := config.DBConfig{Type: savedDBType(f.dbType), Driver: f.driver, SSLMode: f.sslmode}
	return partial.Validate()
}

// savedDBType returns the DBConfig type saved for an add kind. MySQL is saved
// without a type so its YAML shape is unchanged.
func savedDBType(kind string) string {
	if kind == addKindPostgres || kind == addKindSupabase {
		return config.DBTypePostgres
	}
	return ""
}

// hasDBFlags reports whether any direct-DB flag was given, in which case the
// database type defaults to MySQL instead of being asked interactively.
func (f *addFlags) hasDBFlags() bool {
	return f.dbHost != "" || f.dbPort >= 0 || f.dbUser != "" || f.dbName != "" || f.driver != ""
}

// resolveAddKind decides the database kind and parses the PostgreSQL
// connection string (--url, or pasted for the Supabase preset). askedKind
// reports whether the kind was chosen at the interactive prompt; pc is nil
// when no connection string was given.
func resolveAddKind(r *bufio.Reader, flags *addFlags) (kind string, askedKind bool, pc *pgConnString, err error) {
	// Database type: asked first in fully interactive mode. Direct-DB flags
	// without --type/--url keep meaning MySQL so existing scripts are unchanged.
	kind = flags.dbType
	if kind == "" {
		if flags.hasDBFlags() || flags.sshHost != "" {
			kind = addKindMySQL
		} else {
			kind = askDBKind(r)
			askedKind = true
		}
	}

	if flags.url != "" {
		pc, err = parsePostgresURL(flags.url)
		if err != nil {
			return "", false, nil, fmt.Errorf("--url: %w", err)
		}
		if pc.hasPassword {
			fmt.Fprintln(os.Stderr, i18n.T(i18n.AddURLPasswordWarning))
		}
	} else if kind == addKindSupabase && !flags.hasDBFlags() {
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddSupabaseURLHelp1))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddSupabaseURLHelp2))
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddSupabaseURLHelp3))
		raw := askValidated(r, i18n.T(i18n.AddSupabaseURL), func(s string) error {
			_, err := parsePostgresURL(s)
			return err
		})
		pc, _ = parsePostgresURL(raw)
	}
	if pc != nil {
		for _, key := range pc.ignored {
			fmt.Fprintf(os.Stderr, i18n.T(i18n.AddURLIgnoredParam)+"\n", key)
		}
	}
	return kind, askedKind, pc, nil
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

	kind, askedKind, pc, err := resolveAddKind(r, flags)
	if err != nil {
		return err
	}
	isPostgres := kind != addKindMySQL
	dbType := savedDBType(kind)
	label := config.TypeLabel(dbType)

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
	// Explicit --db-* flags take precedence over values from the connection string.
	var urlHost, urlUser, urlDB string
	if pc != nil {
		urlHost, urlUser, urlDB = pc.host, pc.user, pc.database
	}
	dbHost := askIfEmptyDefault(r, firstNonEmpty(flags.dbHost, urlHost), fmt.Sprintf(i18n.T(i18n.AddDBHost), label), defaultHost)
	var dbPort int
	switch {
	case flags.dbPort >= 0:
		dbPort = flags.dbPort
	case pc != nil:
		dbPort = pc.port
	default:
		dbPort = askInt(r, fmt.Sprintf(i18n.T(i18n.AddDBPort), label), config.DefaultPort(dbType))
	}
	dbUser := askIfEmpty(r, firstNonEmpty(flags.dbUser, urlUser), fmt.Sprintf(i18n.T(i18n.AddDBUser), label), "")

	var dbPass string
	if pc != nil && pc.hasPassword {
		dbPass = pc.password
	} else {
		if pc != nil {
			fmt.Fprintln(os.Stderr, i18n.T(i18n.AddURLNoPassword))
		}
		fmt.Fprintf(os.Stderr, i18n.T(i18n.AddDBPassword), label)
		dbPass, err = readPassword()
		if err != nil {
			return err
		}
	}

	dbName := askIfEmpty(r, firstNonEmpty(flags.dbName, urlDB), i18n.T(i18n.AddDatabaseName), "")

	// SSL mode (PostgreSQL only). An empty value is saved for the generic
	// default so the profile follows DefaultSSLMode.
	var sslmode string
	if isPostgres {
		sslmode = flags.sslmode
		if sslmode == "" && pc != nil {
			sslmode = pc.sslmode
		}
		if sslmode == "" {
			switch {
			case kind == addKindSupabase:
				sslmode = supabaseSSLMode
			case askedKind:
				sslmode = askSSLMode(r, config.DefaultSSLMode)
			}
		}
	}

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

	// Driver selection (PostgreSQL always uses the native driver)
	var driver string
	if isPostgres {
		if flags.driver != "" && flags.driver != config.DriverNative {
			return fmt.Errorf("invalid driver %q for PostgreSQL: only native is supported", flags.driver)
		}
	} else if flags.driver != "" {
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
			Type:     dbType,
			SSLMode:  sslmode,
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

	if err := conn.DB.Validate(); err != nil {
		return err
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
			choice := askFixChoice(r, conn.SSH != nil, config.TypeLabel(conn.DB.Type))
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

		c := execCommand("mysql", mysqlArgs...)
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

// askFixChoice shows the fix menu after a failed connection test. label is the
// database product name ("MySQL" or "PostgreSQL").
func askFixChoice(r *bufio.Reader, hasSSH bool, label string) string {
	fmt.Fprintln(os.Stderr, "\n"+i18n.T(i18n.AddFixTitle))
	fmt.Fprintf(os.Stderr, i18n.T(i18n.AddFixDBHostPort)+"\n", label)
	fmt.Fprintf(os.Stderr, i18n.T(i18n.AddFixDBUserPass)+"\n", label)
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
	label := config.TypeLabel(conn.DB.Type)
	switch choice {
	case "db-host":
		conn.DB.Host = askEdit(r, fmt.Sprintf(i18n.T(i18n.AddDBHost), label), conn.DB.Host)
		conn.DB.Port = askIntEdit(r, fmt.Sprintf(i18n.T(i18n.AddDBPort), label), conn.DB.Port)
	case "db-auth":
		conn.DB.User = askEdit(r, fmt.Sprintf(i18n.T(i18n.AddDBUser), label), conn.DB.User)
		fmt.Fprintf(os.Stderr, i18n.T(i18n.AddDBPasswordKeep), label)
		newPass, err := readPassword()
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

// keychainGet retrieves a cached master password from the OS credential store.
// It is a package-level variable so tests can avoid touching the real keychain.
var keychainGet = keychain.Get

func getMasterPassword() ([]byte, error) {
	// Try the OS credential store first (macOS Keychain / Windows Credential
	// Manager; silently ignored on unsupported platforms).
	if cached, err := keychainGet(); err == nil && cached != "" {
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
		pass, err := readPassword()
		if err != nil {
			return nil, err
		}
		if pass == "" {
			return nil, errors.New(i18n.T(i18n.ErrMasterEmpty))
		}
		fmt.Fprint(os.Stderr, i18n.T(i18n.AddMasterConfirm))
		confirm, err := readPassword()
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
	pass, err := readPassword()
	if err != nil {
		return nil, err
	}
	if err := crypto.VerifyMasterPassword([]byte(pass)); err != nil {
		return nil, err
	}
	saveToKeychain(pass)
	return []byte(pass), nil
}

// keychainSet stores the master password in the OS credential store. It is a
// package-level variable so tests can avoid writing to the real keychain.
var keychainSet = keychain.Set

func saveToKeychain(password string) {
	if err := keychainSet(password); err == nil {
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

// askDBKind asks which kind of database the new connection targets.
func askDBKind(r *bufio.Reader) string {
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddDBTypeTitle))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddDBTypeMySQL))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddDBTypePostgres))
	fmt.Fprintln(os.Stderr, i18n.T(i18n.AddDBTypeSupabase))
	for {
		switch strings.ToLower(ask(r, i18n.T(i18n.AddChoice), "1")) {
		case "1", addKindMySQL:
			return addKindMySQL
		case "2", addKindPostgres, "postgresql":
			return addKindPostgres
		case "3", addKindSupabase:
			return addKindSupabase
		}
		fmt.Fprintln(os.Stderr, i18n.T(i18n.AddDBTypeInvalid))
	}
}

// askSSLMode asks for a PostgreSQL sslmode, re-asking until it is valid.
func askSSLMode(r *bufio.Reader, defaultVal string) string {
	for {
		mode := ask(r, i18n.T(i18n.AddSSLMode), defaultVal)
		if config.ValidSSLMode(mode) {
			return mode
		}
		fmt.Fprintf(os.Stderr, "  invalid sslmode %q: must be one of %s\n", mode, strings.Join(config.SSLModes, ", "))
	}
}

// firstNonEmpty returns the first non-empty string.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
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
		apiKey, err = readPassword()
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
