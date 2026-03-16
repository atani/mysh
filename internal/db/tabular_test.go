package db

import (
	"strings"
	"testing"
)

func TestFormatTabularBasic(t *testing.T) {
	headers := []string{"id", "name"}
	rows := [][]string{
		{"1", "Alice"},
		{"2", "Bob"},
	}

	got := FormatTabular(headers, rows)

	if !strings.Contains(got, "| id | name  |") {
		t.Errorf("expected header row, got:\n%s", got)
	}
	if !strings.Contains(got, "| 1  | Alice |") {
		t.Errorf("expected data row for Alice, got:\n%s", got)
	}
	if !strings.Contains(got, "| 2  | Bob   |") {
		t.Errorf("expected data row for Bob, got:\n%s", got)
	}

	// sep + header + sep + 2 data rows + sep = 6 lines
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 6 {
		t.Errorf("expected 6 lines, got %d:\n%s", len(lines), got)
	}
}

func TestFormatTabularEmpty(t *testing.T) {
	got := FormatTabular(nil, nil)
	if got != "" {
		t.Errorf("expected empty string for nil headers, got: %q", got)
	}

	got = FormatTabular([]string{}, nil)
	if got != "" {
		t.Errorf("expected empty string for empty headers, got: %q", got)
	}
}

func TestFormatTabularFewerFieldsThanHeaders(t *testing.T) {
	headers := []string{"id", "name", "email"}
	rows := [][]string{
		{"1", "Alice"}, // missing email
	}

	got := FormatTabular(headers, rows)

	// Should not panic, missing fields should be empty
	if !strings.Contains(got, "| 1  | Alice |") {
		t.Errorf("expected partial row, got:\n%s", got)
	}
}

func TestBuildSeparator(t *testing.T) {
	sep := buildSeparator([]int{3, 5})
	if sep != "+-----+-------+" {
		t.Errorf("unexpected separator: %q", sep)
	}
}

func TestBuildRow(t *testing.T) {
	row := buildRow([]string{"hi", "world"}, []int{3, 5})
	if row != "| hi  | world |" {
		t.Errorf("unexpected row: %q", row)
	}
}
