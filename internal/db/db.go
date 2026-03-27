package db

import (
	"database/sql"
	"fmt"
	"strings"

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
			if val == nil {
				row[i] = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					row[i] = string(v)
				default:
					row[i] = fmt.Sprintf("%v", v)
				}
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return columns, result, nil
}

// Exec executes a SQL statement that doesn't return rows.
func Exec(conn *sql.DB, query string) (sql.Result, error) {
	return conn.Exec(query)
}

// Ping tests the connection.
func Ping(conn *sql.DB) error {
	return conn.Ping()
}

// SplitStatements splits a SQL string into individual statements by semicolons,
// respecting single-quoted, double-quoted, and backtick-quoted strings,
// as well as line comments (--) and block comments (/* */). Empty statements are skipped.
func SplitStatements(sqlText string) []string {
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
