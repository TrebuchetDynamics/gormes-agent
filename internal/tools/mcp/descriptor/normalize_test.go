package descriptor

import (
	"encoding/json"
	"testing"
)

func TestNormalizeToolsRejectsEmptySanitizedNames(t *testing.T) {
	raw := []RawTool{{
		Name:        "",
		Description: "empty raw name cannot produce a provider tool name",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}

	got := NormalizeTools("srv1", raw)

	if len(got.Tools) != 0 {
		t.Fatalf("Tools len = %d, want 0; tools=%+v", len(got.Tools), got.Tools)
	}
	if len(got.Rejected) != 1 {
		t.Fatalf("Rejected len = %d, want 1; %+v", len(got.Rejected), got.Rejected)
	}
	rej := got.Rejected[0]
	if rej.ServerName != "srv1" || rej.ToolName != "" || rej.Reason != SchemaRejectionReasonEmptySanitizedName {
		t.Fatalf("rejection = %+v, want empty sanitized name rejection", rej)
	}
}

func TestNormalizeToolsRejectsDuplicateSanitizedNames(t *testing.T) {
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
	if rej.ServerName != "srv1" || rej.ToolName != "web search" || rej.Reason != SchemaRejectionReasonDuplicateSanitizedName {
		t.Fatalf("rejection = %+v, want duplicate sanitized name for second candidate", rej)
	}
}

func TestNormalizeToolsDoesNotReserveRejectedCandidateNames(t *testing.T) {
	raw := []RawTool{{
		Name:        "web/search",
		Description: "invalid first spelling",
		InputSchema: json.RawMessage(`true`),
	}, {
		Name:        "web search",
		Description: "valid colliding spelling",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}}

	got := NormalizeTools("srv1", raw)

	if len(got.Tools) != 1 {
		t.Fatalf("Tools len = %d, want 1; tools=%+v rejected=%+v", len(got.Tools), got.Tools, got.Rejected)
	}
	if got.Tools[0].Name != "web_search" || got.Tools[0].SourceRaw.Name != "web search" {
		t.Fatalf("kept tool = %+v, want later valid candidate to own sanitized name", got.Tools[0])
	}
	if len(got.Rejected) != 1 {
		t.Fatalf("Rejected len = %d, want 1; %+v", len(got.Rejected), got.Rejected)
	}
	if got.Rejected[0].ToolName != "web/search" || got.Rejected[0].Reason != SchemaRejectionReasonInputSchemaNotObject {
		t.Fatalf("rejection = %+v, want invalid schema rejection for first candidate", got.Rejected[0])
	}
}
