// Package skill defines the Tool interface and registry for MCP tools.
package skill

import "context"

// ToolDef is the MCP-compatible metadata for a tool.
// It matches the format expected by tools/list responses.
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema is a JSON Schema describing the tool's parameters.
type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]PropertyDef `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// PropertyDef describes a single property in a tool's input schema.
type PropertyDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Tool is the interface that every MCP tool must implement.
type Tool interface {
	// Definition returns the tool's MCP metadata used in tools/list responses.
	Definition() ToolDef

	// Execute runs the tool with the given arguments and returns a JSON string.
	// The returned string should be valid JSON so MCP clients can parse it.
	// On failure the error message is wrapped in a user-readable format.
	Execute(ctx context.Context, args map[string]any) (string, error)
}
