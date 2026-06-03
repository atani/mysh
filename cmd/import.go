package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/crypto"
	"github.com/atani/mysh/internal/i18n"
	"github.com/atani/mysh/internal/importer"
)

func RunImport(args []string) error {
	opts, err := parseImportFlags(args)
	if err != nil {
		return err
	}

	provider, ok := importer.Get(opts.from)
	if !ok {
		return fmt.Errorf("unknown source %q (available: %s)", opts.from, strings.Join(importer.Available(), ", "))
	}

	var conns []importer.ImportedConnection
	if fp, ok := provider.(importer.FileProvider); ok && opts.file != "" {
		conns, err = fp.DiscoverFromFile(opts.file)
	} else if opts.file != "" {
		return fmt.Errorf("source %q does not support --file", opts.from)
	} else {
		conns, err = provider.Discover()
	}
	if err != nil {
		return err
	}
	if len(conns) == 0 {
		fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportNoConnections)+"\n", provider.Name())
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Display discovered connections
	fmt.Fprintf(os.Stderr, "\n%s: %d connection(s) found\n\n", provider.Name(), len(conns))
	w := tabwriter.NewWriter(os.Stderr, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "  #\tNAME\tHOST\tPORT\tUSER\tDATABASE\tSSH")
	for i, c := range conns {
		ssh := "-"
		if c.SSH != nil {
			if c.SSH.User != "" {
				ssh = fmt.Sprintf("%s@%s", c.SSH.User, c.SSH.Host)
			} else {
				ssh = c.SSH.Host
			}
		}
		folder := ""
		if c.Folder != "" {
			folder = " [" + c.Folder + "]"
		}
		fmt.Fprintf(w, "  %d\t%s%s\t%s\t%d\t%s\t%s\t%s\n",
			i+1, c.Name, folder, c.DB.Host, c.DB.Port, c.DB.User, c.DB.Database, ssh)
	}
	_ = w.Flush()
	fmt.Fprintln(os.Stderr)

	// Select connections
	r := bufio.NewReader(os.Stdin)
	var selected []importer.ImportedConnection

	if opts.all {
		selected = conns
	} else {
		indices, err := askSelection(r, len(conns))
		if err != nil {
			return err
		}
		if len(indices) == 0 {
			fmt.Fprintln(os.Stderr, "No connections selected.")
			return nil
		}
		for _, idx := range indices {
			selected = append(selected, conns[idx])
		}
	}

	// Import each selected connection
	usedNames := make(map[string]bool)
	for _, existing := range cfg.Connections {
		usedNames[existing.Name] = true
	}

	var importedNames []string
	var sharedDBPasswordFailures []string
	sharedDBPassword, sharedDBPasswordSet, err := readSharedDBPassword(r, provider.Name(), selected, opts)
	if err != nil {
		return err
	}

	for _, ic := range selected {
		fmt.Fprintf(os.Stderr, "\n--- %s ---\n", ic.Name)

		sharedPasswordFailed := false

		// Resolve name
		name := ic.Name
		if usedNames[name] {
			fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportNameConflict)+"\n", name)
			name = askValidated(r, "Connection name", func(s string) error {
				if s == "" {
					return fmt.Errorf("name is required")
				}
				if usedNames[s] {
					return fmt.Errorf("connection %q already exists", s)
				}
				return nil
			})
		}

		env := ic.Env
		if env == "" {
			env = "development"
		}

		conn := config.Connection{
			Name:   name,
			Env:    env,
			SSH:    ic.SSH,
			Mask:   ic.Mask,
			Redash: ic.Redash,
		}

		if isRedashConnection(ic) {
			// Redash connection: prompt for API key
			fmt.Fprintf(os.Stderr, "Redash API key for %s: ", ic.Redash.URL)
			apiKey, err := readPassword()
			if err != nil {
				return err
			}
			if apiKey != "" {
				masterPass, err := getMasterPassword()
				if err != nil {
					return err
				}
				enc, err := crypto.Encrypt([]byte(apiKey), masterPass)
				if err != nil {
					return fmt.Errorf("encrypting API key: %w", err)
				}
				conn.Redash.APIKey, err = crypto.MarshalEncrypted(enc)
				if err != nil {
					return fmt.Errorf("encoding encrypted API key: %w", err)
				}
			}
		} else {
			// Direct DB connection: resolve user/password (and SSH user if missing).
			if ic.SSH != nil && ic.SSH.User == "" {
				ic.SSH.User = askRequired(r, fmt.Sprintf("SSH user for %s", ic.SSH.Host))
			}

			driver := ic.DB.Driver
			if driver == "" {
				driver = config.DriverCLI
			}
			conn.DB = config.DBConfig{
				Host:     ic.DB.Host,
				Port:     ic.DB.Port,
				User:     resolveImportDBUser(r, ic.Name, ic.DB.User, opts),
				Database: ic.DB.Database,
				Driver:   driver,
			}

			if sharedDBPasswordSet {
				conn.DB.Password = sharedDBPassword
				if conn.DB.Password != "" {
					// The shared password is entered once and cannot be re-prompted
					// per connection, so a failure here is collected and reported in
					// the final summary instead of retried.
					if err := testConnection(&conn); err != nil {
						fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportConnFailed)+"\n", err)
						sharedPasswordFailed = true
					}
				}
			} else {
				fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportPasswordPrompt)+"\n", provider.Name())
				if err := promptForDBPassword(r, &conn); err != nil {
					return err
				}
			}
		}

		if err := cfg.Add(conn); err != nil {
			fmt.Fprintf(os.Stderr, "  Skipping: %v\n", err)
			continue
		}
		if err := config.Save(cfg); err != nil {
			return err
		}
		usedNames[name] = true
		importedNames = append(importedNames, name)
		if sharedPasswordFailed {
			sharedDBPasswordFailures = append(sharedDBPasswordFailures, name)
		}
		fmt.Fprintf(os.Stderr, "  Added %q.\n", name)
	}

	if len(importedNames) == 0 {
		fmt.Fprintln(os.Stderr, "\nNo connections imported.")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n"+i18n.T(i18n.ImportSuccess)+"\n", len(importedNames), provider.Name())

	// Ask about default mask settings (skip if all imported connections already have mask config)
	allHaveMask := true
	for _, name := range importedNames {
		conn := cfg.Find(name)
		if conn != nil && !conn.HasMaskConfig() {
			allHaveMask = false
			break
		}
	}

	if !allHaveMask {
		fmt.Fprintln(os.Stderr)
		defaultMask := "email,phone,*password*,*secret*,*token*,*address*"
		fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportMaskAsk)+"\n", defaultMask)
		if askYesNo(r, i18n.T(i18n.ImportMaskPrompt), true) {
			for _, name := range importedNames {
				conn := cfg.Find(name)
				if conn == nil || conn.HasMaskConfig() {
					continue
				}
				conn.Env = "production"
				conn.Mask = parseMaskInput(defaultMask)
			}
			fmt.Fprintln(os.Stderr, i18n.T(i18n.ImportMaskApplied))
		} else {
			fmt.Fprintln(os.Stderr)
			fmt.Fprintln(os.Stderr, i18n.T(i18n.ImportPostHint))
			fmt.Fprintln(os.Stderr)
			for _, name := range importedNames {
				fmt.Fprintf(os.Stderr, "  mysh edit %s\n", name)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	if len(sharedDBPasswordFailures) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportSharedPasswordFailed)+"\n", len(sharedDBPasswordFailures))
		for _, name := range sharedDBPasswordFailures {
			fmt.Fprintf(os.Stderr, "  mysh edit %s\n", name)
		}
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, i18n.T(i18n.ImportPingHint))
	return nil
}

type importOptions struct {
	from          string
	file          string
	all           bool
	askUser       bool
	dbUser        string
	reusePassword bool
}

func parseImportFlags(args []string) (importOptions, error) {
	var opts importOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--from requires a value")
			}
			i++
			opts.from = args[i]
		case "--file":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--file requires a value")
			}
			i++
			opts.file = args[i]
		case "--all":
			opts.all = true
		case "--ask-user":
			opts.askUser = true
		case "--db-user":
			if i+1 >= len(args) {
				return opts, fmt.Errorf("--db-user requires a value")
			}
			i++
			opts.dbUser = strings.TrimSpace(args[i])
			if opts.dbUser == "" {
				return opts, fmt.Errorf("--db-user requires a non-empty value")
			}
		case "--reuse-password":
			opts.reusePassword = true
		default:
			return opts, fmt.Errorf("unknown flag: %s", args[i])
		}
	}
	if opts.from == "" {
		return opts, fmt.Errorf("--from is required (available: %s)", strings.Join(importer.Available(), ", "))
	}
	if opts.askUser && opts.dbUser != "" {
		return opts, fmt.Errorf("--ask-user and --db-user cannot be used together")
	}
	return opts, nil
}

func resolveImportDBUser(input io.Reader, connName, current string, opts importOptions) string {
	if opts.dbUser != "" {
		return opts.dbUser
	}
	if !opts.askUser {
		return current
	}
	r, ok := input.(*bufio.Reader)
	if !ok {
		r = bufio.NewReader(input)
	}
	return ask(r, fmt.Sprintf("DB user for %s", connName), current)
}

func readSharedDBPassword(r *bufio.Reader, sourceName string, selected []importer.ImportedConnection, opts importOptions) (string, bool, error) {
	if !opts.reusePassword || !hasDBConnections(selected) {
		return "", false, nil
	}
	fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportPasswordPrompt)+"\n", sourceName)
	for {
		fmt.Fprint(os.Stderr, i18n.T(i18n.ImportSharedPasswordInput))
		dbPass, err := readPassword()
		if err != nil {
			return "", false, err
		}
		if dbPass == "" {
			if !askYesNo(r, i18n.T(i18n.ImportAddNoPassword), true) {
				continue
			}
			return "", true, nil
		}
		password, err := encryptImportedSecret(dbPass)
		if err != nil {
			return "", false, err
		}
		return password, true, nil
	}
}

// isRedashConnection reports whether ic is a Redash connection (as opposed to a
// direct DB connection). The two cases are mutually exclusive in import flow, so
// keeping this single predicate avoids the two branches drifting apart.
func isRedashConnection(ic importer.ImportedConnection) bool {
	return ic.Redash != nil && ic.Redash.URL != ""
}

func hasDBConnections(conns []importer.ImportedConnection) bool {
	for _, c := range conns {
		if !isRedashConnection(c) {
			return true
		}
	}
	return false
}

func promptForDBPassword(r *bufio.Reader, conn *config.Connection) error {
	const maxRetries = 2
	attempt := 0
	for {
		if attempt == 0 {
			fmt.Fprint(os.Stderr, i18n.T(i18n.ImportPasswordInput))
		} else {
			fmt.Fprint(os.Stderr, i18n.T(i18n.ImportPasswordRetry))
		}
		dbPass, err := readPassword()
		if err != nil {
			return err
		}

		if dbPass == "" {
			if conn.DB.Password != "" {
				break
			}
			if !askYesNo(r, i18n.T(i18n.ImportAddNoPassword), true) {
				attempt = 0
				continue
			}
			break
		}

		password, err := encryptImportedSecret(dbPass)
		if err != nil {
			return err
		}
		conn.DB.Password = password

		if err := testConnection(conn); err != nil {
			fmt.Fprintf(os.Stderr, i18n.T(i18n.ImportConnFailed)+"\n", err)
			attempt++
			if attempt <= maxRetries {
				fmt.Fprintln(os.Stderr, i18n.T(i18n.ImportRetryHint))
				continue
			}
			fmt.Fprintln(os.Stderr, i18n.T(i18n.ImportRetryExhausted))
		}
		break
	}
	return nil
}

func encryptImportedSecret(secret string) (string, error) {
	masterPass, err := getMasterPassword()
	if err != nil {
		return "", err
	}
	enc, err := crypto.Encrypt([]byte(secret), masterPass)
	if err != nil {
		return "", fmt.Errorf("encrypting password: %w", err)
	}
	encoded, err := crypto.MarshalEncrypted(enc)
	if err != nil {
		return "", fmt.Errorf("encoding encrypted password: %w", err)
	}
	return encoded, nil
}

func askSelection(r *bufio.Reader, total int) ([]int, error) {
	input := ask(r, "Select connections (comma-separated numbers, or 'all')", "all")
	input = strings.TrimSpace(input)

	if strings.ToLower(input) == "all" {
		indices := make([]int, total)
		for i := range indices {
			indices[i] = i
		}
		return indices, nil
	}

	var indices []int
	seen := make(map[int]bool)
	for _, part := range strings.Split(input, ",") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > total {
			return nil, fmt.Errorf("invalid selection: %s (enter 1-%d)", s, total)
		}
		idx := n - 1
		if !seen[idx] {
			indices = append(indices, idx)
			seen[idx] = true
		}
	}
	return indices, nil
}
