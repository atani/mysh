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
