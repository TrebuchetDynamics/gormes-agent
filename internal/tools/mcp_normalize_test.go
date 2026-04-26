package tools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeTools_SanitizesUnsafeNames(t *testing.T) {
	raw := []MCPRawTool{{
		Name:        "weather/get current",
		Description: "fetch weather",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}

	got := NormalizeTools("weather_srv", raw)

	if len(got.Tools) != 1 {
		t.Fatalf("Tools len = %d, want 1; rejected=%+v", len(got.Tools), got.Rejected)
	}
	if len(got.Rejected) != 0 {
		t.Fatalf("Rejected len = %d, want 0; %+v", len(got.Rejected), got.Rejected)
	}
	tool := got.Tools[0]
	if tool.Name != "weather_get_current" {
		t.Errorf("Name = %q, want %q", tool.Name, "weather_get_current")
	}
	if tool.SourceRaw.Name != "weather/get current" {
		t.Errorf("SourceRaw.Name = %q, want %q", tool.SourceRaw.Name, "weather/get current")
	}
	if tool.ServerName != "weather_srv" {
		t.Errorf("ServerName = %q, want %q", tool.ServerName, "weather_srv")
	}
	if tool.Description != "fetch weather" {
		t.Errorf("Description = %q, want %q", tool.Description, "fetch weather")
	}
}

func TestNormalizeTools_RejectsInvalidInputSchema(t *testing.T) {
	raw := []MCPRawTool{{
		Name:        "bad_tool",
		Description: "not an object schema",
		InputSchema: json.RawMessage(`true`),
	}}

	got := NormalizeTools("srv1", raw)

	if len(got.Tools) != 0 {
		t.Fatalf("Tools should be empty; got %+v", got.Tools)
	}
	if len(got.Rejected) != 1 {
		t.Fatalf("Rejected len = %d, want 1; %+v", len(got.Rejected), got.Rejected)
	}
	rej := got.Rejected[0]
	if rej.ServerName != "srv1" {
		t.Errorf("ServerName = %q, want %q", rej.ServerName, "srv1")
	}
	if rej.ToolName != "bad_tool" {
		t.Errorf("ToolName = %q, want %q", rej.ToolName, "bad_tool")
	}
	if rej.Reason != "input_schema_must_be_object" {
		t.Errorf("Reason = %q, want %q", rej.Reason, "input_schema_must_be_object")
	}
}

func TestRenderToolCallResult_StructuredContent(t *testing.T) {
	parts := []StructuredContent{
		{Kind: "text", Text: "hello world"},
		{Kind: "image", MimeType: "image/png"},
		{Kind: "resource", URI: "file:///tmp/foo.txt"},
	}

	got := RenderToolCallResult(parts)

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

func TestRenderToolCallResult_UnknownContentKindFallsBackToText(t *testing.T) {
	parts := []StructuredContent{
		{Kind: "unknown_xyz", Text: "fallback text"},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RenderToolCallResult panicked: %v", r)
		}
	}()
	got := RenderToolCallResult(parts)

	if !strings.Contains(got, "fallback text") {
		t.Errorf("expected fallback text in %q", got)
	}
	// no leak of a raw protocol envelope (e.g. JSON object syntax)
	if strings.Contains(got, "{") || strings.Contains(got, "}") {
		t.Errorf("rendered output leaks protocol envelope: %q", got)
	}
}

func TestBoundedStderrSink_TruncatesAtTailBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stderr.log")

	const tail = 8 * 1024
	const total = 32 * 1024
	sink := NewBoundedStderrSink(path, tail)

	payload := bytes.Repeat([]byte("x"), total)
	n, err := sink.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Errorf("Write returned %d, want %d", n, len(payload))
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	const dropped = total - tail
	wantPrefix := "[truncated 24576 bytes]"
	if !bytes.HasPrefix(contents, []byte(wantPrefix)) {
		head := contents
		if len(head) > 64 {
			head = head[:64]
		}
		t.Errorf("missing truncation marker prefix; first bytes = %q", head)
	}
	if !bytes.HasSuffix(contents, bytes.Repeat([]byte("x"), tail)) {
		t.Errorf("file does not end with last %d 'x' bytes", tail)
	}
	if bytes.Count(contents, []byte("x")) != tail {
		t.Errorf("preserved 'x' count = %d, want %d (dropped=%d)",
			bytes.Count(contents, []byte("x")), tail, dropped)
	}
}

func TestBoundedStderrSink_DiscardModeNoFileWrite(t *testing.T) {
	dir := t.TempDir()
	sink := NewBoundedStderrSink("", 8*1024)

	payload := []byte("some stderr output")
	n, err := sink.Write(payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(payload) {
		t.Errorf("Write returned %d, want %d", n, len(payload))
	}
	if err := sink.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("discard sink wrote files: %v", names)
	}
}
