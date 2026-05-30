package mcp

import "encoding/json"

// RawTool is the verbatim tool envelope returned by an MCP server's
// tools/list response. InputSchema is preserved as raw JSON so downstream
// schema normalization can run separately without lossy round-tripping.
type RawTool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}
