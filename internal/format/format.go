package format

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/go-pdf/fpdf"
)

type Type string

const (
	Plain    Type = "plain"
	Markdown Type = "markdown"
	CSV      Type = "csv"
	PDF      Type = "pdf"
)

func Parse(s string) (Type, error) {
	switch strings.ToLower(s) {
	case "plain", "":
		return Plain, nil
	case "markdown", "md":
		return Markdown, nil
	case "csv":
		return CSV, nil
	case "pdf":
		return PDF, nil
	default:
		return "", fmt.Errorf("unknown format %q (supported: plain, markdown, csv, pdf)", s)
	}
}

// Convert transforms mysql tabular/TSV output into the specified format.
func Convert(output string, format Type) (string, error) {
	switch format {
	case Plain:
		return output, nil
	case Markdown:
		return toMarkdown(output), nil
	case CSV:
		return toCSV(output)
	case PDF:
		return "", fmt.Errorf("use WritePDF for PDF output")
	default:
		return output, nil
	}
}

// WritePDF writes mysql output as a PDF file.
func WritePDF(output string, path string) error {
	headers, rows := parseTable(output)
	if len(headers) == 0 {
		return fmt.Errorf("no data to export")
	}

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	pdf.SetFont("Courier", "B", 9)

	pageW, pageH := pdf.GetPageSize()
	colWidths := computeColWidths(headers, rows, pageW, pageH)

	// Header
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	// Rows
	pdf.SetFont("Courier", "", 8)
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				pdf.CellFormat(colWidths[i], 6, cell, "1", 0, "L", false, 0, "")
			}
		}
		pdf.Ln(-1)
	}

	return pdf.OutputFileAndClose(path)
}

// WriteFile writes formatted output to a file.
func WriteFile(content string, path string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

func toMarkdown(output string) string {
	headers, rows := parseTable(output)
	if len(headers) == 0 {
		return output
	}

	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| " + strings.Join(seps, " | ") + " |\n")

	for _, row := range rows {
		// Pad row to header length
		for len(row) < len(headers) {
			row = append(row, "")
		}
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	return b.String()
}

func toCSV(output string) (string, error) {
	headers, rows := parseTable(output)
	if len(headers) == 0 {
		return output, nil
	}

	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(headers); err != nil {
		return "", fmt.Errorf("csv write headers: %w", err)
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return "", fmt.Errorf("csv write row: %w", err)
		}
	}
	w.Flush()
	return b.String(), nil
}

// parseTable extracts headers and data rows from mysql tabular or TSV output.
func parseTable(output string) ([]string, [][]string) {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	// Tabular format: +---+---+
	if strings.HasPrefix(lines[0], "+") {
		return parseTabular(lines)
	}

	// TSV format
	return parseTSV(lines)
}

func parseTabular(lines []string) ([]string, [][]string) {
	if len(lines) < 4 {
		return nil, nil
	}

	headers := splitPipeRow(lines[1])
	var rows [][]string
	for _, line := range lines[3:] {
		if strings.HasPrefix(line, "|") {
			rows = append(rows, splitPipeRow(line))
		}
	}
	return headers, rows
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

func parseTSV(lines []string) ([]string, [][]string) {
	headers := strings.Split(lines[0], "\t")
	var rows [][]string
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return headers, rows
}

func computeColWidths(headers []string, rows [][]string, pageW, _ float64) []float64 {
	margin := 20.0
	usable := pageW - margin

	widths := make([]float64, len(headers))
	for i, h := range headers {
		widths[i] = float64(len(h))
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && float64(len(cell)) > widths[i] {
				widths[i] = float64(len(cell))
			}
		}
	}

	var total float64
	for _, w := range widths {
		total += w
	}

	result := make([]float64, len(widths))
	for i, w := range widths {
		result[i] = (w / total) * usable
		if result[i] < 15 {
			result[i] = 15
		}
	}
	return result
}
