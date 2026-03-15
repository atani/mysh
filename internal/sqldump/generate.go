package sqldump

import (
	"fmt"
	"strings"
	"time"

	"github.com/atani/mysh/internal/mysql"
)

// Options controls dump generation.
type Options struct {
	Where     string
	Timestamp time.Time
}

// Generate converts a QueryResult into INSERT statements.
func Generate(table string, result *mysql.QueryResult, opts Options) string {
	var b strings.Builder

	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	fmt.Fprintf(&b, "-- mysh slice: %s WHERE %s\n", table, opts.Where)
	fmt.Fprintf(&b, "-- Generated at: %s\n", ts.Format(time.RFC3339))

	if result == nil || len(result.Rows) == 0 {
		return b.String()
	}

	b.WriteString("\n")

	quotedHeaders := make([]string, len(result.Headers))
	for i, h := range result.Headers {
		quotedHeaders[i] = "`" + h + "`"
	}
	colList := strings.Join(quotedHeaders, ", ")

	for _, row := range result.Rows {
		// Guard against row/header length mismatch
		n := len(result.Headers)
		if len(row) < n {
			n = len(row)
		}
		vals := make([]string, n)
		for i := 0; i < n; i++ {
			vals[i] = formatValue(row[i])
		}
		fmt.Fprintf(&b, "INSERT INTO `%s` (%s) VALUES (%s);\n", table, colList, strings.Join(vals, ", "))
	}

	return b.String()
}

func formatValue(s string) string {
	if s == "NULL" {
		return "NULL"
	}
	if isNumeric(s) {
		return s
	}
	return "'" + escapeValue(s) + "'"
}

func escapeValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `''`)
	s = strings.ReplaceAll(s, "\x00", `\0`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\x1a", `\Z`)
	return s
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	i := 0
	if s[0] == '-' || s[0] == '+' {
		i = 1
	}
	if i >= len(s) {
		return false
	}
	// Must start with a digit
	if s[i] < '0' || s[i] > '9' {
		return false
	}
	dotSeen := false
	for _, ch := range s[i:] {
		if ch == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
	}
	// Must end with a digit
	return s[len(s)-1] >= '0' && s[len(s)-1] <= '9'
}
