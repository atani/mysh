package mysql

import (
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

func TestParseOutputTabular(t *testing.T) {
	r := ParseOutput(tabularInput)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if len(r.Headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(r.Headers))
	}
	if r.Headers[0] != "id" || r.Headers[1] != "name" || r.Headers[2] != "email" {
		t.Errorf("unexpected headers: %v", r.Headers)
	}
	if len(r.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(r.Rows))
	}
	if r.Rows[0][1] != "Alice" {
		t.Errorf("expected Alice, got %q", r.Rows[0][1])
	}
}

func TestParseOutputTSV(t *testing.T) {
	r := ParseOutput(tsvInput)
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if len(r.Headers) != 3 {
		t.Fatalf("expected 3 headers, got %d", len(r.Headers))
	}
	if r.Headers[0] != "id" || r.Headers[1] != "name" || r.Headers[2] != "email" {
		t.Errorf("unexpected headers: %v", r.Headers)
	}
	if len(r.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(r.Rows))
	}
	if r.Rows[1][1] != "Bob" {
		t.Errorf("expected Bob, got %q", r.Rows[1][1])
	}
}

func TestParseOutputEmpty(t *testing.T) {
	r := ParseOutput("")
	if r != nil {
		t.Error("expected nil for empty input")
	}
}

func TestParseOutputSingleLine(t *testing.T) {
	r := ParseOutput("just one line")
	if r != nil {
		t.Error("expected nil for single-line input")
	}
}

func TestParseOutputShortTabular(t *testing.T) {
	// Tabular format with fewer than 4 lines should return nil
	input := "+---+\n| a |\n+---+\n"
	r := ParseOutput(input)
	if r != nil {
		t.Error("expected nil for short tabular input")
	}
}
