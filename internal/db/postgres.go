package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// PostgresParams holds the settings for a PostgreSQL connection. Every value
// that reaches the driver comes from here: libpq environment variables (PG*),
// ~/.pgpass and service files are never consulted.
type PostgresParams struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
	// TLSServerName, when set, is used for TLS certificate verification and
	// SNI instead of Host. It carries the original server name when Host has
	// been rewritten to the local end of an SSH tunnel.
	TLSServerName string
}

// postgresConnectTimeout bounds how long a connection attempt may take, in seconds.
const postgresConnectTimeout = 10

// OpenPostgres opens a PostgreSQL connection through pgx's database/sql driver.
func OpenPostgres(p PostgresParams) (*sql.DB, error) {
	cfg, err := postgresConfig(p)
	if err != nil {
		return nil, err
	}
	conn := stdlib.OpenDB(*cfg)
	conn.SetMaxOpenConns(1)
	return conn, nil
}

// postgresConfig builds the pgx configuration for p.
//
// pgx's ParseConfig merges libpq environment variables and files underneath
// the connection string, so a value left out of the connection string can be
// supplied by the caller's environment. To keep the connection target, TLS
// behavior and credentials determined solely by the mysh profile, every
// setting ParseConfig reads from the environment is written explicitly here,
// and runtime parameters are replaced afterwards.
//
// Queries always use the simple protocol: Supabase's transaction pooler
// (Supavisor, port 6543) does not support prepared statements.
func postgresConfig(p PostgresParams) (*pgx.ConnConfig, error) {
	sslmode := p.SSLMode
	if sslmode == "" {
		sslmode = "prefer"
	}
	settings := [][2]string{
		{"host", p.Host},
		{"port", strconv.Itoa(p.Port)},
		{"user", p.User},
		{"password", p.Password},
		{"dbname", p.Database},
		{"sslmode", sslmode},
		{"passfile", ""},
		{"sslcert", ""},
		{"sslkey", ""},
		{"sslrootcert", ""},
		{"sslpassword", ""},
		{"sslcertmode", "allow"},
		{"sslsni", "1"},
		{"sslnegotiation", "postgres"},
		{"target_session_attrs", "any"},
		{"connect_timeout", strconv.Itoa(postgresConnectTimeout)},
		{"min_protocol_version", "3.0"},
		{"max_protocol_version", "3.0"},
		{"channel_binding", "prefer"},
		{"require_auth", ""},
	}
	var b strings.Builder
	for i, kv := range settings {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(kv[0])
		b.WriteByte('=')
		b.WriteString(quoteConnValue(kv[1]))
	}

	cfg, err := pgx.ParseConfig(b.String())
	if err != nil {
		// The parse error embeds the connection string, which includes the
		// password. Report only the underlying cause.
		msg := "invalid PostgreSQL connection settings"
		if inner := errors.Unwrap(err); inner != nil {
			return nil, fmt.Errorf("%s: %w", msg, inner)
		}
		return nil, errors.New(msg)
	}

	// PGOPTIONS, PGAPPNAME, PGTZ and service-file entries arrive as runtime
	// parameters; discard them so the session starts with server defaults.
	cfg.RuntimeParams = map[string]string{"application_name": "mysh"}
	cfg.KerberosSrvName = ""
	cfg.KerberosSpn = ""
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	if p.TLSServerName != "" {
		if cfg.TLSConfig != nil {
			cfg.TLSConfig.ServerName = p.TLSServerName
		}
		for _, fb := range cfg.Fallbacks {
			if fb.TLSConfig != nil {
				fb.TLSConfig.ServerName = p.TLSServerName
			}
		}
	}
	return cfg, nil
}

// quoteConnValue quotes a value for a PostgreSQL keyword/value connection string.
func quoteConnValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}
