package db

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	gomysql "github.com/go-sql-driver/mysql"
)

// Open opens a MySQL connection using go-sql-driver/mysql.
// If allowOldPasswords is true, the pre-4.1 old_password authentication is enabled.
// This should only be used for legacy MySQL 4.x connections.
func Open(host string, port int, user, password, database string, allowOldPasswords bool) (*sql.DB, error) {
	cfg := gomysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", host, port)
	cfg.DBName = database
	cfg.AllowOldPasswords = allowOldPasswords
	cfg.AllowNativePasswords = true

	conn, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("opening connection: %w", err)
	}
	conn.SetMaxOpenConns(1)
	return conn, nil
}

// Query executes a SQL query and returns headers and rows.
func Query(conn *sql.DB, query string) ([]string, [][]string, error) {
	rows, err := conn.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("reading columns: %w", err)
	}

	// Non-SELECT statements return an empty column list
	if len(columns) == 0 {
		return nil, nil, nil
	}

	typeNames := make([]string, len(columns))
	if colTypes, err := rows.ColumnTypes(); err == nil && len(colTypes) == len(columns) {
		for i, ct := range colTypes {
			typeNames[i] = ct.DatabaseTypeName()
		}
	}

	var result [][]string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("scanning row: %w", err)
		}

		row := make([]string, len(columns))
		for i, val := range values {
			row[i] = formatValue(val, typeNames[i])
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return columns, result, nil
}

// formatValue renders a scanned column value as text. typeName is the
// driver-reported column type and may be empty.
//
// go-sql-driver/mysql returns every value as []byte (or nil), so MySQL output is
// unaffected by the PostgreSQL-specific cases: pgx returns date/time columns as
// time.Time and bytea as raw bytes, which are rendered the way psql shows them.
func formatValue(val interface{}, typeName string) string {
	switch v := val.(type) {
	case nil:
		return "NULL"
	case []byte:
		if typeName == "BYTEA" {
			return `\x` + hex.EncodeToString(v)
		}
		return string(v)
	case time.Time:
		switch typeName {
		case "DATE":
			return v.Format("2006-01-02")
		case "TIMESTAMP":
			return v.Format("2006-01-02 15:04:05.999999")
		default:
			return v.Format("2006-01-02 15:04:05.999999-07:00")
		}
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Exec executes a SQL statement that doesn't return rows.
func Exec(conn *sql.DB, query string) (sql.Result, error) {
	return conn.Exec(query)
}

// Ping tests the connection.
func Ping(conn *sql.DB) error {
	return conn.Ping()
}

// Dialect selects the lexical rules SplitStatements applies.
type Dialect int

const (
	// DialectMySQL: backslash escapes inside '...' and "...", backtick identifiers.
	DialectMySQL Dialect = iota
	// DialectPostgres: standard-conforming strings, E'...' escape strings,
	// "..." identifiers, dollar quoting ($$...$$, $tag$...$tag$) and nested
	// block comments.
	DialectPostgres
)

// SplitStatements splits a SQL string into individual statements by semicolons
// that appear outside quoted text and comments, using the lexical rules of the
// given dialect. Empty statements are skipped.
func SplitStatements(sqlText string, dialect Dialect) []string {
	if dialect == DialectPostgres {
		return splitPostgres(sqlText)
	}
	return splitMySQL(sqlText)
}

// endProbe is appended by EndsWithTerminator; it is a plain identifier so it
// cannot open or close any quote or comment.
const endProbe = "mysh_end_probe"

// EndsWithTerminator reports whether sqlText ends with a ';' that is outside
// quoted text, dollar quotes and comments, i.e. the input is complete and can
// be executed. It reuses SplitStatements so both share the lexical rules: a
// statement appended after the input comes back on its own only when nothing
// before it is left open and no text follows the last terminator.
func EndsWithTerminator(sqlText string, dialect Dialect) bool {
	if strings.TrimSpace(sqlText) == "" {
		return false
	}
	stmts := SplitStatements(sqlText+"\n"+endProbe, dialect)
	return len(stmts) > 0 && stmts[len(stmts)-1] == endProbe
}

// splitPostgres implements SplitStatements for PostgreSQL.
func splitPostgres(sqlText string) []string {
	var stmts []string
	var current strings.Builder
	runes := []rune(sqlText)
	n := len(runes)

	flush := func() {
		if stmt := strings.TrimSpace(current.String()); stmt != "" {
			stmts = append(stmts, stmt)
		}
		current.Reset()
	}

	for i := 0; i < n; i++ {
		ch := runes[i]
		switch {
		case ch == '-' && i+1 < n && runes[i+1] == '-':
			// Line comment: copy through the end of the line.
			for i < n && runes[i] != '\n' {
				current.WriteRune(runes[i])
				i++
			}
			if i < n {
				current.WriteRune(runes[i])
			}
		case ch == '/' && i+1 < n && runes[i+1] == '*':
			// Block comment; PostgreSQL allows nesting.
			depth := 0
			for i < n {
				if runes[i] == '/' && i+1 < n && runes[i+1] == '*' {
					depth++
					current.WriteString("/*")
					i += 2
					continue
				}
				if runes[i] == '*' && i+1 < n && runes[i+1] == '/' {
					depth--
					current.WriteString("*/")
					i += 2
					if depth == 0 {
						break
					}
					continue
				}
				current.WriteRune(runes[i])
				i++
			}
			i-- // the loop increment moves past the comment
		case ch == '\'':
			escapes := isEscapeStringPrefix(runes, i)
			current.WriteRune(ch)
			i++
			for i < n {
				c := runes[i]
				current.WriteRune(c)
				if escapes && c == '\\' && i+1 < n {
					current.WriteRune(runes[i+1])
					i += 2
					continue
				}
				if c == '\'' {
					break
				}
				i++
			}
		case ch == '"':
			current.WriteRune(ch)
			i++
			for i < n && runes[i] != '"' {
				current.WriteRune(runes[i])
				i++
			}
			if i < n {
				current.WriteRune(runes[i])
			}
		case ch == '$' && (i == 0 || !isIdentRune(runes[i-1])):
			tag, ok := dollarTag(runes, i)
			if !ok {
				current.WriteRune(ch)
				continue
			}
			// Copy the opening tag, the body and the closing tag.
			current.WriteString(tag)
			i += len([]rune(tag))
			closing := []rune(tag)
			for i < n && !hasRunesAt(runes, i, closing) {
				current.WriteRune(runes[i])
				i++
			}
			if i < n {
				current.WriteString(tag)
				i += len(closing)
			}
			i--
		case ch == ';':
			flush()
		default:
			current.WriteRune(ch)
		}
	}
	flush()
	return stmts
}

// dollarTag returns the dollar-quote delimiter ("$$" or "$tag$") starting at
// runes[i], or false when the '$' does not open a dollar quote (for example a
// positional parameter such as $1).
func dollarTag(runes []rune, i int) (string, bool) {
	j := i + 1
	for j < len(runes) && runes[j] != '$' {
		r := runes[j]
		if r == '_' || unicode.IsLetter(r) || (j > i+1 && unicode.IsDigit(r)) {
			j++
			continue
		}
		return "", false
	}
	if j >= len(runes) {
		return "", false
	}
	return string(runes[i : j+1]), true
}

// isEscapeStringPrefix reports whether the quote at runes[i] opens an E'...'
// escape string, where backslash escapes apply. The E must stand alone: in
// name'...' the e ends an identifier and the string is a standard one.
func isEscapeStringPrefix(runes []rune, i int) bool {
	if i == 0 || (runes[i-1] != 'E' && runes[i-1] != 'e') {
		return false
	}
	return i < 2 || !isIdentRune(runes[i-2])
}

// isIdentRune reports whether r can appear inside an unquoted PostgreSQL identifier.
func isIdentRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func hasRunesAt(runes []rune, i int, want []rune) bool {
	if i+len(want) > len(runes) {
		return false
	}
	for k, r := range want {
		if runes[i+k] != r {
			return false
		}
	}
	return true
}

// splitMySQL implements SplitStatements for MySQL: it respects single-quoted,
// double-quoted, and backtick-quoted strings, as well as line comments (--) and
// block comments (/* */).
func splitMySQL(sqlText string) []string {
	var stmts []string
	var current strings.Builder
	runes := []rune(sqlText)
	n := len(runes)
	inSingleQuote := false
	inDoubleQuote := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for i := 0; i < n; i++ {
		ch := runes[i]

		// Handle line comment end
		if inLineComment {
			current.WriteRune(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}

		// Handle block comment end
		if inBlockComment {
			current.WriteRune(ch)
			if ch == '*' && i+1 < n && runes[i+1] == '/' {
				current.WriteRune(runes[i+1])
				i++
				inBlockComment = false
			}
			continue
		}

		// Handle escaped characters inside quotes
		if escaped {
			current.WriteRune(ch)
			escaped = false
			continue
		}
		if ch == '\\' && (inSingleQuote || inDoubleQuote) {
			current.WriteRune(ch)
			escaped = true
			continue
		}

		// Handle quote toggling
		if ch == '\'' && !inDoubleQuote && !inBacktick {
			inSingleQuote = !inSingleQuote
			current.WriteRune(ch)
			continue
		}
		if ch == '"' && !inSingleQuote && !inBacktick {
			inDoubleQuote = !inDoubleQuote
			current.WriteRune(ch)
			continue
		}
		if ch == '`' && !inSingleQuote && !inDoubleQuote {
			inBacktick = !inBacktick
			current.WriteRune(ch)
			continue
		}

		inQuote := inSingleQuote || inDoubleQuote || inBacktick

		// Detect comment start (only outside quotes)
		if !inQuote {
			if ch == '-' && i+1 < n && runes[i+1] == '-' {
				inLineComment = true
				current.WriteRune(ch)
				continue
			}
			if ch == '/' && i+1 < n && runes[i+1] == '*' {
				inBlockComment = true
				current.WriteRune(ch)
				current.WriteRune(runes[i+1])
				i++
				continue
			}
		}

		// Statement separator
		if ch == ';' && !inQuote {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteRune(ch)
	}

	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		stmts = append(stmts, stmt)
	}

	return stmts
}
