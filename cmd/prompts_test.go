package cmd

import (
	"bufio"
	"errors"
	"strings"
	"testing"

	"github.com/atani/mysh/internal/config"
)

var errInvalid = errors.New("invalid")

func reader(s string) *bufio.Reader { return bufio.NewReader(strings.NewReader(s)) }

func TestAsk(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal string
		want       string
	}{
		{"uses entered value", "alice\n", "bob", "alice"},
		{"empty falls back to default", "\n", "bob", "bob"},
		{"trims whitespace", "  alice  \n", "", "alice"},
		{"empty with no default", "\n", "", ""},
		{"EOF falls back to default", "", "fallback", "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ask(reader(tt.input), "prompt", tt.defaultVal); got != tt.want {
				t.Errorf("ask = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAskRequired(t *testing.T) {
	// First entry is empty (re-prompts), second is non-empty.
	got := askRequired(reader("\nvalue\n"), "prompt")
	if got != "value" {
		t.Errorf("askRequired = %q, want %q", got, "value")
	}
}

func TestAskValidated(t *testing.T) {
	calls := 0
	validate := func(s string) error {
		calls++
		if s == "good" {
			return nil
		}
		return errInvalid
	}
	got := askValidated(reader("bad\ngood\n"), "prompt", validate)
	if got != "good" {
		t.Errorf("askValidated = %q, want %q", got, "good")
	}
	if calls != 2 {
		t.Errorf("validate called %d times, want 2", calls)
	}
}

func TestAskInt(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal int
		want       int
	}{
		{"3307\n", 3306, 3307},
		{"\n", 3306, 3306},
		{"notanumber\n", 22, 22},
	}
	for _, tt := range tests {
		if got := askInt(reader(tt.input), "port", tt.defaultVal); got != tt.want {
			t.Errorf("askInt(%q, %d) = %d, want %d", tt.input, tt.defaultVal, got, tt.want)
		}
	}
}

func TestAskYesNo(t *testing.T) {
	tests := []struct {
		input      string
		defaultVal bool
		want       bool
	}{
		{"y\n", false, true},
		{"yes\n", false, true},
		{"n\n", true, false},
		{"no\n", true, false},
		{"\n", true, true},
		{"\n", false, false},
		{"garbage\n", true, true},
	}
	for _, tt := range tests {
		if got := askYesNo(reader(tt.input), "ok?", tt.defaultVal); got != tt.want {
			t.Errorf("askYesNo(%q, %v) = %v, want %v", tt.input, tt.defaultVal, got, tt.want)
		}
	}
}

func TestAskEnv(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1\n", "production"},
		{"2\n", "staging"},
		{"3\n", "development"},
		{"prod\n", "production"},
		{"stg\n", "staging"},
		{"dev\n", "development"},
		{"bad\n2\n", "staging"}, // invalid then valid
	}
	for _, tt := range tests {
		if got := askEnv(reader(tt.input), "development"); got != tt.want {
			t.Errorf("askEnv(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAskEnvDefault(t *testing.T) {
	// Empty input uses the default for the given current value.
	if got := askEnv(reader("\n"), "production"); got != "production" {
		t.Errorf("askEnv default = %q, want production", got)
	}
}

func TestAskDriver(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1\n", config.DriverCLI},
		{"cli\n", config.DriverCLI},
		{"2\n", config.DriverNative},
		{"native\n", config.DriverNative},
		{"x\n1\n", config.DriverCLI}, // invalid then valid
	}
	for _, tt := range tests {
		if got := askDriver(reader(tt.input)); got != tt.want {
			t.Errorf("askDriver(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAskDriverEditDefault(t *testing.T) {
	// Empty input keeps the current driver via the default option.
	if got := askDriverEdit(reader("\n"), config.DriverNative); got != config.DriverNative {
		t.Errorf("askDriverEdit default = %q, want native", got)
	}
	if got := askDriverEdit(reader("\n"), config.DriverCLI); got != config.DriverCLI {
		t.Errorf("askDriverEdit default = %q, want cli", got)
	}
}

func TestAskEdit(t *testing.T) {
	tests := []struct {
		input   string
		current string
		want    string
	}{
		{"new\n", "old", "new"},
		{"\n", "old", "old"},
		{"new\n", "", "new"},
	}
	for _, tt := range tests {
		if got := askEdit(reader(tt.input), "prompt", tt.current); got != tt.want {
			t.Errorf("askEdit(%q, %q) = %q, want %q", tt.input, tt.current, got, tt.want)
		}
	}
}

func TestAskIntEdit(t *testing.T) {
	if got := askIntEdit(reader("8080\n"), "port", 22); got != 8080 {
		t.Errorf("askIntEdit = %d, want 8080", got)
	}
	if got := askIntEdit(reader("\n"), "port", 22); got != 22 {
		t.Errorf("askIntEdit keep = %d, want 22", got)
	}
	if got := askIntEdit(reader("bad\n"), "port", 22); got != 22 {
		t.Errorf("askIntEdit invalid = %d, want 22", got)
	}
}

func TestAskIfEmpty(t *testing.T) {
	// Flag value present: prompt is skipped.
	if got := askIfEmpty(reader(""), "flagval", "prompt", ""); got != "flagval" {
		t.Errorf("askIfEmpty with flag = %q, want flagval", got)
	}
	// No flag, has default: prompts with default.
	if got := askIfEmpty(reader("\n"), "", "prompt", "def"); got != "def" {
		t.Errorf("askIfEmpty default = %q, want def", got)
	}
	// No flag, no default: required prompt loops until non-empty.
	if got := askIfEmpty(reader("\nentered\n"), "", "prompt", ""); got != "entered" {
		t.Errorf("askIfEmpty required = %q, want entered", got)
	}
}

func TestAskIfEmptyDefault(t *testing.T) {
	if got := askIfEmptyDefault(reader(""), "flagval", "prompt", "def"); got != "flagval" {
		t.Errorf("with flag = %q, want flagval", got)
	}
	if got := askIfEmptyDefault(reader("\n"), "", "prompt", "def"); got != "def" {
		t.Errorf("empty = %q, want def", got)
	}
	if got := askIfEmptyDefault(reader("typed\n"), "", "prompt", "def"); got != "typed" {
		t.Errorf("typed = %q, want typed", got)
	}
}

func TestAskFixChoice(t *testing.T) {
	tests := []struct {
		input  string
		hasSSH bool
		want   string
	}{
		{"1\n", false, "db-host"},
		{"2\n", false, "db-auth"},
		{"3\n", false, "db-name"},
		{"4\n", false, "skip"},
		{"4\n", true, "ssh"},
		{"5\n", true, "skip"},
		{"9\n1\n", false, "db-host"}, // invalid then valid
	}
	for _, tt := range tests {
		if got := askFixChoice(reader(tt.input), tt.hasSSH); got != tt.want {
			t.Errorf("askFixChoice(%q, ssh=%v) = %q, want %q", tt.input, tt.hasSSH, got, tt.want)
		}
	}
}

func TestAskSelection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		total   int
		want    []int
		wantErr bool
	}{
		{"all keyword", "all\n", 3, []int{0, 1, 2}, false},
		{"empty defaults to all", "\n", 2, []int{0, 1}, false},
		{"specific", "1,3\n", 3, []int{0, 2}, false},
		{"dedupe", "2,2,1\n", 3, []int{1, 0}, false},
		{"out of range", "5\n", 3, nil, true},
		{"non-numeric", "abc\n", 3, nil, true},
		{"zero invalid", "0\n", 3, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := askSelection(reader(tt.input), tt.total)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("askSelection: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}
