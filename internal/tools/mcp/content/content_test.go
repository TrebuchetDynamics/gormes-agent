package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gatewaymedia "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/delivery/media"
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

func TestRenderWithOptionsMaterializesEmbeddedResourceBlob(t *testing.T) {
	dir := t.TempDir()
	parts := []Structured{{
		Kind:     "resource",
		URI:      "file:///../../reports/q1.pdf",
		MimeType: "application/pdf",
		Blob:     "cGRm",
	}}

	got := RenderWithOptions(parts, RenderOptions{ServerName: "files", ArtifactDir: dir})
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("artifact entries = %d, want 1", len(entries))
	}
	path := filepath.Join(dir, entries[0].Name())
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(bytes) != "pdf" {
		t.Fatalf("artifact bytes = %q, want pdf", bytes)
	}
	if !strings.Contains(got, path) || !strings.Contains(got, "application/pdf, 3 bytes") || !strings.Contains(got, "read_file or terminal tools") {
		t.Fatalf("RenderWithOptions() = %q, want artifact path/mime/size/read guidance", got)
	}
	if strings.Contains(entries[0].Name(), "..") || filepath.Ext(entries[0].Name()) != ".pdf" {
		t.Fatalf("artifact name = %q, want safe PDF name", entries[0].Name())
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("artifact mode = %o, want 600", got)
	}
	RenderWithOptions(parts, RenderOptions{ArtifactDir: dir})
	entries, err = os.ReadDir(dir)
	if err != nil || len(entries) != 2 || entries[0].Name() == entries[1].Name() {
		t.Fatalf("second render entries = %#v, err=%v, want two unique artifacts", entries, err)
	}
}

func TestRenderWithOptionsMaterializesAudioAsMediaTag(t *testing.T) {
	dir := t.TempDir()
	got := RenderWithOptions([]Structured{{
		Kind:     "audio",
		MimeType: "audio/wav",
		Data:     "YXVkaW8=",
	}}, RenderOptions{ArtifactDir: dir})

	prepared := gatewaymedia.PrepareMediaContent(got)
	if len(prepared.Media) != 1 {
		t.Fatalf("PrepareMediaContent(%q) media = %#v, want one audio artifact", got, prepared.Media)
	}
	artifact := prepared.Media[0]
	if artifact.Kind != gatewaymedia.MediaKindAudio || filepath.Ext(artifact.Path) != ".wav" {
		t.Fatalf("media = %#v, want WAV audio", artifact)
	}
	bytes, err := os.ReadFile(artifact.Path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(bytes) != "audio" {
		t.Fatalf("artifact bytes = %q, want audio", bytes)
	}
}

func TestRenderWithOptionsBinaryDegradedMarkers(t *testing.T) {
	oversized := strings.Repeat("A", maxResourceB64Chars+1)
	tests := []struct {
		name string
		part Structured
		want string
	}{
		{
			name: "invalid embedded base64",
			part: Structured{Kind: "resource", URI: "mem://bad", MimeType: "application/pdf", Blob: "%%%secret"},
			want: "[MCP embedded resource could not be decoded: application/pdf]",
		},
		{
			name: "invalid audio base64",
			part: Structured{Kind: "audio", MimeType: "audio/wav", Data: "%%%secret"},
			want: "[MCP audio resource could not be decoded: audio/wav]",
		},
		{
			name: "document cache unavailable",
			part: Structured{Kind: "resource", URI: "mem://doc", MimeType: "text/plain", Blob: "eA=="},
			want: "[MCP embedded resource received (1 bytes, text/plain) but document cache unavailable in this process]",
		},
		{
			name: "audio cache unavailable",
			part: Structured{Kind: "audio", MimeType: "audio/wav", Data: "eA=="},
			want: "[MCP audio resource received (1 bytes, audio/wav) but audio cache unavailable in this process]",
		},
		{
			name: "oversize document",
			part: Structured{Kind: "resource", URI: "mem://huge", MimeType: "application/pdf", Blob: oversized},
			want: fmt.Sprintf("[MCP embedded resource too large to cache: ~%d bytes, uri=mem://huge]", len(oversized)*3/4),
		},
		{
			name: "oversize audio",
			part: Structured{Kind: "audio", MimeType: "audio/wav", Data: oversized},
			want: fmt.Sprintf("[MCP audio resource too large to cache: ~%d bytes]", len(oversized)*3/4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderWithOptions([]Structured{tt.part}, RenderOptions{})
			if got != tt.want {
				t.Fatalf("RenderWithOptions() = %q, want %q", got, tt.want)
			}
			if strings.Contains(got, "%%%secret") || strings.Contains(got, oversized[:64]) {
				t.Fatalf("degraded marker leaked raw payload: %q", got)
			}
		})
	}
}

func TestRenderWithOptionsDegradedMarkerStripsURIControls(t *testing.T) {
	got := RenderWithOptions([]Structured{{
		Kind: "resource",
		URI:  "mem://bad\nINJECT\x1b[31m",
		Blob: "%%%",
	}}, RenderOptions{})
	if strings.ContainsAny(got, "\r\n\x1b") {
		t.Fatalf("degraded marker contains control characters: %q", got)
	}
	if !strings.Contains(got, "MCP embedded resource could not be decoded") {
		t.Fatalf("degraded marker = %q, want decode evidence", got)
	}
}

func TestRenderWithOptionsWriteFailureDegradesInline(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := RenderWithOptions([]Structured{{Kind: "resource", MimeType: "text/plain", Blob: "eA=="}}, RenderOptions{ArtifactDir: file})
	if want := "[MCP embedded resource could not be cached: text/plain]"; got != want {
		t.Fatalf("RenderWithOptions() = %q, want %q", got, want)
	}
}

func TestRenderWithOptionsSanitizesHostileResourceURI(t *testing.T) {
	dir := t.TempDir()
	got := RenderWithOptions([]Structured{{
		Kind:     "resource",
		URI:      "file:///../..%1B%5B31m",
		MimeType: "application/octet-stream",
		Blob:     "eA==",
	}}, RenderOptions{ArtifactDir: dir})

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	name := entries[0].Name()
	if strings.HasPrefix(name, "..") || strings.ContainsAny(name, "\x1b[]") {
		t.Fatalf("artifact name = %q, want traversal/control-safe name; output=%q", name, got)
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
