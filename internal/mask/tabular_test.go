package mask

import (
	"strings"
	"testing"
)

// boxTable is mysql's default box-drawing tabular output.
const boxTable = `+----+-------+-------------------+
| id | name  | email             |
+----+-------+-------------------+
|  1 | Alice | alice@example.com |
|  2 | Bob   | bob@example.com   |
+----+-------+-------------------+`

func TestTabularOutputBoxFormatMasksValueColumn(t *testing.T) {
	got := TabularOutput(boxTable, map[int]bool{2: true})
	lines := strings.Split(got, "\n")

	// Header (line index 1) must stay untouched.
	if !strings.Contains(lines[1], "email") {
		t.Errorf("header changed: %q", lines[1])
	}
	// Data rows: email masked, name preserved.
	if !strings.Contains(got, "a***@example.com") {
		t.Errorf("alice email not masked:\n%s", got)
	}
	if !strings.Contains(got, "b***@example.com") {
		t.Errorf("bob email not masked:\n%s", got)
	}
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("name column should be preserved:\n%s", got)
	}
	// Separator rows preserved.
	if !strings.HasPrefix(lines[0], "+") {
		t.Errorf("separator row changed: %q", lines[0])
	}
}

func TestTabularOutputBoxFormatPreservesColumnWidth(t *testing.T) {
	// Masking must not change the table's overall line length so the box
	// borders stay aligned.
	got := TabularOutput(boxTable, map[int]bool{2: true})
	origLines := strings.Split(boxTable, "\n")
	gotLines := strings.Split(got, "\n")
	if len(origLines) != len(gotLines) {
		t.Fatalf("line count changed: %d -> %d", len(origLines), len(gotLines))
	}
	for i := range origLines {
		if len([]rune(origLines[i])) != len([]rune(gotLines[i])) {
			t.Errorf("line %d width changed: %d -> %d\n%q\n%q",
				i, len([]rune(origLines[i])), len([]rune(gotLines[i])), origLines[i], gotLines[i])
		}
	}
}

func TestTabularOutputNoMaskedColumns(t *testing.T) {
	if got := TabularOutput(boxTable, map[int]bool{}); got != boxTable {
		t.Error("no masked columns should return input unchanged")
	}
	if got := TabularOutput(boxTable, nil); got != boxTable {
		t.Error("nil masked columns should return input unchanged")
	}
}

func TestTabularOutputTooFewLines(t *testing.T) {
	// Fewer than 2 lines: nothing to mask, returned as-is.
	in := "only-one-line"
	if got := TabularOutput(in, map[int]bool{0: true}); got != in {
		t.Errorf("single line should be unchanged, got %q", got)
	}
}

func TestMaskTabularFormatTooFewLines(t *testing.T) {
	// A box-drawing prefix but fewer than 4 lines should be returned joined,
	// unchanged.
	in := "+---+\n| a |\n+---+"
	got := TabularOutput(in, map[int]bool{0: true})
	if got != in {
		t.Errorf("3-line box should be unchanged, got %q", got)
	}
}

func TestTabularOutputMaskFirstColumn(t *testing.T) {
	got := TabularOutput(boxTable, map[int]bool{0: true})
	// id column values 1 and 2 are <=3 chars -> masked to *** then truncated
	// to the narrow 2-char column width, yielding "**".
	if strings.Contains(got, " 1 ") || strings.Contains(got, " 2 ") {
		t.Errorf("id column should be masked (no raw 1/2):\n%s", got)
	}
	if !strings.Contains(got, "**") {
		t.Errorf("id column should contain mask chars:\n%s", got)
	}
	// email column untouched.
	if !strings.Contains(got, "alice@example.com") {
		t.Errorf("email should be preserved:\n%s", got)
	}
}

func TestTabularOutputPreservesTrailingBlankLine(t *testing.T) {
	// mysql output often ends with a trailing newline -> empty last element.
	got := TabularOutput(boxTable+"\n", map[int]bool{2: true})
	if !strings.HasSuffix(got, "\n") {
		t.Error("trailing blank line should be preserved")
	}
}
