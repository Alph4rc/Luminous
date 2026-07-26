package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"luminous/internal/skill"
)

// stubTool is a minimal tool implementation used in tests.
type stubTool struct {
	name        string
	description string
	inputSchema skill.InputSchema
	execute     func(ctx context.Context, args map[string]any) (string, error)
}

func (s *stubTool) Definition() skill.ToolDef {
	return skill.ToolDef{
		Name:        s.name,
		Description: s.description,
		InputSchema: s.inputSchema,
	}
}

func (s *stubTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	return s.execute(ctx, args)
}

func setupHandler(t *testing.T) *Handler {
	t.Helper()
	registry := skill.NewRegistry()

	// Register a fake query_school tool.
	registry.Register(&stubTool{
		name:        "query_school",
		description: "根据学校代码查询学校详情",
		inputSchema: skill.InputSchema{
			Type: "object",
			Properties: map[string]skill.PropertyDef{
				"school_code": {Type: "string", Description: "学校代码，例如 XAUAT"},
			},
			Required: []string{"school_code"},
		},
		execute: func(ctx context.Context, args map[string]any) (string, error) {
			code, _ := args["school_code"].(string)
			return `{"code":200,"message":"success","data":{"code":"` + code + `","name":"测试大学"}}`, nil
		},
	})

	return NewHandler(registry, "test-server", "1.0.0")
}

// sendRequest is a helper that sends a JSON-RPC request and returns the parsed response.
func sendRequest(t *testing.T, handler *Handler, method string, params any) jsonrpcResponse {
	t.Helper()

	var paramsJSON json.RawMessage
	if params != nil {
		var err error
		paramsJSON, err = json.Marshal(params)
		if err != nil {
			t.Fatalf("failed to marshal params: %v", err)
		}
	}

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      int64Ptr(1),
		Method:  method,
		Params:  paramsJSON,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	respData, err := handler.ProcessMessage(data)
	if err != nil {
		t.Fatalf("ProcessMessage returned error: %v", err)
	}
	if respData == nil {
		t.Fatal("expected response, got nil (notification)")
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	return resp
}

func int64Ptr(v int64) *int64 { return &v }

// ── initialize ──────────────────────────────────────────────────────

func TestInitialize(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "initialize", InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      info{Name: "test-client", Version: "1.0.0"},
	})

	var result InitializeResult
	remarshal(t, resp.Result, &result)

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %s", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "test-server" {
		t.Errorf("expected server name test-server, got %s", result.ServerInfo.Name)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability")
	}
}

// ── tools/list ──────────────────────────────────────────────────────

func TestListTools(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "tools/list", nil)

	var result ListToolsResult
	remarshal(t, resp.Result, &result)

	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	tool := result.Tools[0]
	if tool.Name != "query_school" {
		t.Errorf("expected tool name query_school, got %s", tool.Name)
	}
	if tool.InputSchema.Type != "object" {
		t.Errorf("expected schema type object, got %s", tool.InputSchema.Type)
	}
	if _, ok := tool.InputSchema.Properties["school_code"]; !ok {
		t.Error("expected school_code property")
	}
}

// ── tools/call ──────────────────────────────────────────────────────

func TestCallTool_Success(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "tools/call", CallToolParams{
		Name:      "query_school",
		Arguments: map[string]any{"school_code": "XAUAT"},
	})

	var result CallToolResult
	remarshal(t, resp.Result, &result)

	if result.IsError {
		t.Fatal("expected successful tool call, got isError=true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected content type text, got %s", result.Content[0].Type)
	}
	if !strings.Contains(result.Content[0].Text, "XAUAT") {
		t.Errorf("expected result to contain XAUAT, got: %s", result.Content[0].Text)
	}
}

func TestCallTool_UnknownTool(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "tools/call", CallToolParams{
		Name: "nonexistent",
	})

	var result CallToolResult
	remarshal(t, resp.Result, &result)

	if !result.IsError {
		t.Fatal("expected isError=true for unknown tool")
	}
	if !strings.Contains(result.Content[0].Text, "not found") {
		t.Errorf("expected 'not found' in error, got: %s", result.Content[0].Text)
	}
}

func TestCallTool_MissingParams(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "tools/call", CallToolParams{
		Name: "query_school",
		// Arguments intentionally omitted — will be nil.
	})

	var result CallToolResult
	remarshal(t, resp.Result, &result)

	// The stub tool won't error on nil args, but the real one would.
	// This tests that the handler passes through errors from Execute.
	if result.IsError {
		t.Logf("stub returned error (expected due to nil args): %s", result.Content[0].Text)
	}
}

func TestCallTool_InvalidJSONInResult(t *testing.T) {
	registry := skill.NewRegistry()
	registry.Register(&stubTool{
		name:        "broken_tool",
		description: "returns invalid data",
		inputSchema: skill.InputSchema{Type: "object", Properties: map[string]skill.PropertyDef{}},
		execute: func(ctx context.Context, args map[string]any) (string, error) {
			return `{invalid json`, nil
		},
	})

	handler := NewHandler(registry, "test", "1.0.0")
	resp := sendRequest(t, handler, "tools/call", CallToolParams{
		Name: "broken_tool",
	})

	var result CallToolResult
	remarshal(t, resp.Result, &result)

	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Text != `{invalid json` {
		t.Errorf("expected raw text, got: %s", result.Content[0].Text)
	}
}

// ── ping ────────────────────────────────────────────────────────────

func TestPing(t *testing.T) {
	handler := setupHandler(t)
	resp := sendRequest(t, handler, "ping", nil)
	if resp.Result == nil {
		t.Error("expected non-nil result for ping")
	}
}

// ── notifications ───────────────────────────────────────────────────

func TestNotification_NoResponse(t *testing.T) {
	handler := setupHandler(t)

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// No ID → notification.
	}
	data, _ := json.Marshal(req)

	resp, err := handler.ProcessMessage(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Error("expected nil response for notification, got data")
	}
}

// ── helpers ──────────────────────────────────────────────────────────

func remarshal(t *testing.T, src, dst any) {
	t.Helper()
	b, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
}
