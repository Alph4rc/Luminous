package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"luminous/internal/skill"
)

// Handler processes MCP JSON-RPC messages and dispatches them to the
// appropriate method handlers. It is I/O agnostic and can be used with
// any transport (stdio, SSE, etc.).
type Handler struct {
	registry    *skill.Registry
	serverInfo  info
	initialized bool
}

// NewHandler creates an MCP Handler with the given tool registry.
func NewHandler(registry *skill.Registry, name, version string) *Handler {
	return &Handler{
		registry: registry,
		serverInfo: info{
			Name:    name,
			Version: version,
		},
	}
}

// ProcessMessage handles a single JSON-RPC message. It returns the response
// payload (nil for notifications) and an error only for protocol-level
// failures that should cause the server to stop.
func (h *Handler) ProcessMessage(data []byte) ([]byte, error) {
	var req jsonrpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return h.errorResponse(0, ErrCodeParse, "failed to parse JSON-RPC request: "+err.Error())
	}

	// Stderr logging — never to stdout which is the MCP transport channel.
	slog.Debug("received mcp request", "method", req.Method, "hasID", req.ID != nil)

	// Notifications — no response expected.
	if req.isNotification() {
		if err := h.handleNotification(req.Method, req.Params); err != nil {
			slog.Warn("notification handler error", "method", req.Method, "error", err)
		}
		return nil, nil
	}

	result, rpcErr := h.dispatch(req.Method, req.Params)
	if rpcErr != nil {
		return json.Marshal(jsonrpcError{
			JSONRPC: "2.0",
			ID:      *req.ID,
			Error:   *rpcErr,
		})
	}

	return json.Marshal(jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      *req.ID,
		Result:  result,
	})
}

func (h *Handler) dispatch(method string, params json.RawMessage) (any, *rpcError) {
	switch method {
	case "initialize":
		return h.handleInitialize(params)
	case "tools/list":
		return h.handleListTools(params)
	case "tools/call":
		return h.handleCallTool(params)
	case "ping":
		return struct{}{}, nil
	default:
		return nil, &rpcError{
			Code:    ErrCodeMethodNotFound,
			Message: fmt.Sprintf("unknown method: %s", method),
		}
	}
}

func (h *Handler) handleNotification(method string, _ json.RawMessage) error {
	switch method {
	case "notifications/initialized":
		slog.Debug("client sent initialized notification")
		return nil
	case "notifications/cancelled":
		slog.Debug("client sent cancelled notification")
		return nil
	default:
		return fmt.Errorf("unknown notification method: %s", method)
	}
}

func (h *Handler) handleInitialize(params json.RawMessage) (any, *rpcError) {
	var initParams InitializeParams
	if err := json.Unmarshal(params, &initParams); err != nil {
		return nil, &rpcError{
			Code:    ErrCodeInvalidParams,
			Message: "invalid initialize params: " + err.Error(),
		}
	}

	slog.Info("client initialized",
		"clientName", initParams.ClientInfo.Name,
		"clientVersion", initParams.ClientInfo.Version,
		"protocolVersion", initParams.ProtocolVersion,
	)

	h.initialized = true

	return InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: serverCapabilities{
			Tools: &toolsCapability{},
		},
		ServerInfo: h.serverInfo,
	}, nil
}

func (h *Handler) handleListTools(_ json.RawMessage) (any, *rpcError) {
	tools := h.registry.List()
	return ListToolsResult{Tools: tools}, nil
}

func (h *Handler) handleCallTool(params json.RawMessage) (any, *rpcError) {
	var callParams CallToolParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, &rpcError{
			Code:    ErrCodeInvalidParams,
			Message: "invalid tools/call params: " + err.Error(),
		}
	}

	tool, err := h.registry.Get(callParams.Name)
	if err != nil {
		return CallToolResult{
			Content: []ContentItem{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, execErr := tool.Execute(ctx, callParams.Arguments)
	if execErr != nil {
		slog.Warn("tool execution failed", "tool", callParams.Name, "error", execErr)
		return CallToolResult{
			Content: []ContentItem{{Type: "text", Text: execErr.Error()}},
			IsError: true,
		}, nil
	}

	return CallToolResult{
		Content: []ContentItem{{Type: "text", Text: result}},
	}, nil
}

// errorResponse builds a JSON-RPC error response for cases where we cannot
// even parse the request id (e.g. malformed JSON).
func (h *Handler) errorResponse(id int64, code int, message string) ([]byte, error) {
	return json.Marshal(jsonrpcError{
		JSONRPC: "2.0",
		ID:      id,
		Error: rpcError{
			Code:    code,
			Message: message,
		},
	})
}
