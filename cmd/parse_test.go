package cmd

import (
	"strings"
	"testing"

	"github.com/atani/mysh/internal/format"
)

func TestParseRunQueryArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, o runQueryOptions)
	}{
		{
			name: "inline sql with name",
			args: []string{"prod", "-e", "SELECT 1"},
			check: func(t *testing.T, o runQueryOptions) {
				if o.connName != "prod" || o.sqlExpr != "SELECT 1" {
					t.Errorf("got %+v", o)
				}
			},
		},
		{
			name: "format and output flags",
			args: []string{"prod", "-e", "SELECT 1", "--format", "csv", "-o", "out.csv"},
			check: func(t *testing.T, o runQueryOptions) {
				if o.outFmt != format.CSV || o.outputFile != "out.csv" {
					t.Errorf("got %+v", o)
				}
			},
		},
		{
			name: "mask and raw flags",
			args: []string{"prod", "-e", "SELECT 1", "--mask", "--raw"},
			check: func(t *testing.T, o runQueryOptions) {
				if !o.forceMask || !o.forceRaw {
					t.Errorf("flags not set: %+v", o)
				}
			},
		},
		{name: "format requires value", args: []string{"--format"}, wantErr: "--format requires"},
		{name: "output requires value", args: []string{"-o"}, wantErr: "-o requires"},
		{name: "e requires value", args: []string{"-e"}, wantErr: "usage: mysh run"},
		{name: "invalid format", args: []string{"-e", "x", "--format", "bogus"}, wantErr: ""},
		{name: "pdf requires output", args: []string{"-e", "x", "--format", "pdf"}, wantErr: "PDF format requires"},
		{name: "no sql", args: []string{"prod"}, wantErr: "usage: mysh run"},
		{name: "too many positional", args: []string{"a", "b", "c"}, wantErr: "usage: mysh run"},
		{name: "missing sql file", args: []string{"prod", "/no/such/file.sql"}, wantErr: "SQL file not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := parseRunQueryArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if tt.name == "invalid format" {
				if err == nil {
					t.Fatal("expected format parse error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}

func TestParseRunQueryArgsFileResolution(t *testing.T) {
	dir := t.TempDir()
	sqlPath := dir + "/q.sql"
	if err := writeFile(t, sqlPath, "SELECT 1;"); err != nil {
		t.Fatal(err)
	}
	// A single positional that is an existing file becomes sqlFile, not connName.
	o, err := parseRunQueryArgs([]string{sqlPath})
	if err != nil {
		t.Fatalf("parseRunQueryArgs: %v", err)
	}
	if o.sqlFile != sqlPath || o.connName != "" {
		t.Errorf("file should resolve to sqlFile: %+v", o)
	}
}

func TestParseSliceArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, o sliceOptions)
	}{
		{
			name: "valid",
			args: []string{"prod", "users", "--where", "id > 0", "-o", "out.sql", "--raw"},
			check: func(t *testing.T, o sliceOptions) {
				if o.connName != "prod" || o.tableName != "users" || o.where != "id > 0" || o.outputFile != "out.sql" || !o.forceRaw {
					t.Errorf("got %+v", o)
				}
			},
		},
		{name: "where requires value", args: []string{"--where"}, wantErr: "--where requires"},
		{name: "output requires value", args: []string{"prod", "users", "--where", "x", "-o"}, wantErr: "-o requires"},
		{name: "too few positional", args: []string{"prod"}, wantErr: "usage: mysh slice"},
		{name: "missing where", args: []string{"prod", "users"}, wantErr: "--where is required"},
		{name: "backtick in table", args: []string{"prod", "us`ers", "--where", "id>0"}, wantErr: "backtick"},
		{name: "semicolon in where", args: []string{"prod", "users", "--where", "id>0; DROP TABLE x"}, wantErr: "semicolons"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := parseSliceArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}

func TestParseTablesArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(t *testing.T, o tablesOptions)
	}{
		{
			name: "valid",
			args: []string{"prod", "--format", "markdown", "-o", "t.md"},
			check: func(t *testing.T, o tablesOptions) {
				if o.connName != "prod" || o.outFmt != format.Markdown || o.outputFile != "t.md" {
					t.Errorf("got %+v", o)
				}
			},
		},
		{name: "format requires value", args: []string{"--format"}, wantErr: "--format requires"},
		{name: "output requires value", args: []string{"-o"}, wantErr: "-o requires"},
		{name: "unexpected arg", args: []string{"a", "b"}, wantErr: "unexpected argument"},
		{name: "pdf requires output", args: []string{"prod", "--format", "pdf"}, wantErr: "PDF format requires"},
		{name: "invalid format", args: []string{"--format", "bogus"}, wantErr: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, err := parseTablesArgs(tt.args)
			if tt.name == "invalid format" {
				if err == nil {
					t.Fatal("expected format parse error")
				}
				return
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, o)
			}
		})
	}
}
