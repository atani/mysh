package mask

import (
	"strings"
	"unicode"

	"github.com/atani/mysh/internal/mysql"
)

// IsInvisibleRune reports whether r is whitespace or an invisible character
// (zero-width space, BOM, zero-width joiners) that could silently break
// config matching if present in user-supplied mask rules.
func IsInvisibleRune(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '\u200B', '\u200C', '\u200D', '\u2060', '\uFEFF':
		return true
	}
	return false
}

// CleanEntries trims invisible/whitespace characters from each entry and drops
// entries that become empty. This lets users tolerate accidental whitespace
// in YAML config without silently skipping masking.
func CleanEntries(entries []string) []string {
	if len(entries) == 0 {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		e = strings.TrimFunc(e, IsInvisibleRune)
		if e == "" {
			continue
		}
		out = append(out, e)
	}
	return out
}

// FindMaskColumns determines which column indices to mask based on exact
// column names and wildcard patterns. Leading/trailing whitespace (including
// tabs, full-width space, and zero-width characters) is trimmed from each
// entry; entries that become empty are skipped.
func FindMaskColumns(headers []string, columns []string, patterns []string) map[int]bool {
	cleanCols := CleanEntries(columns)
	cleanPats := CleanEntries(patterns)
	masked := make(map[int]bool)
	for i, h := range headers {
		for _, col := range cleanCols {
			if strings.EqualFold(h, col) {
				masked[i] = true
				break
			}
		}
		lowerH := strings.ToLower(h)
		for _, pat := range cleanPats {
			if matchWildcard(strings.ToLower(pat), lowerH) {
				masked[i] = true
				break
			}
		}
	}
	return masked
}

// matchWildcard supports simple wildcard matching (* only).
func matchWildcard(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == s
	}

	parts := strings.Split(pattern, "*")
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(s[pos:], part)
		if idx < 0 {
			return false
		}
		if i == 0 && idx != 0 {
			return false
		}
		pos += idx + len(part)
	}
	if parts[len(parts)-1] != "" && pos != len(s) {
		return false
	}
	return true
}

// ApplyToOutput parses mysql output, applies masking, and returns the masked
// output string along with the names of masked columns.
func ApplyToOutput(output string, columns []string, patterns []string) (string, []string) {
	r := mysql.ParseOutput(output)
	if r == nil {
		return output, nil
	}

	maskedCols := FindMaskColumns(r.Headers, columns, patterns)
	if len(maskedCols) == 0 {
		return output, nil
	}

	var names []string
	for i, h := range r.Headers {
		if maskedCols[i] {
			names = append(names, h)
		}
	}

	masked := TabularOutput(output, maskedCols)
	return masked, names
}
