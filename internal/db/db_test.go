package db

import (
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBuildDSN(t *testing.T) {
	// Verify that Config.FormatDSN produces a valid DSN via Open's internals.
	// We can't connect to a real DB in unit tests, but we can verify
	// the function doesn't panic and returns a non-nil *sql.DB.
	conn, err := Open("127.0.0.1", 3306, "testuser", "p@ss:w0rd/special", "testdb", false)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = conn.Close() }()
	// sql.Open is lazy; it doesn't actually connect.
	// This just verifies DSN construction doesn't fail for special characters.
}

func TestQueryBasic(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(1, "Alice", "alice@example.com").
		AddRow(2, "Bob", "bob@example.com")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	headers, data, err := Query(db, "SELECT id, name, email FROM users")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(headers) != 3 {
		t.Fatalf("headers: got %d, want 3", len(headers))
	}
	if headers[0] != "id" || headers[1] != "name" || headers[2] != "email" {
		t.Errorf("headers: got %v", headers)
	}

	if len(data) != 2 {
		t.Fatalf("rows: got %d, want 2", len(data))
	}
	if data[0][1] != "Alice" {
		t.Errorf("row[0][1]: got %q, want Alice", data[0][1])
	}
	if data[1][2] != "bob@example.com" {
		t.Errorf("row[1][2]: got %q, want bob@example.com", data[1][2])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestQueryNullValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "value"}).
		AddRow(1, nil).
		AddRow(2, "present")

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, data, err := Query(db, "SELECT id, value FROM t")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if data[0][1] != "NULL" {
		t.Errorf("null value: got %q, want NULL", data[0][1])
	}
	if data[1][1] != "present" {
		t.Errorf("present value: got %q, want present", data[1][1])
	}
}

func TestQueryByteValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"data"}).
		AddRow([]byte("binary-content"))

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, data, err := Query(db, "SELECT data FROM t")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if data[0][0] != "binary-content" {
		t.Errorf("byte value: got %q, want binary-content", data[0][0])
	}
}

func TestQueryEmptyResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"id", "name"})
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	headers, data, err := Query(db, "SELECT id, name FROM empty_table")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if len(headers) != 2 {
		t.Errorf("headers: got %d, want 2", len(headers))
	}
	if len(data) != 0 {
		t.Errorf("rows: got %d, want 0", len(data))
	}
}

func TestQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery("SELECT").WillReturnError(fmt.Errorf("table not found"))

	_, _, err = Query(db, "SELECT * FROM nonexistent")
	if err == nil {
		t.Error("expected error for failed query")
	}
}

func TestExecSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))

	result, err := Exec(db, "INSERT INTO t (name) VALUES ('test')")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}

	affected, _ := result.RowsAffected()
	if affected != 1 {
		t.Errorf("rows affected: got %d, want 1", affected)
	}
}

func TestExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectExec("INSERT").WillReturnError(fmt.Errorf("constraint violation"))

	_, err = Exec(db, "INSERT INTO t (id) VALUES (1)")
	if err == nil {
		t.Error("expected error for failed exec")
	}
}

func TestPingSuccess(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectPing()

	if err := Ping(db); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestPingError(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectPing().WillReturnError(fmt.Errorf("connection refused"))

	if err := Ping(db); err == nil {
		t.Error("expected error for failed ping")
	}
}

func TestQueryIntValues(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	rows := sqlmock.NewRows([]string{"count"}).
		AddRow(int64(42))

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	_, data, err := Query(db, "SELECT COUNT(*) as count FROM t")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if data[0][0] != "42" {
		t.Errorf("int value: got %q, want 42", data[0][0])
	}
}

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"single statement", "SELECT 1", []string{"SELECT 1"}},
		{"single with semicolon", "SELECT 1;", []string{"SELECT 1"}},
		{"two statements", "SELECT 1; SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"quoted semicolon", "SELECT 'a;b'", []string{"SELECT 'a;b'"}},
		{"double quoted semicolon", `SELECT "a;b"`, []string{`SELECT "a;b"`}},
		{"empty statement skipped", "SELECT 1;; SELECT 2", []string{"SELECT 1", "SELECT 2"}},
		{"whitespace only", "  ;  ;  ", nil},
		{"escaped quote", `SELECT 'it\'s'`, []string{`SELECT 'it\'s'`}},
		{"multi-line", "CREATE TABLE t (\n  id INT\n);\nINSERT INTO t VALUES (1)", []string{"CREATE TABLE t (\n  id INT\n)", "INSERT INTO t VALUES (1)"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitStatements(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d statements, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("stmt[%d]: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
