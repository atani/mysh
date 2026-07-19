package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atani/mysh/internal/config"
	"gopkg.in/yaml.v3"
)

// ExportedConnection is the YAML format for sharing connections.
// Passwords and API keys are always omitted so the file can be safely shared.
type ExportedConnection struct {
	Name   string             `yaml:"name"`
	Env    string             `yaml:"env,omitempty"`
	SSH    *config.SSHConfig  `yaml:"ssh,omitempty"`
	DB     *ExportedDBConfig  `yaml:"db,omitempty"`
	Redash *ExportedRedash    `yaml:"redash,omitempty"`
	Mask   *config.MaskConfig `yaml:"mask,omitempty"`
}

type ExportedDBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`
	Driver   string `yaml:"driver,omitempty"`
}

type ExportedRedash struct {
	URL          string `yaml:"url"`
	DataSourceID int    `yaml:"data_source_id"`
}

// ExportedQuery is the YAML form of a saved SQL query bundled by
// `mysh export --with-queries`. Saved queries are plain .sql files under
// QueriesDir, so the only information available is the file's base name (used as
// the query name) and its contents (the query text). Unlike connections, saved
// queries have no per-connection association in this codebase, so they cannot be
// filtered by connection name.
type ExportedQuery struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

// exportBundle is the top-level YAML shape used when --with-queries is set.
// Without the flag, export keeps emitting a bare connection list (see RunExport)
// so existing import tooling that expects a top-level list stays compatible.
type exportBundle struct {
	Connections []ExportedConnection `yaml:"connections"`
	Queries     []ExportedQuery      `yaml:"queries"`
}

// buildExported converts internal connections into their shareable export form,
// omitting all secrets (passwords and API keys). Extracted from RunExport so the
// secret-stripping and Redash/DB branching can be unit-tested.
func buildExported(conns []config.Connection) []ExportedConnection {
	exported := make([]ExportedConnection, len(conns))
	for i, c := range conns {
		ec := ExportedConnection{
			Name: c.Name,
			Env:  c.Env,
			SSH:  c.SSH,
			Mask: c.Mask,
		}

		if c.IsRedash() {
			ec.Redash = &ExportedRedash{
				URL:          c.Redash.URL,
				DataSourceID: c.Redash.DataSourceID,
			}
		} else {
			ec.DB = &ExportedDBConfig{
				Host:     c.DB.Host,
				Port:     c.DB.Port,
				User:     c.DB.User,
				Database: c.DB.Database,
				Driver:   c.DB.Driver,
			}
		}

		exported[i] = ec
	}
	return exported
}

// loadSavedQueries reads the saved .sql files under QueriesDir and returns them
// as exportable queries (name = file base name without the .sql suffix, query =
// file contents). A missing queries directory yields an empty slice, not an
// error, so exporting works before any query has been saved.
func loadSavedQueries() ([]ExportedQuery, error) {
	dir := config.QueriesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading queries directory: %w", err)
	}

	var queries []ExportedQuery
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading query %s: %w", e.Name(), err)
		}
		queries = append(queries, ExportedQuery{
			Name:  strings.TrimSuffix(e.Name(), ".sql"),
			Query: string(content),
		})
	}
	return queries, nil
}

type exportOptions struct {
	name        string
	withQueries bool
}

// parseExportFlags splits the export arguments into the optional connection name
// and the --with-queries flag. Unknown flags are rejected to match the manual
// argument parsing used by the other commands.
func parseExportFlags(args []string) (exportOptions, error) {
	var opts exportOptions
	for _, arg := range args {
		switch {
		case arg == "--with-queries":
			opts.withQueries = true
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Errorf("unknown flag: %s", arg)
		default:
			if opts.name != "" {
				return opts, fmt.Errorf("unexpected argument: %s", arg)
			}
			opts.name = arg
		}
	}
	return opts, nil
}

func RunExport(args []string) error {
	opts, err := parseExportFlags(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if len(cfg.Connections) == 0 {
		return fmt.Errorf("no connections configured")
	}

	// If a name is given, export only that connection.
	var conns []config.Connection
	if opts.name != "" {
		conn := cfg.Find(opts.name)
		if conn == nil {
			return fmt.Errorf("connection %q not found", opts.name)
		}
		conns = []config.Connection{*conn}
	} else {
		conns = cfg.Connections
	}

	exported := buildExported(conns)

	// Default output is a bare connection list. When --with-queries is set, wrap
	// the connections in a bundle with a top-level queries: list. Saved queries
	// have no connection association, so all saved queries are always included,
	// even when a single connection is exported.
	var payload any = exported
	if opts.withQueries {
		queries, err := loadSavedQueries()
		if err != nil {
			return err
		}
		payload = exportBundle{Connections: exported, Queries: queries}
	}

	data, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling export: %w", err)
	}

	fmt.Fprint(os.Stdout, string(data))
	return nil
}
