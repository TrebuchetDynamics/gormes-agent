package mcp

import (
	"encoding/json"
	"testing"
)

func TestNormalizeTools_SanitizesUnsafeNames(t *testing.T) {
	raw := []RawTool{{
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

func TestNormalizeTools_RejectsDuplicateSanitizedNames(t *testing.T) {
	raw := []RawTool{{
		Name:        "web/search",
		Description: "first spelling",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, {
		Name:        "web search",
		Description: "colliding spelling",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}

	got := NormalizeTools("srv1", raw)

	if len(got.Tools) != 1 {
		t.Fatalf("Tools len = %d, want 1; tools=%+v rejected=%+v", len(got.Tools), got.Tools, got.Rejected)
	}
	if got.Tools[0].Name != "web_search" || got.Tools[0].SourceRaw.Name != "web/search" {
		t.Fatalf("kept tool = %+v, want first colliding candidate preserved", got.Tools[0])
	}
	if len(got.Rejected) != 1 {
		t.Fatalf("Rejected len = %d, want 1; %+v", len(got.Rejected), got.Rejected)
	}
	rej := got.Rejected[0]
	if rej.ToolName != "web search" || rej.Reason != SchemaRejectionReasonDuplicateSanitizedName {
		t.Fatalf("rejection = %+v, want duplicate sanitized name for second candidate", rej)
	}
}

func TestNormalizeTools_RejectsInvalidInputSchema(t *testing.T) {
	raw := []RawTool{{
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
