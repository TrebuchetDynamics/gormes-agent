package callresult

import (
	"encoding/json"
	"fmt"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/content"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/jsonvalue"
)

// Result is the normalized envelope produced by an MCP `tools/call` response.
// Content captures the model-facing content blocks in the same content.Structured
// shape used by descriptor normalization/rendering so call sites do not need
// transport-specific decoders. StructuredContent preserves the optional
// protocol envelope as replayable JSON; callers can inspect it without relying
// on lossy text rendering. IsError mirrors the protocol's `isError` boolean: a
// true value means the tool reported a failure inside an otherwise successful
// JSON-RPC response (transport-level errors stay separate).
type Result struct {
	Content           []content.Structured
	StructuredContent json.RawMessage
	IsError           bool
}

// rawToolCallResult mirrors the on-the-wire shape of an MCP tools/call
// response. Content blocks are decoded into a representation-agnostic
// content.Structured slice via Parse.
type rawToolCallResult struct {
	Content           []rawToolCallContent `json:"content"`
	StructuredContent json.RawMessage      `json:"structuredContent"`
	IsError           bool                 `json:"isError"`
}

// rawToolCallContent captures the fields the structured content renderer needs
// from each MCP content block. Unknown fields are ignored so SDK extensions
// degrade gracefully instead of failing the parse.
type rawToolCallContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Name     string `json:"name,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	URI      string `json:"uri,omitempty"`
	Data     string `json:"data,omitempty"`
	Resource *struct {
		URI      string `json:"uri,omitempty"`
		MimeType string `json:"mimeType,omitempty"`
		Text     string `json:"text,omitempty"`
		Blob     string `json:"blob,omitempty"`
	} `json:"resource,omitempty"`
}

// Parse turns a raw `result` JSON document into a transport-free Result. Empty
// bodies are valid: tools that report success without content come back with
// IsError=false and zero Content blocks so callers can render them as a no-op.
func Parse(raw json.RawMessage) (Result, error) {
	if emptyResultRaw(raw) {
		return Result{}, nil
	}
	var decoded rawToolCallResult
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Result{}, fmt.Errorf("mcp call: parse result: %w", err)
	}
	out := Result{
		StructuredContent: normalizeStructuredContent(decoded.StructuredContent),
		IsError:           decoded.IsError,
	}
	if len(decoded.Content) == 0 {
		return out, nil
	}
	out.Content = make([]content.Structured, 0, len(decoded.Content))
	for _, block := range decoded.Content {
		out.Content = append(out.Content, normalizeContent(block))
	}
	return out, nil
}

// normalizeContent collapses a single content block into the shared structured
// content shape. Unknown kinds keep their type label so callers can branch on
// it, and resource blocks merge their nested `resource.uri` into the top-level
// URI field that content.Render inspects.
func emptyResultRaw(raw json.RawMessage) bool {
	return jsonvalue.NullishRaw(raw)
}

func normalizeStructuredContent(raw json.RawMessage) json.RawMessage {
	if jsonvalue.NullishRaw(raw) {
		return nil
	}
	return jsonvalue.CloneRaw(raw)
}

func normalizeContent(block rawToolCallContent) content.Structured {
	out := content.Structured{
		Kind:     block.Type,
		Text:     block.Text,
		Name:     block.Name,
		MimeType: block.MimeType,
		URI:      block.URI,
		Data:     block.Data,
	}
	if block.Resource != nil {
		if out.URI == "" {
			out.URI = block.Resource.URI
		}
		if out.MimeType == "" {
			out.MimeType = block.Resource.MimeType
		}
		if out.Text == "" {
			out.Text = block.Resource.Text
		}
		out.Blob = block.Resource.Blob
	}
	return out
}
