package mask

import (
	"strings"
	"testing"
)

func TestFindMaskColumns(t *testing.T) {
	tests := []struct {
		name     string
		headers  []string
		columns  []string
		patterns []string
		want     map[int]bool
	}{
		{
			name:     "exact column name matching",
			headers:  []string{"id", "name", "email", "phone"},
			columns:  []string{"email", "phone"},
			patterns: nil,
			want:     map[int]bool{2: true, 3: true},
		},
		{
			name:     "case-insensitive exact match",
			headers:  []string{"id", "Email", "PHONE"},
			columns:  []string{"email", "phone"},
			patterns: nil,
			want:     map[int]bool{1: true, 2: true},
		},
		{
			name:     "wildcard prefix pattern",
			headers:  []string{"id", "user_email", "name"},
			columns:  nil,
			patterns: []string{"*email"},
			want:     map[int]bool{1: true},
		},
		{
			name:     "wildcard suffix pattern",
			headers:  []string{"id", "email_address", "name"},
			columns:  nil,
			patterns: []string{"email*"},
			want:     map[int]bool{1: true},
		},
		{
			name:     "wildcard contains pattern",
			headers:  []string{"id", "home_address_line", "name"},
			columns:  nil,
			patterns: []string{"*address*"},
			want:     map[int]bool{1: true},
		},
		{
			name:     "no matches",
			headers:  []string{"id", "name", "created_at"},
			columns:  []string{"email"},
			patterns: []string{"*phone*"},
			want:     map[int]bool{},
		},
		{
			name:     "empty patterns and columns",
			headers:  []string{"id", "name"},
			columns:  nil,
			patterns: nil,
			want:     map[int]bool{},
		},
		{
			name:     "combined exact and pattern",
			headers:  []string{"id", "email", "home_address", "created_at"},
			columns:  []string{"email"},
			patterns: []string{"*address*"},
			want:     map[int]bool{1: true, 2: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindMaskColumns(tt.headers, tt.columns, tt.patterns)
			for i := range tt.headers {
				if tt.want[i] != got[i] {
					t.Errorf("column %q (index %d): got masked=%v, want masked=%v",
						tt.headers[i], i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMatchWildcard(t *testing.T) {
	tests := []struct {
		pattern string
		s       string
		want    bool
	}{
		// prefix match: *suffix
		{"*email", "user_email", true},
		{"*email", "email", true},
		{"*email", "email_addr", false},
		// suffix match: prefix*
		{"phone*", "phone_number", true},
		{"phone*", "phone", true},
		{"phone*", "home_phone", false},
		// contains match: *mid*
		{"*addr*", "home_address", true},
		{"*addr*", "addr_line", true},
		{"*addr*", "name", false},
		// exact match via wildcard-all
		{"*", "anything", true},
		// no-wildcard exact match
		{"email", "email", true},
		{"email", "Email", false},
		// case sensitivity (matchWildcard itself is case-sensitive)
		{"Email", "email", false},
		// no match
		{"phone", "email", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.s, func(t *testing.T) {
			got := matchWildcard(tt.pattern, tt.s)
			if got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v",
					tt.pattern, tt.s, got, tt.want)
			}
		})
	}
}

func TestApplyToOutput(t *testing.T) {
	t.Run("tabular input with masking", func(t *testing.T) {
		input := strings.Join([]string{
			"+----+-------+-------------------+",
			"| id | name  | email             |",
			"+----+-------+-------------------+",
			"|  1 | Alice | alice@example.com |",
			"+----+-------+-------------------+",
		}, "\n")

		result, names := ApplyToOutput(input, []string{"email"}, nil)
		if len(names) != 1 || names[0] != "email" {
			t.Errorf("masked column names = %v, want [email]", names)
		}
		if strings.Contains(result, "alice@example.com") {
			t.Error("email value should be masked in output")
		}
		// id and name should remain
		if !strings.Contains(result, "Alice") {
			t.Error("non-masked column 'name' should be unchanged")
		}
	})

	t.Run("TSV input with masking", func(t *testing.T) {
		input := "id\tname\temail\n1\tAlice\talice@example.com"

		result, names := ApplyToOutput(input, []string{"email"}, nil)
		if len(names) != 1 || names[0] != "email" {
			t.Errorf("masked column names = %v, want [email]", names)
		}
		lines := strings.Split(result, "\n")
		if len(lines) < 2 {
			t.Fatal("expected at least 2 lines in output")
		}
		fields := strings.Split(lines[1], "\t")
		if len(fields) < 3 {
			t.Fatal("expected at least 3 fields in data row")
		}
		if fields[2] == "alice@example.com" {
			t.Error("email should be masked")
		}
		if fields[1] != "Alice" {
			t.Error("name should be unchanged")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		result, names := ApplyToOutput("", []string{"email"}, nil)
		if result != "" {
			t.Errorf("expected empty result, got %q", result)
		}
		if names != nil {
			t.Errorf("expected nil names, got %v", names)
		}
	})

	t.Run("no matching columns leaves output unchanged", func(t *testing.T) {
		input := "id\tname\n1\tAlice"
		result, names := ApplyToOutput(input, []string{"email"}, nil)
		if result != input {
			t.Errorf("output should be unchanged when no columns match")
		}
		if names != nil {
			t.Errorf("expected nil names, got %v", names)
		}
	})

	t.Run("nil columns and patterns", func(t *testing.T) {
		input := "id\tname\temail\n1\tAlice\talice@example.com"
		result, names := ApplyToOutput(input, nil, nil)
		if result != input {
			t.Errorf("output should be unchanged with nil columns/patterns")
		}
		if names != nil {
			t.Errorf("expected nil names, got %v", names)
		}
	})

	t.Run("empty columns and patterns", func(t *testing.T) {
		input := "id\tname\temail\n1\tAlice\talice@example.com"
		result, names := ApplyToOutput(input, []string{}, []string{})
		if result != input {
			t.Errorf("output should be unchanged with empty columns/patterns")
		}
		if names != nil {
			t.Errorf("expected nil names, got %v", names)
		}
	})
}
