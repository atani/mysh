package format

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	resultHeaders = []string{"id", "name", "email"}
	resultRows    = [][]string{
		{"1", "Alice", "alice@example.com"},
		{"2", "Bob", "bob@example.com"},
	}
)

func TestConvertResultMarkdown(t *testing.T) {
	got, err := ConvertResult(resultHeaders, resultRows, Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "| id | name | email |") {
		t.Errorf("markdown header missing, got:\n%s", got)
	}
	if !strings.Contains(got, "| --- | --- | --- |") {
		t.Errorf("markdown separator missing, got:\n%s", got)
	}
	if !strings.Contains(got, "| 1 | Alice | alice@example.com |") {
		t.Errorf("markdown data row missing, got:\n%s", got)
	}
}

func TestConvertResultMarkdownPadsShortRows(t *testing.T) {
	// A row with fewer cells than headers must be padded with empty strings.
	got, err := ConvertResult(resultHeaders, [][]string{{"1", "Alice"}}, Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "| 1 | Alice |  |") {
		t.Errorf("short row should be padded, got:\n%s", got)
	}
}

func TestConvertResultMarkdownEscapesPipes(t *testing.T) {
	got, err := ConvertResult([]string{"col"}, [][]string{{"a|b"}}, Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `a\|b`) {
		t.Errorf("pipe should be escaped, got:\n%s", got)
	}
}

func TestConvertResultCSV(t *testing.T) {
	got, err := ConvertResult(resultHeaders, resultRows, CSV)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines, got %d:\n%s", len(lines), got)
	}
	if lines[0] != "id,name,email" {
		t.Errorf("CSV header = %q", lines[0])
	}
	if lines[1] != "1,Alice,alice@example.com" {
		t.Errorf("CSV row = %q", lines[1])
	}
}

func TestConvertResultCSVQuotesSpecialChars(t *testing.T) {
	// Values containing commas must be quoted by encoding/csv.
	got, err := ConvertResult([]string{"a", "b"}, [][]string{{"x,y", "z"}}, CSV)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"x,y"`) {
		t.Errorf("comma value should be quoted, got:\n%s", got)
	}
}

func TestConvertResultJSON(t *testing.T) {
	got, err := ConvertResult(resultHeaders, resultRows, JSON)
	if err != nil {
		t.Fatal(err)
	}

	var records []map[string]string
	if err := json.Unmarshal([]byte(got), &records); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, got)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "Alice" || records[1]["email"] != "bob@example.com" {
		t.Errorf("unexpected records: %+v", records)
	}
}

func TestConvertResultJSONPadsMissingCells(t *testing.T) {
	// When a row is shorter than headers, missing keys map to empty strings.
	got, err := ConvertResult(resultHeaders, [][]string{{"1", "Alice"}}, JSON)
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]string
	if err := json.Unmarshal([]byte(got), &records); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if records[0]["email"] != "" {
		t.Errorf("missing cell should map to empty string, got %q", records[0]["email"])
	}
}

func TestConvertResultJSONEmptyRows(t *testing.T) {
	got, err := ConvertResult(resultHeaders, nil, JSON)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "[]" {
		t.Errorf("expected [] for no rows, got %q", got)
	}
}

func TestConvertResultPlainReturnsError(t *testing.T) {
	_, err := ConvertResult(resultHeaders, resultRows, Plain)
	if err == nil {
		t.Error("ConvertResult with Plain should return an error")
	}
}

func TestConvertResultPDFReturnsError(t *testing.T) {
	_, err := ConvertResult(resultHeaders, resultRows, PDF)
	if err == nil {
		t.Error("ConvertResult with PDF should return an error")
	}
}

func TestConvertResultUnsupportedFormat(t *testing.T) {
	_, err := ConvertResult(resultHeaders, resultRows, Type("xml"))
	if err == nil {
		t.Error("ConvertResult with unsupported format should return an error")
	}
}

func TestWritePDFResult(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result.pdf")

	if err := WritePDFResult(resultHeaders, resultRows, path); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("PDF permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestWritePDFResultEmptyHeaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pdf")
	err := WritePDFResult(nil, resultRows, path)
	if err == nil {
		t.Fatal("WritePDFResult with no headers should return an error")
	}
	if !strings.Contains(err.Error(), "no data to export") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWritePDFResultInvalidPath(t *testing.T) {
	err := WritePDFResult(resultHeaders, resultRows, "/nonexistent/dir/f.pdf")
	if err == nil {
		t.Fatal("WritePDFResult with invalid path should return an error")
	}
}

func TestWritePDFResultNonASCII(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonascii.pdf")
	headers := []string{"id", "名前"}
	rows := [][]string{{"1", "太郎"}, {"2", "café"}}
	if err := WritePDFResult(headers, rows, path); err != nil {
		t.Fatalf("WritePDFResult with non-ASCII should not error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}

func TestWritePDFResultTruncatesLargeData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.pdf")

	rows := make([][]string, pdfMaxRows+5)
	for i := range rows {
		rows[i] = []string{"x", "y"}
	}
	if err := WritePDFResult([]string{"a", "b"}, rows, path); err != nil {
		t.Fatalf("WritePDFResult with >pdfMaxRows rows: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}
