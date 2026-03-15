package sqldump

import (
	"strings"
	"testing"
	"time"

	"github.com/atani/mysh/internal/mysql"
)

func TestGenerate_Basic(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"id", "name", "active"},
		Rows: [][]string{
			{"7", "widgets", "1"},
			{"8", "gadgets", "NULL"},
		},
	}

	ts := time.Date(2026, 3, 15, 12, 0, 0, 0, time.FixedZone("JST", 9*3600))
	got := Generate("products", result, Options{Where: "id IN (7,8)", Timestamp: ts})

	if !strings.Contains(got, "-- mysh slice: products WHERE id IN (7,8)") {
		t.Error("missing header comment")
	}
	if !strings.Contains(got, "INSERT INTO `products` (`id`, `name`, `active`) VALUES (7, 'widgets', 1);") {
		t.Errorf("unexpected first row:\n%s", got)
	}
	if !strings.Contains(got, "INSERT INTO `products` (`id`, `name`, `active`) VALUES (8, 'gadgets', NULL);") {
		t.Errorf("unexpected second row (NULL handling):\n%s", got)
	}
}

func TestGenerate_EmptyResult(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"id"},
		Rows:    nil,
	}

	got := Generate("accounts", result, Options{Where: "id=0"})
	if strings.Contains(got, "INSERT") {
		t.Error("should not contain INSERT for empty result")
	}
	if !strings.Contains(got, "-- mysh slice:") {
		t.Error("should still have comment header")
	}
}

func TestGenerate_NilResult(t *testing.T) {
	got := Generate("accounts", nil, Options{Where: "id=0"})
	if strings.Contains(got, "INSERT") {
		t.Error("should not contain INSERT for nil result")
	}
}

func TestGenerate_Escape(t *testing.T) {
	result := &mysql.QueryResult{
		Headers: []string{"name"},
		Rows: [][]string{
			{"O'Brien"},
			{`back\slash`},
		},
	}

	got := Generate("staff", result, Options{Where: "id=3"})
	if !strings.Contains(got, "VALUES ('O''Brien');") {
		t.Errorf("single quote not escaped:\n%s", got)
	}
	if !strings.Contains(got, `VALUES ('back\\slash');`) {
		t.Errorf("backslash not escaped:\n%s", got)
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"-42", true},
		{"3.14", true},
		{"-0.5", true},
		{"", false},
		{"abc", false},
		{"12a", false},
		{"-", false},
		{"1.2.3", false},
	}
	for _, tt := range tests {
		if got := isNumeric(tt.input); got != tt.want {
			t.Errorf("isNumeric(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
