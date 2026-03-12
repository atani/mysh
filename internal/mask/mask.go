package mask

import (
	"regexp"
	"strings"
)

// Value masks a string value based on its content.
func Value(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "NULL" {
		return s
	}

	// Email: keep first char + domain
	if isEmail(s) {
		return maskEmail(s)
	}

	// Short values: full mask
	if len(s) <= 3 {
		return "***"
	}

	// Default: keep first char, mask rest
	return string([]rune(s)[0]) + "***"
}

var emailRe = regexp.MustCompile(`^[^@]+@[^@]+\.[^@]+$`)

func isEmail(s string) bool {
	return emailRe.MatchString(s)
}

func maskEmail(s string) string {
	parts := strings.SplitN(s, "@", 2)
	local := parts[0]
	domain := parts[1]

	return string(local[0]) + "***@" + domain
}

// TabularOutput masks columns in mysql's default tabular output format.
// It detects the header row, identifies which columns to mask, and replaces values.
func TabularOutput(output string, maskedCols map[int]bool) string {
	if len(maskedCols) == 0 {
		return output
	}

	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		return output
	}

	// Detect format: tabular (+----|----+) or tab-separated
	if strings.HasPrefix(lines[0], "+") {
		return maskTabularFormat(lines, maskedCols)
	}

	return maskTSVFormat(lines, maskedCols)
}

// maskTabularFormat handles mysql's box-drawing table format:
// +----+-------+------------------+
// | id | name  | email            |
// +----+-------+------------------+
// |  1 | Alice | alice@example.com|
// +----+-------+------------------+
func maskTabularFormat(lines []string, maskedCols map[int]bool) string {
	// Find column boundaries from separator line
	if len(lines) < 4 {
		return strings.Join(lines, "\n")
	}

	sep := lines[0]
	boundaries := findBoundaries(sep)
	if len(boundaries) < 2 {
		return strings.Join(lines, "\n")
	}

	var result []string
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || line == "" {
			result = append(result, line)
			continue
		}

		if !strings.HasPrefix(line, "|") {
			result = append(result, line)
			continue
		}

		// Check if this is the header row (line index 1)
		isHeader := false
		for i, l := range lines {
			if l == line && i == 1 {
				isHeader = true
				break
			}
		}
		if isHeader {
			result = append(result, line)
			continue
		}

		result = append(result, maskTableRow(line, boundaries, maskedCols))
	}

	return strings.Join(result, "\n")
}

func findBoundaries(sep string) []int {
	var bounds []int
	for i, ch := range sep {
		if ch == '+' {
			bounds = append(bounds, i)
		}
	}
	return bounds
}

func maskTableRow(line string, boundaries []int, maskedCols map[int]bool) string {
	runes := []rune(line)
	colIdx := 0

	for i := 0; i < len(boundaries)-1; i++ {
		if !maskedCols[colIdx] {
			colIdx++
			continue
		}

		start := boundaries[i] + 1
		end := boundaries[i+1]
		if start >= len(runes) || end > len(runes) {
			colIdx++
			continue
		}

		// Extract value, mask it, pad to fit
		val := string(runes[start:end])
		val = strings.TrimRight(val, " ")
		val = strings.TrimPrefix(val, " ")

		masked := Value(val)
		width := end - start - 2 // exclude padding spaces
		if len(masked) > width {
			masked = masked[:width]
		}
		padded := " " + masked + strings.Repeat(" ", width-len(masked)) + " "

		for j, ch := range []rune(padded) {
			if start+j < end {
				runes[start+j] = ch
			}
		}

		colIdx++
	}

	return string(runes)
}

// maskTSVFormat handles tab-separated output (mysql -B or batch mode).
func maskTSVFormat(lines []string, maskedCols map[int]bool) string {
	if len(lines) < 2 {
		return strings.Join(lines, "\n")
	}

	var result []string
	// First line is header, keep as-is
	result = append(result, lines[0])

	for _, line := range lines[1:] {
		if line == "" {
			result = append(result, line)
			continue
		}

		fields := strings.Split(line, "\t")
		for i := range fields {
			if maskedCols[i] {
				fields[i] = Value(fields[i])
			}
		}
		result = append(result, strings.Join(fields, "\t"))
	}

	return strings.Join(result, "\n")
}
