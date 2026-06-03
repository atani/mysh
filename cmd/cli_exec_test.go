package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/atani/mysh/internal/config"
)

// TestHelperProcess stands in for the external mysql/mycli client. When invoked
// with GO_WANT_HELPER_PROCESS=1 it prints a fixed MySQL-style tabular result to
// stdout (or fails, per MYSH_CLI_MODE) so the CLI capture/output paths can run
// without a real client binary or database.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if os.Getenv("MYSH_CLI_MODE") == "fail" {
		os.Exit(2)
	}
	// MySQL CLI tabular output: tab-separated header + rows.
	os.Stdout.WriteString("id\tname\n1\talice\n2\tbob\n")
	os.Exit(0)
}

func stubCLI(t *testing.T, mode string) {
	t.Helper()
	orig := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		cs := append([]string{"-test.run=TestHelperProcess", "--"}, args...)
		c := exec.Command(os.Args[0], cs...)
		c.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "MYSH_CLI_MODE="+mode)
		return c
	}
	t.Cleanup(func() { execCommand = orig })
}

func TestRunQueryCLICaptureCSV(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")
	dir := t.TempDir()
	out := filepath.Join(dir, "out.csv")
	// Capture path (format != plain) -> runQueryCLI buffers and converts.
	if err := RunQuery([]string{"cli", "-e", "SELECT 1", "--format", "csv", "-o", out}); err != nil {
		t.Fatalf("RunQuery CLI csv: %v", err)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty CSV output")
	}
}

func TestRunQueryCLIWithMasking(t *testing.T) {
	c := dbConn("cli")
	c.Env = "production"
	c.Mask = &config.MaskConfig{Columns: []string{"name"}}
	setupConfig(t, c)
	stubCLI(t, "ok")
	// shouldMask true (production + mask) -> capture + ApplyToOutput path.
	if err := RunQuery([]string{"cli", "-e", "SELECT 1", "--mask"}); err != nil {
		t.Fatalf("RunQuery CLI masking: %v", err)
	}
}

func TestRunQueryCLIError(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "fail")
	if err := RunQuery([]string{"cli", "-e", "SELECT 1", "--format", "csv"}); err == nil {
		t.Error("expected error when client fails")
	}
}

func TestRunTablesCLICapture(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")
	dir := t.TempDir()
	out := filepath.Join(dir, "tables.csv")
	if err := RunTables([]string{"cli", "--format", "csv", "-o", out}); err != nil {
		t.Fatalf("RunTables CLI capture: %v", err)
	}
	if data, _ := os.ReadFile(out); len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestRunSliceCLICapture(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")
	dir := t.TempDir()
	out := filepath.Join(dir, "slice.sql")
	if err := RunSlice([]string{"cli", "users", "--where", "id > 0", "-o", out}); err != nil {
		t.Fatalf("RunSlice CLI capture: %v", err)
	}
	if data, _ := os.ReadFile(out); len(data) == 0 {
		t.Error("expected non-empty slice output")
	}
}

func TestRunPingCLISuccess(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")
	if err := RunPing([]string{"cli"}); err != nil {
		t.Fatalf("RunPing CLI: %v", err)
	}
}

func TestRunConnectCLISuccess(t *testing.T) {
	setupConfig(t, dbConn("cli"))
	stubCLI(t, "ok")
	// Force mysql selection: pretend mycli is absent but mysql is present.
	origLook := lookPath
	lookPath = func(file string) (string, error) {
		if file == "mycli" {
			return "", exec.ErrNotFound
		}
		return "/usr/bin/mysql", nil
	}
	t.Cleanup(func() { lookPath = origLook })

	if err := RunConnect([]string{"cli"}); err != nil {
		t.Fatalf("RunConnect CLI: %v", err)
	}
}
