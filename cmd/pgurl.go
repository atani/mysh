package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/atani/mysh/internal/config"
)

// pgConnString holds the settings taken from a postgresql:// connection string.
// The string itself is never stored; only these fields are copied into the
// connection profile (the password is encrypted like any other).
type pgConnString struct {
	host        string
	port        int
	user        string
	password    string
	hasPassword bool // false when absent or still the dashboard placeholder
	database    string
	sslmode     string
	ignored     []string // query parameter names mysh does not use
}

// supabasePasswordPlaceholder is what the Supabase dashboard shows in place of
// the database password in its Connect dialog.
const supabasePasswordPlaceholder = "[YOUR-PASSWORD]"

// parsePostgresURL parses a postgresql:// (or postgres://) connection string,
// such as the ones the Supabase dashboard's Connect dialog shows:
//
//	postgresql://postgres:[YOUR-PASSWORD]@db.<ref>.supabase.co:5432/postgres
//	postgresql://postgres.<ref>:[YOUR-PASSWORD]@<pooler-host>:5432/postgres
//	postgresql://postgres.<ref>:[YOUR-PASSWORD]@<pooler-host>:6543/postgres
//
// The user info is split by hand rather than with net/url: the dashboard's
// "[YOUR-PASSWORD]" placeholder and unescaped special characters in pasted
// passwords are not valid URL user info. Errors never include the input, since
// it may contain a password.
func parsePostgresURL(raw string) (*pgConnString, error) {
	raw = strings.TrimSpace(raw)
	var rest string
	switch {
	case strings.HasPrefix(raw, "postgresql://"):
		rest = raw[len("postgresql://"):]
	case strings.HasPrefix(raw, "postgres://"):
		rest = raw[len("postgres://"):]
	default:
		return nil, errors.New("connection string must start with postgresql:// or postgres://")
	}

	pc := &pgConnString{port: config.DefaultPostgresPort}

	// The password may contain '@', '/', '?' or ':' when pasted unescaped, so the
	// host part starts after the last '@'.
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		userinfo := rest[:at]
		rest = rest[at+1:]
		user, pass, hasPass := strings.Cut(userinfo, ":")
		decodedUser, err := url.PathUnescape(user)
		if err != nil {
			return nil, errors.New("connection string: invalid percent-encoding in user name")
		}
		pc.user = decodedUser
		if hasPass && pass != "" && pass != supabasePasswordPlaceholder {
			pc.password = decodePassword(pass)
			pc.hasPassword = true
		}
	}

	var query string
	rest, query, _ = strings.Cut(rest, "?")
	hostport, path, _ := strings.Cut(rest, "/")

	if strings.Contains(hostport, ",") {
		return nil, errors.New("connection string: multiple hosts are not supported")
	}
	host, portStr, err := splitHostPort(hostport)
	if err != nil {
		return nil, err
	}
	if host == "" {
		return nil, errors.New("connection string: host is required")
	}
	pc.host = host
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil || port <= 0 || port > 65535 {
			return nil, errors.New("connection string: invalid port")
		}
		pc.port = port
	}

	if path != "" {
		db, err := url.PathUnescape(path)
		if err != nil {
			return nil, errors.New("connection string: invalid percent-encoding in database name")
		}
		pc.database = db
	}

	if query != "" {
		values, err := url.ParseQuery(query)
		if err != nil {
			return nil, errors.New("connection string: invalid query parameters")
		}
		for key, vals := range values {
			switch key {
			case "sslmode":
				mode := vals[len(vals)-1]
				if !config.ValidSSLMode(mode) {
					return nil, fmt.Errorf("connection string: invalid sslmode %q: must be one of %s", mode, strings.Join(config.SSLModes, ", "))
				}
				pc.sslmode = mode
			case "user":
				pc.user = vals[len(vals)-1]
			case "dbname":
				pc.database = vals[len(vals)-1]
			case "password":
				return nil, errors.New("connection string: put the password in the user info (user:password@host), not in a query parameter")
			default:
				pc.ignored = append(pc.ignored, key)
			}
		}
	}
	return pc, nil
}

// decodePassword percent-decodes a password from the connection string. A
// password pasted without encoding may contain a literal '%' that does not form
// a valid escape; it is then used as written.
func decodePassword(s string) string {
	if decoded, err := url.PathUnescape(s); err == nil {
		return decoded
	}
	return s
}

// splitHostPort splits "host", "host:port", "[v6]" or "[v6]:port".
func splitHostPort(hostport string) (host, port string, err error) {
	if strings.HasPrefix(hostport, "[") {
		end := strings.Index(hostport, "]")
		if end < 0 {
			return "", "", errors.New("connection string: unterminated IPv6 address")
		}
		host = hostport[1:end]
		after := hostport[end+1:]
		if after == "" {
			return host, "", nil
		}
		if !strings.HasPrefix(after, ":") {
			return "", "", errors.New("connection string: invalid host")
		}
		return host, after[1:], nil
	}
	host, port, _ = strings.Cut(hostport, ":")
	return host, port, nil
}
