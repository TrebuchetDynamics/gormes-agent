package tools

import (
	"encoding/json"
	"os"
	"strings"
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

func TestRenderMCPCallResultWithOptions_MaterializesEmbeddedBlob(t *testing.T) {
	result, err := parseMCPCallResult(json.RawMessage(`{
		"content":[{"type":"resource","resource":{"uri":"file:///report.txt","mimeType":"text/plain","blob":"aGVsbG8="}}]
	}`))
	if err != nil {
		t.Fatalf("parseMCPCallResult returned error: %v", err)
	}
	dir := t.TempDir()
	got := RenderMCPCallResultWithOptions(result, MCPRenderOptions{ServerName: "files", ArtifactDir: dir})
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("artifact entries = %v, err=%v, want one", entries, err)
	}
	if !strings.Contains(got, "read_file or terminal tools") {
		t.Fatalf("RenderMCPCallResultWithOptions() = %q, want read guidance", got)
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
