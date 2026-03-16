package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tabularInput = `+----+-------+------------------+
| id | name  | email            |
+----+-------+------------------+
|  1 | Alice | alice@example.com|
|  2 | Bob   | bob@example.com  |
+----+-------+------------------+
`

const tsvInput = "id\tname\temail\n1\tAlice\talice@example.com\n2\tBob\tbob@example.com\n"

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  Type
		err   bool
	}{
		{"plain", Plain, false},
		{"", Plain, false},
		{"markdown", Markdown, false},
		{"md", Markdown, false},
		{"csv", CSV, false},
		{"json", JSON, false},
		{"pdf", PDF, false},
		{"xml", "", true},
	}

	for _, tt := range tests {
		got, err := Parse(tt.input)
		if tt.err && err == nil {
			t.Errorf("Parse(%q) expected error", tt.input)
		}
		if !tt.err && got != tt.want {
			t.Errorf("Parse(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestConvertMarkdownTabular(t *testing.T) {
	got, err := Convert(tabularInput, Markdown)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "| id | name | email |") {
		t.Errorf("markdown header missing, got:\n%s", got)
	}
	if !strings.Contains(got, "| --- | --- | --- |") {
		t.Errorf("markdown separator missing, got:\n%s", got)
	}
	if !strings.Contains(got, "| Alice |") {
		t.Errorf("markdown data row missing, got:\n%s", got)
	}
}

func TestConvertMarkdownTSV(t *testing.T) {
	got, err := Convert(tsvInput, Markdown)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, "| id | name | email |") {
		t.Errorf("markdown header missing, got:\n%s", got)
	}
}

func TestConvertCSV(t *testing.T) {
	got, err := Convert(tabularInput, CSV)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines, got %d:\n%s", len(lines), got)
	}
	if lines[0] != "id,name,email" {
		t.Errorf("CSV header = %q, want %q", lines[0], "id,name,email")
	}
}

func TestConvertJSONTabular(t *testing.T) {
	got, err := Convert(tabularInput, JSON)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `"id"`) {
		t.Errorf("JSON should contain id key, got:\n%s", got)
	}
	if !strings.Contains(got, `"Alice"`) {
		t.Errorf("JSON should contain Alice value, got:\n%s", got)
	}
	if !strings.Contains(got, `"email"`) {
		t.Errorf("JSON should contain email key, got:\n%s", got)
	}
}

func TestConvertJSONFromTSV(t *testing.T) {
	got, err := Convert(tsvInput, JSON)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(got, `"name"`) {
		t.Errorf("JSON should contain name key, got:\n%s", got)
	}
	if !strings.Contains(got, `"Bob"`) {
		t.Errorf("JSON should contain Bob value, got:\n%s", got)
	}
}

func TestConvertJSONEmpty(t *testing.T) {
	got, err := Convert("", JSON)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(got) != "[]" {
		t.Errorf("JSON of empty input should be [], got: %q", got)
	}
}

func TestConvertPlain(t *testing.T) {
	got, err := Convert(tabularInput, Plain)
	if err != nil {
		t.Fatal(err)
	}
	if got != tabularInput {
		t.Error("plain format should return input unchanged")
	}
}

func TestWritePDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")

	err := WritePDF(tabularInput, path)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}

func TestConvertPDFReturnsError(t *testing.T) {
	_, err := Convert(tabularInput, PDF)
	if err == nil {
		t.Fatal("Convert with PDF format should return an error")
	}
}

func TestWritePDFEmptyInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.pdf")

	err := WritePDF("", path)
	if err == nil {
		t.Fatal("WritePDF with empty input should return an error")
	}
	if !strings.Contains(err.Error(), "no data to export") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWritePDFInvalidPath(t *testing.T) {
	err := WritePDF(tabularInput, "/nonexistent/dir/f.pdf")
	if err == nil {
		t.Fatal("WritePDF with invalid path should return an error")
	}
}

func TestWriteFileInvalidPath(t *testing.T) {
	err := WriteFile("data", "/nonexistent/dir/f.csv")
	if err == nil {
		t.Fatal("WriteFile with invalid path should return an error")
	}
}

func TestConvertCSVFromTSV(t *testing.T) {
	got, err := Convert(tsvInput, CSV)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines, got %d:\n%s", len(lines), got)
	}
	if lines[0] != "id,name,email" {
		t.Errorf("CSV header = %q, want %q", lines[0], "id,name,email")
	}
	if !strings.Contains(lines[1], "Alice") {
		t.Errorf("CSV row missing Alice, got: %s", lines[1])
	}
}

func TestMarkdownEscapesPipes(t *testing.T) {
	input := "col1\tcol2\nfoo|bar\tbaz\n"
	got, err := Convert(input, Markdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `foo\|bar`) {
		t.Errorf("pipe character should be escaped in markdown, got:\n%s", got)
	}
}

func TestComputeColWidthsZeroTotal(t *testing.T) {
	// All empty headers and no rows should not panic
	headers := []string{"", "", ""}
	var rows [][]string
	widths := computeColWidths(headers, rows, 297, 210)
	if len(widths) != 3 {
		t.Fatalf("expected 3 widths, got %d", len(widths))
	}
	for i, w := range widths {
		if w <= 0 {
			t.Errorf("width[%d] = %f, expected positive value", i, w)
		}
	}
}

func TestWritePDFFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perms.pdf")

	err := WritePDF(tabularInput, path)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("PDF file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestContainsNonASCII(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello", false},
		{"", false},
		{"ASCII-123!", false},
		{"日本語", true},
		{"café", true},
		{"hello 世界", true},
	}

	for _, tt := range tests {
		got := containsNonASCII(tt.input)
		if got != tt.want {
			t.Errorf("containsNonASCII(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestTableHasNonASCII(t *testing.T) {
	if tableHasNonASCII([]string{"id", "name"}, [][]string{{"1", "Alice"}}) {
		t.Error("expected false for ASCII-only table")
	}
	if !tableHasNonASCII([]string{"id", "名前"}, nil) {
		t.Error("expected true for non-ASCII header")
	}
	if !tableHasNonASCII([]string{"id"}, [][]string{{"café"}}) {
		t.Error("expected true for non-ASCII cell")
	}
}

func TestWritePDFNonASCII(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonascii.pdf")

	input := "+----+--------+\n| id | name   |\n+----+--------+\n|  1 | 太郎   |\n|  2 | café   |\n+----+--------+\n"

	err := WritePDF(input, path)
	if err != nil {
		t.Fatalf("WritePDF with non-ASCII input should not error, got: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("PDF file is empty")
	}
}


func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.csv")

	content := "id,name\n1,Alice\n"
	err := WriteFile(content, path)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != content {
		t.Errorf("file content = %q, want %q", string(data), content)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions = %o, want 0600", info.Mode().Perm())
	}
}
