package mask

import (
	"strings"
	"testing"
)

func TestValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"NULL", "NULL"},
		{"alice@example.com", "a***@example.com"},
		{"a@example.com", "a***@example.com"},
		{"x", "***"},
		{"ab", "***"},
		{"abc", "***"},
		{"Alice", "A***"},
		{"090-1234-5678", "0***"},
		{"Tokyo Shibuya-ku", "T***"},
	}

	for _, tt := range tests {
		got := Value(tt.input)
		if got != tt.want {
			t.Errorf("Value(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMaskTSVFormat(t *testing.T) {
	input := strings.Join([]string{
		"id\tname\temail",
		"1\tAlice\talice@example.com",
		"2\tBob\tbob@example.com",
	}, "\n")

	maskedCols := map[int]bool{2: true} // mask email column
	got := TabularOutput(input, maskedCols)
	lines := strings.Split(got, "\n")

	if lines[0] != "id\tname\temail" {
		t.Errorf("header changed: %q", lines[0])
	}

	if !strings.Contains(lines[1], "a***@example.com") {
		t.Errorf("email not masked in line 1: %q", lines[1])
	}

	if !strings.Contains(lines[2], "b***@example.com") {
		t.Errorf("email not masked in line 2: %q", lines[2])
	}

	// name should not be masked
	if !strings.Contains(lines[1], "Alice") {
		t.Errorf("name should not be masked: %q", lines[1])
	}
}

func TestMaskTSVMultipleColumns(t *testing.T) {
	input := strings.Join([]string{
		"id\tname\temail\tphone",
		"1\tAlice\talice@example.com\t090-1234-5678",
	}, "\n")

	maskedCols := map[int]bool{2: true, 3: true}
	got := TabularOutput(input, maskedCols)
	lines := strings.Split(got, "\n")

	fields := strings.Split(lines[1], "\t")
	if fields[1] != "Alice" {
		t.Errorf("name should not be masked: %q", fields[1])
	}
	if fields[2] != "a***@example.com" {
		t.Errorf("email not masked: %q", fields[2])
	}
	if fields[3] != "0***" {
		t.Errorf("phone not masked: %q", fields[3])
	}
}
