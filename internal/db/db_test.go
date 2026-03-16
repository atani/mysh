package db

import (
	"testing"
)

func TestBuildDSN(t *testing.T) {
	// Verify that Config.FormatDSN produces a valid DSN via Open's internals.
	// We can't connect to a real DB in unit tests, but we can verify
	// the function doesn't panic and returns a non-nil *sql.DB.
	conn, err := Open("127.0.0.1", 3306, "testuser", "p@ss:w0rd/special", "testdb")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = conn.Close() }()
	// sql.Open is lazy; it doesn't actually connect.
	// This just verifies DSN construction doesn't fail for special characters.
}
