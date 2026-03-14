package mask

import (
	"strings"

	"github.com/atani/mysh/internal/mysql"
)

// FindMaskColumns determines which column indices to mask based on exact
// column names and wildcard patterns.
func FindMaskColumns(headers []string, columns []string, patterns []string) map[int]bool {
	masked := make(map[int]bool)
	for i, h := range headers {
		for _, col := range columns {
			if strings.EqualFold(h, col) {
				masked[i] = true
			}
		}
		for _, pat := range patterns {
			if matchWildcard(strings.ToLower(pat), strings.ToLower(h)) {
				masked[i] = true
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
