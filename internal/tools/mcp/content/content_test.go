package content

import (
	"strings"
	"testing"
)

func TestRenderToolCallResultStructuredContent(t *testing.T) {
	parts := []Structured{
		{Kind: "text", Text: "hello world"},
		{Kind: "image", MimeType: "image/png"},
		{Kind: "resource", URI: "file:///tmp/foo.txt"},
	}

	got := Render(parts)

	if !strings.Contains(got, "hello world") {
		t.Errorf("missing verbatim text part in %q", got)
	}
	if !strings.Contains(got, "[image: image/png]") {
		t.Errorf("missing image marker in %q", got)
	}
	if !strings.Contains(got, "[resource: file:///tmp/foo.txt]") {
		t.Errorf("missing resource marker in %q", got)
	}
}

func TestRenderForServerPreservesMixedResourceOrderAndWireName(t *testing.T) {
	parts := []Structured{
		{Kind: "text", Text: "File ID: F123"},
		{Kind: "resource_link", URI: "slack://files/F123", Name: "report.pdf", MimeType: "application/pdf"},
		{Kind: "resource", URI: "mem://notes", MimeType: "text/plain", Text: "embedded notes"},
	}

	got := RenderForServer(parts, "slack/team")
	want := "File ID: F123\n[MCP resource link: uri=slack://files/F123, name=report.pdf, mimeType=application/pdf — fetch it with mcp__slack_team__read_resource]\nembedded notes"
	if got != want {
		t.Fatalf("RenderForServer() = %q, want %q", got, want)
	}
}

func TestRenderToolCallResultUnknownContentKindFallsBackToText(t *testing.T) {
	parts := []Structured{
		{Kind: "unknown_xyz", Text: "fallback text"},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Render panicked: %v", r)
		}
	}()
	got := Render(parts)

	if !strings.Contains(got, "fallback text") {
		t.Errorf("expected fallback text in %q", got)
	}
	// no leak of a raw protocol envelope (e.g. JSON object syntax)
	if strings.Contains(got, "{") || strings.Contains(got, "}") {
		t.Errorf("rendered output leaks protocol envelope: %q", got)
	}
}
