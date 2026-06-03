package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

// runRequests feeds the given newline-delimited JSON-RPC lines through the
// server and returns the decoded responses in order.
func runRequests(t *testing.T, s *Server, lines ...string) []map[string]any {
	t.Helper()
	in := strings.NewReader(strings.Join(lines, "\n") + "\n")
	var out strings.Builder
	if err := s.Serve(in, &out); err != nil {
		t.Fatalf("Serve returned error: %v", err)
	}
	var responses []map[string]any
	for _, raw := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if raw == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("response is not valid JSON %q: %v", raw, err)
		}
		responses = append(responses, m)
	}
	return responses
}

func newTestServer() *Server {
	s := NewServer("mysh-test", "1.2.3")
	s.AddTool(Tool{
		Name:        "echo",
		Description: "echoes the message argument",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"message": map[string]any{"type": "string"}},
			"required":   []any{"message"},
		},
		Handler: func(args map[string]any) (string, error) {
			msg, _ := args["message"].(string)
			return "you said: " + msg, nil
		},
	})
	s.AddTool(Tool{
		Name:        "boom",
		Description: "always fails",
		Handler: func(map[string]any) (string, error) {
			return "", errFail
		},
	})
	return s
}

var errFail = &stringError{"kaboom"}

type stringError struct{ s string }

func (e *stringError) Error() string { return e.s }

func TestInitializeHandshake(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)
	if len(resp) != 1 {
		t.Fatalf("expected 1 response, got %d", len(resp))
	}
	result, ok := resp[0]["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp[0])
	}
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion not echoed: %v", result["protocolVersion"])
	}
	info, ok := result["serverInfo"].(map[string]any)
	if !ok || info["name"] != "mysh-test" || info["version"] != "1.2.3" {
		t.Errorf("unexpected serverInfo: %v", result["serverInfo"])
	}
	if _, ok := result["capabilities"].(map[string]any)["tools"]; !ok {
		t.Errorf("expected tools capability: %v", result["capabilities"])
	}
}

func TestInitializeDefaultsProtocolVersion(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	result := resp[0]["result"].(map[string]any)
	if result["protocolVersion"] != defaultProtocolVersion {
		t.Errorf("expected default protocol version, got %v", result["protocolVersion"])
	}
}

func TestToolsList(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resp[0]["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "echo" {
		t.Errorf("unexpected first tool: %v", first["name"])
	}
	if _, ok := first["inputSchema"].(map[string]any); !ok {
		t.Errorf("tool missing inputSchema: %v", first)
	}
}

func TestToolsListSuppliesEmptySchema(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	tools := resp[0]["result"].(map[string]any)["tools"].([]any)
	// "boom" was registered without an InputSchema.
	boom := tools[1].(map[string]any)
	schema, ok := boom["inputSchema"].(map[string]any)
	if !ok || schema["type"] != "object" {
		t.Errorf("expected default object schema, got %v", boom["inputSchema"])
	}
}

func TestToolsCallSuccess(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{"message":"hi"}}}`)
	result := resp[0]["result"].(map[string]any)
	if result["isError"] != nil {
		t.Errorf("did not expect isError: %v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if content["text"] != "you said: hi" {
		t.Errorf("unexpected content: %v", content)
	}
}

func TestToolsCallHandlerError(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"boom"}}`)
	result := resp[0]["result"].(map[string]any)
	if result["isError"] != true {
		t.Errorf("expected isError true, got %v", result)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), "kaboom") {
		t.Errorf("expected error text, got %v", content)
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"nope"}}`)
	result := resp[0]["result"].(map[string]any)
	if result["isError"] != true {
		t.Errorf("expected isError for unknown tool, got %v", result)
	}
}

func TestNotificationProducesNoResponse(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":6,"method":"ping"}`)
	// Only the ping (which has an id) should produce a response.
	if len(resp) != 1 {
		t.Fatalf("expected 1 response, got %d: %v", len(resp), resp)
	}
	if resp[0]["id"].(float64) != 6 {
		t.Errorf("expected response to ping id 6, got %v", resp[0]["id"])
	}
}

func TestUnknownMethodReturnsError(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s, `{"jsonrpc":"2.0","id":7,"method":"does/not/exist"}`)
	rpcErr, ok := resp[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error, got %v", resp[0])
	}
	if rpcErr["code"].(float64) != codeMethodNotFound {
		t.Errorf("expected method-not-found code, got %v", rpcErr["code"])
	}
}

func TestParseErrorReturnsResponse(t *testing.T) {
	s := newTestServer()
	resp := runRequests(t, s, `{not json`)
	rpcErr, ok := resp[0]["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected parse error response, got %v", resp[0])
	}
	if rpcErr["code"].(float64) != codeParseError {
		t.Errorf("expected parse error code, got %v", rpcErr["code"])
	}
}

func TestStringEnumSchemaRoundTrips(t *testing.T) {
	// Guards that a slightly complex schema survives JSON encoding intact.
	s := NewServer("x", "0")
	s.AddTool(Tool{
		Name: "fmt",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"format": map[string]any{"type": "string", "enum": []any{"a", "b"}},
			},
		},
		Handler: func(map[string]any) (string, error) { return "ok", nil },
	})
	resp := runRequests(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tool := resp[0]["result"].(map[string]any)["tools"].([]any)[0].(map[string]any)
	props := tool["inputSchema"].(map[string]any)["properties"].(map[string]any)
	enum := props["format"].(map[string]any)["enum"].([]any)
	if len(enum) != 2 || enum[0] != "a" {
		t.Errorf("enum did not round-trip: %v", enum)
	}
}
