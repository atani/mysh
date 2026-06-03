// Package mcp implements a minimal Model Context Protocol (MCP) server over
// stdio using newline-delimited JSON-RPC 2.0. It has no external dependencies
// so mysh can expose its commands to AI coding agents (Claude Code, Cursor,
// etc.) without growing its dependency footprint.
//
// Only the subset of MCP needed to expose tools is implemented: the
// initialize handshake, tools/list, and tools/call. That is enough for an
// agent to discover and invoke mysh's read-oriented tools.
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// defaultProtocolVersion is the MCP protocol version advertised when the client
// does not request a specific one. The server echoes the client's requested
// version when present to maximize compatibility.
const defaultProtocolVersion = "2024-11-05"

// ToolHandler executes a tool call. arguments holds the decoded "arguments"
// object from the tools/call request (may be nil). The returned string is sent
// back to the agent as text content. Returning an error marks the tool result
// as an error without tearing down the connection.
type ToolHandler func(arguments map[string]any) (string, error)

// Tool describes a single MCP tool exposed by the server.
type Tool struct {
	Name        string
	Description string
	// InputSchema is the JSON Schema describing the tool's arguments. It must
	// be a JSON object schema (i.e. {"type":"object", ...}).
	InputSchema map[string]any
	Handler     ToolHandler
}

// Server is a minimal stdio MCP server.
type Server struct {
	name    string
	version string
	tools   []Tool
}

// NewServer creates an MCP server that identifies itself with the given name
// and version during the initialize handshake.
func NewServer(name, version string) *Server {
	return &Server{name: name, version: version}
}

// AddTool registers a tool. Tools should be added before Serve is called.
func (s *Server) AddTool(t Tool) {
	s.tools = append(s.tools, t)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
)

// Serve reads newline-delimited JSON-RPC messages from in and writes responses
// to out until in reaches EOF. Each request produces at most one response;
// notifications (messages without an id) produce none.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	enc := json.NewEncoder(out)
	for {
		line, err := reader.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			if resp, ok := s.handleLine(trimmed); ok {
				if encErr := enc.Encode(resp); encErr != nil {
					return encErr
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// handleLine processes one JSON-RPC message and reports whether a response
// should be written.
func (s *Server) handleLine(line []byte) (rpcResponse, bool) {
	var req rpcRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return errorResponse(nil, codeParseError, "parse error"), true
	}

	// Notifications have no id and never receive a response.
	isNotification := len(req.ID) == 0
	if isNotification {
		return rpcResponse{}, false
	}

	result, rpcErr := s.dispatch(req)
	if rpcErr != nil {
		return errorResponseRaw(req.ID, rpcErr), true
	}
	return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, true
}

func (s *Server) dispatch(req rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return s.initializeResult(req.Params), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": s.toolDescriptors()}, nil
	case "tools/call":
		return s.callTool(req.Params)
	default:
		return nil, &rpcError{Code: codeMethodNotFound, Message: "method not found: " + req.Method}
	}
}

func (s *Server) initializeResult(params json.RawMessage) map[string]any {
	protocolVersion := defaultProtocolVersion
	if len(params) > 0 {
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(params, &p); err == nil && p.ProtocolVersion != "" {
			protocolVersion = p.ProtocolVersion
		}
	}
	return map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    s.name,
			"version": s.version,
		},
	}
}

func (s *Server) toolDescriptors() []map[string]any {
	descriptors := make([]map[string]any, 0, len(s.tools))
	for _, t := range s.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		descriptors = append(descriptors, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": schema,
		})
	}
	return descriptors
}

func (s *Server) callTool(params json.RawMessage) (any, *rpcError) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if len(params) > 0 {
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpcError{Code: codeInvalidParams, Message: "invalid params: " + err.Error()}
		}
	}
	if p.Name == "" {
		return nil, &rpcError{Code: codeInvalidParams, Message: "tools/call requires a tool name"}
	}

	for _, t := range s.tools {
		if t.Name == p.Name {
			text, err := t.Handler(p.Arguments)
			if err != nil {
				return toolResult("Error: "+err.Error(), true), nil
			}
			return toolResult(text, false), nil
		}
	}
	return toolResult(fmt.Sprintf("Error: unknown tool %q", p.Name), true), nil
}

// toolResult builds the MCP tools/call result payload. Tool-level failures are
// reported via isError rather than as JSON-RPC errors so the agent can read the
// message.
func toolResult(text string, isError bool) map[string]any {
	result := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}
	if isError {
		result["isError"] = true
	}
	return result
}

func errorResponse(id json.RawMessage, code int, message string) rpcResponse {
	return errorResponseRaw(id, &rpcError{Code: code, Message: message})
}

func errorResponseRaw(id json.RawMessage, err *rpcError) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: err}
}
