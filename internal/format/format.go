package format

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/atani/mysh/internal/mysql"
)

const pdfMaxRows = 10000

type Type string

const (
	Plain    Type = "plain"
	Markdown Type = "markdown"
	CSV      Type = "csv"
	JSON     Type = "json"
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
	case "json":
		return JSON, nil
	case "pdf":
		return PDF, nil
	default:
		return "", fmt.Errorf("unknown format %q (supported: plain, markdown, csv, json, pdf)", s)
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
	case JSON:
		return toJSON(output)
	case PDF:
		return "", fmt.Errorf("use WritePDF for PDF output")
	default:
		return output, nil
	}
}

// containsNonASCII reports whether s contains any non-ASCII character.
func containsNonASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return true
		}
	}
	return false
}

// tableHasNonASCII checks whether any header or cell value contains non-ASCII characters.
func tableHasNonASCII(headers []string, rows [][]string) bool {
	for _, h := range headers {
		if containsNonASCII(h) {
			return true
		}
	}
	for _, row := range rows {
		for _, cell := range row {
			if containsNonASCII(cell) {
				return true
			}
		}
	}
	return false
}

// WritePDF writes mysql output as a PDF file.
func WritePDF(output string, path string) error {
	headers, rows := parseOutput(output)
	if len(headers) == 0 {
		return fmt.Errorf("no data to export")
	}

	if tableHasNonASCII(headers, rows) {
		fmt.Fprintf(os.Stderr, "[mysh] warning: PDF output may not render non-ASCII characters correctly (CJK, accented chars)\n")
	}

	truncated := 0
	if len(rows) > pdfMaxRows {
		truncated = len(rows) - pdfMaxRows
		rows = rows[:pdfMaxRows]
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

	if truncated > 0 {
		note := fmt.Sprintf("[truncated: %d more rows]", truncated)
		pdf.CellFormat(colWidths[0], 6, note, "1", 0, "L", false, 0, "")
		for i := 1; i < len(colWidths); i++ {
			pdf.CellFormat(colWidths[i], 6, "", "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
}

// WriteFile writes formatted output to a file.
func WriteFile(content string, path string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

func toMarkdown(output string) string {
	headers, rows := parseOutput(output)
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
		escaped := make([]string, len(row))
		for i, cell := range row {
			escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
		}
		b.WriteString("| " + strings.Join(escaped, " | ") + " |\n")
	}

	return b.String()
}

func toCSV(output string) (string, error) {
	headers, rows := parseOutput(output)
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
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func toJSON(output string) (string, error) {
	headers, rows := parseOutput(output)
	if len(headers) == 0 {
		return "[]", nil
	}

	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		record := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				record[h] = row[i]
			} else {
				record[h] = ""
			}
		}
		records = append(records, record)
	}

	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b) + "\n", nil
}

// ConvertResult converts structured data directly to the specified format,
// avoiding the round-trip through tabular string serialization.
func ConvertResult(headers []string, rows [][]string, format Type) (string, error) {
	switch format {
	case Markdown:
		return toMarkdownResult(headers, rows), nil
	case CSV:
		return toCSVResult(headers, rows)
	case JSON:
		return toJSONResult(headers, rows)
	case Plain:
		return "", fmt.Errorf("use db.FormatTabular for plain output")
	case PDF:
		return "", fmt.Errorf("use WritePDFResult for PDF output")
	default:
		return "", fmt.Errorf("unsupported format for structured output: %s", format)
	}
}

// WritePDFResult writes structured data as a PDF file.
func WritePDFResult(headers []string, rows [][]string, path string) error {
	if len(headers) == 0 {
		return fmt.Errorf("no data to export")
	}
	return writePDFData(headers, rows, path)
}

func toMarkdownResult(headers []string, rows [][]string) string {
	var b strings.Builder
	b.WriteString("| " + strings.Join(headers, " | ") + " |\n")
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| " + strings.Join(seps, " | ") + " |\n")

	for _, row := range rows {
		for len(row) < len(headers) {
			row = append(row, "")
		}
		escaped := make([]string, len(row))
		for i, cell := range row {
			escaped[i] = strings.ReplaceAll(cell, "|", "\\|")
		}
		b.WriteString("| " + strings.Join(escaped, " | ") + " |\n")
	}
	return b.String()
}

func toCSVResult(headers []string, rows [][]string) (string, error) {
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
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func toJSONResult(headers []string, rows [][]string) (string, error) {
	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		record := make(map[string]string, len(headers))
		for i, h := range headers {
			if i < len(row) {
				record[h] = row[i]
			} else {
				record[h] = ""
			}
		}
		records = append(records, record)
	}
	b, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return "", fmt.Errorf("json marshal: %w", err)
	}
	return string(b) + "\n", nil
}

// parseOutput delegates to mysql.ParseOutput and returns headers and rows.
func parseOutput(output string) ([]string, [][]string) {
	r := mysql.ParseOutput(output)
	if r == nil {
		return nil, nil
	}
	return r.Headers, r.Rows
}

func writePDFData(headers []string, rows [][]string, path string) error {
	if tableHasNonASCII(headers, rows) {
		fmt.Fprintf(os.Stderr, "[mysh] warning: PDF output may not render non-ASCII characters correctly (CJK, accented chars)\n")
	}

	truncated := 0
	if len(rows) > pdfMaxRows {
		truncated = len(rows) - pdfMaxRows
		rows = rows[:pdfMaxRows]
	}

	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	pdf.SetFont("Courier", "B", 9)

	pageW, pageH := pdf.GetPageSize()
	colWidths := computeColWidths(headers, rows, pageW, pageH)

	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", false, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Courier", "", 8)
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) {
				pdf.CellFormat(colWidths[i], 6, cell, "1", 0, "L", false, 0, "")
			}
		}
		pdf.Ln(-1)
	}

	if truncated > 0 {
		note := fmt.Sprintf("[truncated: %d more rows]", truncated)
		pdf.CellFormat(colWidths[0], 6, note, "1", 0, "L", false, 0, "")
		for i := 1; i < len(colWidths); i++ {
			pdf.CellFormat(colWidths[i], 6, "", "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0600)
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
	if total == 0 {
		even := usable / float64(len(widths))
		for i := range result {
			result[i] = even
		}
		return result
	}
	for i, w := range widths {
		result[i] = (w / total) * usable
		if result[i] < 15 {
			result[i] = 15
		}
	}
	return result
}
