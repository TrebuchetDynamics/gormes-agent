package hermes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDisabledContextEngine_StatusToolFixture(t *testing.T) {
	engine := NewDisabledContextEngine("compression disabled by config")
	engine.UpdateModelContext(ContextModelContext{
		Model:            "fixture-model",
		ContextLength:    8000,
		ThresholdPercent: 0.75,
	})
	engine.UpdateFromResponse(ContextUsage{
		PromptTokens:     5800,
		CompletionTokens: 120,
		TotalTokens:      5920,
	})
	engine.SetCompressionCooldown(90, "summary provider unavailable")
	engine.RecordReplayGap(ContextReplayGap{
		Kind:    "missing_fixture",
		Message: "no compression replay fixture for fixture-model",
	})

	unknownPayload, err := engine.HandleToolCall(context.Background(), "missing_context_tool", json.RawMessage(`{"query":"x"}`), ContextToolCallOptions{})
	if !errors.Is(err, ErrUnknownContextTool) {
		t.Fatalf("unknown tool err = %v, want ErrUnknownContextTool", err)
	}
	assertJSONEqual(t, unknownPayload, []byte(`{
		"error": {
			"type": "unknown_context_tool",
			"tool": "missing_context_tool",
			"message": "Unknown context engine tool: missing_context_tool"
		}
	}`))

	statusPayload, err := engine.HandleToolCall(context.Background(), ContextStatusToolName, json.RawMessage(`{}`), ContextToolCallOptions{})
	if err != nil {
		t.Fatalf("context status tool returned error: %v", err)
	}

	want, err := os.ReadFile(filepath.Join("testdata", "context_status", "disabled_pressure_unknown_tool.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, statusPayload, want)
}

func TestDisabledContextEngine_UpdateModelContextRecalculatesThreshold(t *testing.T) {
	engine := NewDisabledContextEngine("disabled")

	engine.UpdateModelContext(ContextModelContext{
		Model:            "small",
		ContextLength:    4096,
		ThresholdPercent: 0.5,
	})
	status := engine.Status()
	if status.Model != "small" || status.ContextLength != 4096 || status.ThresholdTokens != 2048 {
		t.Fatalf("status after first update = %#v, want model small context 4096 threshold 2048", status)
	}

	engine.UpdateModelContext(ContextModelContext{
		Model:         "larger",
		ContextLength: 10000,
	})
	status = engine.Status()
	if status.Model != "larger" || status.ContextLength != 10000 || status.ThresholdPercent != 0.5 || status.ThresholdTokens != 5000 {
		t.Fatalf("status after preserving threshold percent = %#v, want larger context with 50%% threshold", status)
	}
}

func TestDisabledContextEngine_CompressIsExplicitDisabledBoundary(t *testing.T) {
	engine := NewDisabledContextEngine("compression disabled by config")
	messages := []Message{{Role: "user", Content: "hello"}}

	got, report, err := engine.Compress(context.Background(), messages, CompressionRequest{CurrentTokens: 9000})
	if !errors.Is(err, ErrCompressionDisabled) {
		t.Fatalf("Compress err = %v, want ErrCompressionDisabled", err)
	}
	if !reflect.DeepEqual(got, messages) {
		t.Fatalf("Compress messages = %#v, want original messages unchanged", got)
	}
	if report.State != "disabled" || report.BeforeMessages != 1 || report.AfterMessages != 1 {
		t.Fatalf("Compression report = %#v, want disabled no-op boundary report", report)
	}
	if engine.Status().CompressionCount != 0 {
		t.Fatalf("compression_count = %d, want 0 for disabled no-op", engine.Status().CompressionCount)
	}
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	var gotAny any
	if err := json.Unmarshal(got, &gotAny); err != nil {
		t.Fatalf("decode got JSON: %v\n%s", err, got)
	}
	var wantAny any
	if err := json.Unmarshal(want, &wantAny); err != nil {
		t.Fatalf("decode want JSON: %v\n%s", err, want)
	}
	if !reflect.DeepEqual(gotAny, wantAny) {
		gotPretty, _ := json.MarshalIndent(gotAny, "", "  ")
		wantPretty, _ := json.MarshalIndent(wantAny, "", "  ")
		t.Fatalf("JSON mismatch\n got: %s\nwant: %s", gotPretty, wantPretty)
	}
}
