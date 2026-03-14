package mysql

import "strings"

// QueryResult represents parsed mysql output.
type QueryResult struct {
	Headers []string
	Rows    [][]string
}

// ParseOutput parses mysql tabular or TSV output into structured data.
func ParseOutput(output string) *QueryResult {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 2 {
		return nil
	}

	if strings.HasPrefix(lines[0], "+") {
		return parseTabular(lines)
	}

	return parseTSV(lines)
}

func parseTabular(lines []string) *QueryResult {
	if len(lines) < 4 {
		return nil
	}

	headers := splitPipeRow(lines[1])
	var rows [][]string
	for _, line := range lines[3:] {
		if strings.HasPrefix(line, "|") {
			rows = append(rows, splitPipeRow(line))
		}
	}
	return &QueryResult{Headers: headers, Rows: rows}
}

func splitPipeRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	var fields []string
	for _, p := range parts {
		fields = append(fields, strings.TrimSpace(p))
	}
	return fields
}

func parseTSV(lines []string) *QueryResult {
	headers := strings.Split(lines[0], "\t")
	var rows [][]string
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return &QueryResult{Headers: headers, Rows: rows}
}
