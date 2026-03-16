package db

import (
	"fmt"
	"strings"
)

// FormatTabular formats query results as mysql-style tabular output.
// This allows reuse of the existing mask/format/parse infrastructure.
func FormatTabular(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := 0; i < len(row) && i < len(widths); i++ {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}

	var b strings.Builder

	sep := buildSeparator(widths)
	b.WriteString(sep)
	b.WriteByte('\n')

	b.WriteString(buildRow(headers, widths))
	b.WriteByte('\n')

	b.WriteString(sep)
	b.WriteByte('\n')

	for _, row := range rows {
		b.WriteString(buildRow(row, widths))
		b.WriteByte('\n')
	}

	b.WriteString(sep)
	b.WriteByte('\n')

	return b.String()
}

func buildSeparator(widths []int) string {
	var b strings.Builder
	for _, w := range widths {
		b.WriteByte('+')
		b.WriteString(strings.Repeat("-", w+2))
	}
	b.WriteByte('+')
	return b.String()
}

func buildRow(fields []string, widths []int) string {
	var b strings.Builder
	for i, w := range widths {
		b.WriteByte('|')
		val := ""
		if i < len(fields) {
			val = fields[i]
		}
		b.WriteString(fmt.Sprintf(" %-*s ", w, val))
	}
	b.WriteByte('|')
	return b.String()
}
