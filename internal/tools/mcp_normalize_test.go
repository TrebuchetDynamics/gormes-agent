package tools

import (
	"encoding/json"
	"testing"
)

func TestRenderMCPCallResult_ErrorEmbeddedResourceText(t *testing.T) {
	result, err := parseMCPCallResult(json.RawMessage(`{
		"content":[{"type":"resource","resource":{"uri":"mem://err","mimeType":"text/plain","text":"quota exceeded for workspace W1"}}],
		"isError":true
	}`))
	if err != nil {
		t.Fatalf("parseMCPCallResult returned error: %v", err)
	}

	if got, want := RenderMCPCallResult(result, "test-server"), "quota exceeded for workspace W1"; got != want {
		t.Fatalf("RenderMCPCallResult() = %q, want %q", got, want)
	}
}

func TestRenderMCPCallResult_EmptyErrorFallback(t *testing.T) {
	result, err := parseMCPCallResult(json.RawMessage(`{"content":[],"isError":true}`))
	if err != nil {
		t.Fatalf("parseMCPCallResult returned error: %v", err)
	}

	if got, want := RenderMCPCallResult(result, "test-server"), "MCP tool returned an error"; got != want {
		t.Fatalf("RenderMCPCallResult() = %q, want %q", got, want)
	}
}
