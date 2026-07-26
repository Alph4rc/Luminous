// Package mcp implements a minimal MCP (Model Context Protocol) server over stdio.
//
// It supports the JSON-RPC 2.0 transport and the following MCP methods:
//   - initialize
//   - tools/list
//   - tools/call
//   - ping
package mcp

import (
	"encoding/json"

	"luminous/internal/skill"
)

// ───────────────────────── JSON-RPC 2.0 types ─────────────────────────

// jsonrpcRequest is a JSON-RPC 2.0 request or notification.
// When ID is nil the message is a notification and no response is expected.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse is a JSON-RPC 2.0 success response.
type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Result  any    `json:"result"`
}

// jsonrpcError is a JSON-RPC 2.0 error response.
type jsonrpcError struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Error   rpcError    `json:"error"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// isNotification returns true when the request has no id field.
func (r *jsonrpcRequest) isNotification() bool {
	return r.ID == nil
}

// ─────────────────────── MCP initialize types ─────────────────────────

// InitializeParams is sent by the client during the initialize handshake.
type InitializeParams struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    clientCapabilities `json:"capabilities"`
	ClientInfo      info              `json:"clientInfo"`
}

type clientCapabilities struct{}

// InitializeResult is returned by the server after a successful initialize.
type InitializeResult struct {
	ProtocolVersion string            `json:"protocolVersion"`
	Capabilities    serverCapabilities `json:"capabilities"`
	ServerInfo      info              `json:"serverInfo"`
}

type serverCapabilities struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct{}

type info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ─────────────────────── MCP tools types ──────────────────────────────

// ListToolsResult is the response to a tools/list request.
type ListToolsResult struct {
	Tools []skill.ToolDef `json:"tools"`
}

// CallToolParams is sent by the client to invoke a tool.
type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// CallToolResult is returned after a successful tool invocation.
// When IsError is true the tool execution itself failed (not the MCP call).
type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem represents a single piece of content in a tool result.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}
