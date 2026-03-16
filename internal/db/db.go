package db

import (
	"database/sql"
	"fmt"

	gomysql "github.com/go-sql-driver/mysql"
)

// Open opens a MySQL connection using go-sql-driver/mysql with old_password support.
func Open(host string, port int, user, password, database string) (*sql.DB, error) {
	cfg := gomysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", host, port)
	cfg.DBName = database
	cfg.AllowOldPasswords = true
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
	defer rows.Close()

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
