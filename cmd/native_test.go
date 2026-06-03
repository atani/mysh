package cmd

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/atani/mysh/internal/config"
)

// withMockDB installs a sqlmock database as the native opener and returns the
// mock for setting expectations. The original opener is restored on cleanup.
func withMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	orig := dbOpen
	dbOpen = func(_ *resolvedConn) (*sql.DB, error) { return mockDB, nil }
	t.Cleanup(func() {
		dbOpen = orig
		_ = mockDB.Close()
	})
	return mockDB, mock
}

func nativeConn(name string) config.Connection {
	c := dbConn(name)
	c.DB.Driver = config.DriverNative
	return c
}

func TestRunQueryNativeSuccess(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "alice"))

	if err := RunQuery([]string{"n", "-e", "SELECT id, name FROM users"}); err != nil {
		t.Fatalf("RunQuery native: %v", err)
	}
}

func TestRunQueryNativeQueryOK(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	// A statement that returns no rows (e.g. UPDATE) is run via Query and
	// produces a "Query OK" path.
	mock.ExpectQuery("UPDATE").WillReturnRows(sqlmock.NewRows(nil))

	if err := RunQuery([]string{"n", "-e", "UPDATE t SET x=1"}); err != nil {
		t.Fatalf("RunQuery native OK: %v", err)
	}
}

func TestRunQueryNativeError(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("boom"))

	if err := RunQuery([]string{"n", "-e", "SELECT 1"}); err == nil {
		t.Fatal("expected query error")
	}
}

func TestRunQueryNativeMasking(t *testing.T) {
	c := nativeConn("n")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"email"}}
	setupConfig(t, c)
	_, mock := withMockDB(t)
	mock.ExpectQuery("SELECT").WillReturnRows(
		sqlmock.NewRows([]string{"id", "email"}).AddRow("1", "alice@example.com"))

	// Force production-like masking via --mask.
	if err := RunQuery([]string{"n", "-e", "SELECT id, email FROM users", "--mask"}); err != nil {
		t.Fatalf("RunQuery masking: %v", err)
	}
}

func TestRunTablesNativeSuccess(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SHOW TABLES").WillReturnRows(
		sqlmock.NewRows([]string{"Tables_in_test"}).AddRow("users").AddRow("orders"))

	if err := RunTables([]string{"n"}); err != nil {
		t.Fatalf("RunTables native: %v", err)
	}
}

func TestRunTablesNativeError(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectQuery("SHOW TABLES").WillReturnError(errors.New("denied"))

	if err := RunTables([]string{"n"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRunSliceNativeSuccess(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectExec("SET SESSION TRANSACTION READ ONLY").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT \\* FROM").WillReturnRows(
		sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "alice"))

	if err := RunSlice([]string{"n", "users", "--where", "id > 0"}); err != nil {
		t.Fatalf("RunSlice native: %v", err)
	}
}

func TestRunSliceNativeReadOnlyUnsupported(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectExec("SET SESSION TRANSACTION READ ONLY").WillReturnError(errors.New("not supported"))

	if err := RunSlice([]string{"n", "users", "--where", "id > 0"}); err == nil {
		t.Fatal("expected read-only error")
	}
}

func TestRunSliceNativeNoRows(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectExec("SET SESSION TRANSACTION READ ONLY").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT \\* FROM").WillReturnRows(sqlmock.NewRows([]string{"id"}))

	if err := RunSlice([]string{"n", "users", "--where", "id > 999999"}); err != nil {
		t.Fatalf("RunSlice no rows: %v", err)
	}
}

func TestRunPingNativeSuccess(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectPing()

	if err := RunPing([]string{"n"}); err != nil {
		t.Fatalf("RunPing native: %v", err)
	}
}

func TestRunPingNativeFailure(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectPing().WillReturnError(errors.New("unreachable"))

	if err := RunPing([]string{"n"}); err == nil {
		t.Fatal("expected ping failure")
	}
}

func TestRunConnectNativePingFailure(t *testing.T) {
	setupConfig(t, nativeConn("n"))
	_, mock := withMockDB(t)
	mock.ExpectPing().WillReturnError(errors.New("unreachable"))

	if err := RunConnect([]string{"n"}); err == nil {
		t.Fatal("expected connect failure on ping error")
	}
}
