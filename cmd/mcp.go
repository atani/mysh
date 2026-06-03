package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/atani/mysh/internal/config"
	"github.com/atani/mysh/internal/db"
	"github.com/atani/mysh/internal/format"
	"github.com/atani/mysh/internal/i18n"
	"github.com/atani/mysh/internal/mask"
	"github.com/atani/mysh/internal/mcp"
)

// mcpVersion is overridden from main via SetVersion so the MCP server reports
// the same build version as `mysh version`.
var mcpVersion = "dev"

// nonInteractive disables interactive prompts (e.g. the master password prompt).
// It is set while the MCP server runs, because stdin carries JSON-RPC traffic
// and must not be consumed by a terminal prompt.
var nonInteractive bool

// SetVersion lets package main inject the build version used by the MCP server's
// initialize handshake.
func SetVersion(v string) {
	if v != "" {
		mcpVersion = v
	}
}

// RunMCP starts an MCP server over stdio so AI coding agents can call mysh.
//
// The server intentionally exposes only read-oriented tools, and query output
// is masked exactly as it is for non-TTY CLI execution (production and staging
// connections with mask rules are masked; there is no way to request raw output
// over MCP). This keeps the AI-safety guarantee that motivates mysh: sensitive
// values are masked before they ever reach the agent.
func RunMCP(_ []string) error {
	nonInteractive = true
	srv := mcp.NewServer("mysh", mcpVersion)

	srv.AddTool(mcp.Tool{
		Name: "mysh_list_connections",
		Description: "List configured mysh database connections (names, environment, " +
			"host, database, SSH/Redash). Passwords and API keys are never returned. " +
			"Use this first to discover which connection names are available.",
		InputSchema: objectSchema(nil, nil),
		Handler:     mcpListConnections,
	})

	srv.AddTool(mcp.Tool{
		Name: "mysh_query",
		Description: "Run a read-only SQL query against a configured connection and return " +
			"the results. Sensitive columns are automatically masked for production and " +
			"staging connections (the same rules as non-TTY CLI output); raw output cannot " +
			"be requested over MCP. Prefer SELECT statements with a LIMIT.",
		InputSchema: objectSchema(map[string]any{
			"connection": map[string]any{
				"type":        "string",
				"description": "Connection name. Optional when exactly one connection is configured.",
			},
			"sql": map[string]any{
				"type":        "string",
				"description": "The SQL to execute (e.g. \"SELECT id, name FROM users LIMIT 10\").",
			},
			"format": map[string]any{
				"type":        "string",
				"enum":        []any{"markdown", "json", "csv", "plain"},
				"description": "Output format. Defaults to markdown.",
			},
		}, []string{"sql"}),
		Handler: mcpQuery,
	})

	srv.AddTool(mcp.Tool{
		Name:        "mysh_tables",
		Description: "List the tables in a configured connection's database.",
		InputSchema: objectSchema(map[string]any{
			"connection": map[string]any{
				"type":        "string",
				"description": "Connection name. Optional when exactly one connection is configured.",
			},
			"format": map[string]any{
				"type":        "string",
				"enum":        []any{"markdown", "json", "csv", "plain"},
				"description": "Output format. Defaults to markdown.",
			},
		}, nil),
		Handler: mcpTables,
	})

	srv.AddTool(mcp.Tool{
		Name:        "mysh_ping",
		Description: "Test connectivity to a configured connection and report latency.",
		InputSchema: objectSchema(map[string]any{
			"connection": map[string]any{
				"type":        "string",
				"description": "Connection name. Optional when exactly one connection is configured.",
			},
		}, nil),
		Handler: mcpPing,
	})

	return srv.Serve(os.Stdin, os.Stdout)
}

// objectSchema builds a JSON Schema object with the given properties and
// required fields. A nil properties map produces an empty object schema.
func objectSchema(properties map[string]any, required []string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = toAnySlice(required)
	}
	return schema
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// argString extracts a string argument, returning the empty string when absent.
func argString(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// mcpOutputFormat resolves the requested format argument, defaulting to
// markdown and rejecting PDF (which requires a file path and is meaningless for
// an agent response).
func mcpOutputFormat(args map[string]any) (format.Type, error) {
	raw := argString(args, "format")
	if raw == "" {
		return format.Markdown, nil
	}
	outFmt, err := format.Parse(raw)
	if err != nil {
		return "", err
	}
	if outFmt == format.PDF {
		return "", errors.New(i18n.T(i18n.McpErrPDFUnsupported))
	}
	return outFmt, nil
}

func mcpListConnections(_ map[string]any) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	if len(cfg.Connections) == 0 {
		return i18n.T(i18n.McpNoConnections), nil
	}

	type connInfo struct {
		Name     string `json:"name"`
		Env      string `json:"env,omitempty"`
		Kind     string `json:"kind"`
		Host     string `json:"host,omitempty"`
		Port     int    `json:"port,omitempty"`
		Database string `json:"database,omitempty"`
		User     string `json:"user,omitempty"`
		SSH      string `json:"ssh,omitempty"`
		Redash   string `json:"redash,omitempty"`
		Masked   bool   `json:"masking_enabled"`
	}

	infos := make([]connInfo, 0, len(cfg.Connections))
	for i := range cfg.Connections {
		c := &cfg.Connections[i]
		info := connInfo{Name: c.Name, Env: c.Env, Masked: c.HasMaskConfig()}
		if c.IsRedash() {
			info.Kind = "redash"
			info.Redash = c.Redash.URL
		} else {
			info.Kind = "mysql"
			info.Host = c.DB.Host
			info.Port = c.DB.Port
			info.Database = c.DB.Database
			info.User = c.DB.User
			if c.SSH != nil {
				info.SSH = fmt.Sprintf("%s@%s", c.SSH.User, c.SSH.Host)
			}
		}
		infos = append(infos, info)
	}

	data, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func mcpQuery(args map[string]any) (string, error) {
	sqlExpr := argString(args, "sql")
	if sqlExpr == "" {
		return "", errors.New(i18n.T(i18n.McpErrSQLRequired))
	}
	outFmt, err := mcpOutputFormat(args)
	if err != nil {
		return "", err
	}

	_, conn, err := findConnection(argString(args, "connection"))
	if err != nil {
		return "", err
	}

	// Masking is forced to the non-TTY (agent) policy. Raw output is never
	// available over MCP.
	shouldMask := conn.ShouldMask(false)

	if conn.IsRedash() {
		return mcpQueryRedash(conn, sqlExpr, shouldMask, outFmt)
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return "", err
	}
	defer rc.cleanup()

	if rc.isNative() {
		return mcpQueryNative(rc, conn, sqlExpr, shouldMask, outFmt)
	}
	return mcpQueryCLI(rc, conn, sqlExpr, shouldMask, outFmt)
}

func mcpQueryRedash(conn *config.Connection, sqlExpr string, shouldMask bool, outFmt format.Type) (string, error) {
	client, err := resolveRedashClient(conn)
	if err != nil {
		return "", err
	}
	result, err := client.Query(sqlExpr, conn.Redash.DataSourceID)
	if err != nil {
		return "", err
	}
	if result.Headers == nil {
		return i18n.T(i18n.McpQueryOK), nil
	}
	if shouldMask {
		maskStructured(result.Headers, result.Rows, conn)
	}
	return formatStructured(result.Headers, result.Rows, outFmt)
}

func mcpQueryNative(rc *resolvedConn, conn *config.Connection, sqlExpr string, shouldMask bool, outFmt format.Type) (string, error) {
	dbConn, err := rc.openDB()
	if err != nil {
		return "", err
	}
	defer func() { _ = dbConn.Close() }()

	stmts := db.SplitStatements(sqlExpr)
	var lastHeaders []string
	var lastRows [][]string
	for _, stmt := range stmts {
		headers, rows, err := db.Query(dbConn, stmt)
		if err != nil {
			return "", err
		}
		if headers != nil {
			lastHeaders = headers
			lastRows = rows
		}
	}
	if lastHeaders == nil {
		return i18n.T(i18n.McpQueryOK), nil
	}
	if shouldMask {
		maskStructured(lastHeaders, lastRows, conn)
	}
	return formatStructured(lastHeaders, lastRows, outFmt)
}

func mcpQueryCLI(rc *resolvedConn, conn *config.Connection, sqlExpr string, shouldMask bool, outFmt format.Type) (string, error) {
	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return "", err
	}
	defer cleanup()

	mysqlArgs = append(mysqlArgs, "-e", sqlExpr)

	c := exec.Command("mysql", mysqlArgs...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", stderr.String())
		}
		return "", err
	}

	output := stdout.String()
	if shouldMask {
		masked, _ := mask.ApplyToOutput(output, conn.Mask.Columns, conn.Mask.Patterns)
		output = masked
	}
	return format.Convert(output, outFmt)
}

func mcpTables(args map[string]any) (string, error) {
	outFmt, err := mcpOutputFormat(args)
	if err != nil {
		return "", err
	}
	_, conn, err := findConnection(argString(args, "connection"))
	if err != nil {
		return "", err
	}
	if conn.IsRedash() {
		return "", errors.New(i18n.T(i18n.McpErrTablesRedash))
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return "", err
	}
	defer rc.cleanup()

	if rc.isNative() {
		dbConn, err := rc.openDB()
		if err != nil {
			return "", err
		}
		defer func() { _ = dbConn.Close() }()
		headers, rows, err := db.Query(dbConn, "SHOW TABLES")
		if err != nil {
			return "", err
		}
		return formatStructured(headers, rows, outFmt)
	}

	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return "", err
	}
	defer cleanup()
	mysqlArgs = append(mysqlArgs, "-e", "SHOW TABLES")

	c := exec.Command("mysql", mysqlArgs...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s", stderr.String())
		}
		return "", err
	}
	return format.Convert(stdout.String(), outFmt)
}

func mcpPing(args map[string]any) (string, error) {
	_, conn, err := findConnection(argString(args, "connection"))
	if err != nil {
		return "", err
	}

	if conn.IsRedash() {
		client, err := resolveRedashClient(conn)
		if err != nil {
			return "", err
		}
		if err := client.Ping(); err != nil {
			return "", fmt.Errorf("%s: %w", fmt.Sprintf(i18n.T(i18n.McpPingFailedRedash), conn.Name), err)
		}
		return fmt.Sprintf(i18n.T(i18n.McpPingOKRedash), conn.Name), nil
	}

	rc, err := resolveConnection(conn)
	if err != nil {
		return "", err
	}
	defer rc.cleanup()

	if rc.isNative() {
		dbConn, err := rc.openDB()
		if err != nil {
			return "", fmt.Errorf("%s: %w", fmt.Sprintf(i18n.T(i18n.McpPingFailed), conn.Name), err)
		}
		defer func() { _ = dbConn.Close() }()
		if err := db.Ping(dbConn); err != nil {
			return "", fmt.Errorf("%s: %w", fmt.Sprintf(i18n.T(i18n.McpPingFailed), conn.Name), err)
		}
		return fmt.Sprintf(i18n.T(i18n.McpPingOK), conn.Name), nil
	}

	mysqlArgs, cleanup, err := rc.mysqlArgsWithPassword()
	if err != nil {
		return "", err
	}
	defer cleanup()
	mysqlArgs = append(mysqlArgs, "-e", "SELECT 1")
	c := exec.Command("mysql", mysqlArgs...)
	var stderr bytes.Buffer
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%s: %s", fmt.Sprintf(i18n.T(i18n.McpPingFailed), conn.Name), stderr.String())
		}
		return "", fmt.Errorf("%s: %w", fmt.Sprintf(i18n.T(i18n.McpPingFailed), conn.Name), err)
	}
	return fmt.Sprintf(i18n.T(i18n.McpPingOK), conn.Name), nil
}

// maskStructured applies the connection's mask rules to structured rows in place.
func maskStructured(headers []string, rows [][]string, conn *config.Connection) {
	if conn.Mask == nil {
		return
	}
	maskedCols := mask.FindMaskColumns(headers, conn.Mask.Columns, conn.Mask.Patterns)
	if len(maskedCols) == 0 {
		return
	}
	for _, row := range rows {
		for idx := range row {
			if maskedCols[idx] {
				row[idx] = mask.Value(row[idx])
			}
		}
	}
}

// formatStructured renders structured results to the requested format as a
// string. PDF is rejected upstream, so only plain/markdown/csv/json reach here.
func formatStructured(headers []string, rows [][]string, outFmt format.Type) (string, error) {
	if outFmt == format.Plain {
		return db.FormatTabular(headers, rows), nil
	}
	return format.ConvertResult(headers, rows, outFmt)
}
