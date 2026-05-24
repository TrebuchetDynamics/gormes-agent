package loopcost

import (
	"strings"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
}

func TestParseRunCost_ValidOpenCodeJSONL(t *testing.T) {
	jsonl := `{"usage":{"cost":0.0123,"input_tokens":1500,"output_tokens":200},"timestamp":"2026-05-07T11:00:00Z"}`
	rc, err := ParseRunCost(strings.NewReader(jsonl), "run-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.RunID != "run-001" {
		t.Errorf("RunID = %q, want %q", rc.RunID, "run-001")
	}
	if rc.CostUSD != 0.0123 {
		t.Errorf("CostUSD = %f, want 0.0123", rc.CostUSD)
	}
	if rc.InputTokens != 1500 {
		t.Errorf("InputTokens = %d, want 1500", rc.InputTokens)
	}
	if rc.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", rc.OutputTokens)
	}
	if rc.Backend == "" {
		t.Error("Backend should not be empty")
	}
}

func TestParseRunCost_NoCostField(t *testing.T) {
	jsonl := `{"usage":{"input_tokens":100},"timestamp":"2026-05-07T11:00:00Z"}`
	_, err := ParseRunCost(strings.NewReader(jsonl), "run-002")
	if err == nil {
		t.Fatal("expected error for missing cost")
	}
	if !strings.Contains(err.Error(), "cost") {
		t.Errorf("error should mention cost: %v", err)
	}
}

func TestParseRunCost_InvalidJSON(t *testing.T) {
	jsonl := `not json`
	_, err := ParseRunCost(strings.NewReader(jsonl), "run-003")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseRunCost_EmptyInput(t *testing.T) {
	_, err := ParseRunCost(strings.NewReader(""), "run-empty")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParseRunCost_CostAsString(t *testing.T) {
	jsonl := `{"usage":{"cost":"0.0456","input_tokens":3000,"output_tokens":500},"timestamp":"2026-05-07T10:00:00Z"}`
	rc, err := ParseRunCost(strings.NewReader(jsonl), "run-str")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.CostUSD != 0.0456 {
		t.Errorf("CostUSD = %f, want 0.0456", rc.CostUSD)
	}
}

func TestParseRunCost_CurrentOpenCodePartCostSchema(t *testing.T) {
	jsonl := `{"type":"step_start","timestamp":1778160166743,"sessionID":"ses-test"}
{"type":"step_finish","timestamp":1778160185296,"sessionID":"ses-test","part":{"type":"step-finish","cost":0.08239596,"tokens":{"input":46206,"output":126,"reasoning":448,"cache":{"read":0,"write":0}}}}
{"type":"message","timestamp":1778160186000,"sessionID":"ses-test","part":{"type":"text","text":"done"}}`
	rc, err := ParseRunCost(strings.NewReader(jsonl), "run-part")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rc.CostUSD != 0.08239596 {
		t.Errorf("CostUSD = %f, want 0.08239596", rc.CostUSD)
	}
	if rc.InputTokens != 46206 {
		t.Errorf("InputTokens = %d, want 46206", rc.InputTokens)
	}
	if rc.OutputTokens != 126 {
		t.Errorf("OutputTokens = %d, want 126", rc.OutputTokens)
	}
	if got, want := rc.Timestamp, time.UnixMilli(1778160185296).UTC(); !got.Equal(want) {
		t.Errorf("Timestamp = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestDailyRollup(t *testing.T) {
	now := fixedNow()
	costs := []RunCost{
		{RunID: "r1", Timestamp: now.Add(-1 * time.Hour), CostUSD: 0.01, InputTokens: 100, OutputTokens: 50},
		{RunID: "r2", Timestamp: now.Add(-12 * time.Hour), CostUSD: 0.02, InputTokens: 200, OutputTokens: 100},
		{RunID: "r3", Timestamp: now.Add(-25 * time.Hour), CostUSD: 0.03, InputTokens: 300, OutputTokens: 150},
	}
	r := DailyRollup(costs, now)
	if r.Summary.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2 (r3 is outside 24h window)", r.Summary.RunCount)
	}
	if r.Summary.TotalCost != 0.03 {
		t.Errorf("TotalCost = %f, want 0.03", r.Summary.TotalCost)
	}
	if r.Summary.TotalInputTokens != 300 {
		t.Errorf("TotalInputTokens = %d, want 300", r.Summary.TotalInputTokens)
	}
	if r.Period != "daily" {
		t.Errorf("Period = %q, want %q", r.Period, "daily")
	}
}

func TestWindowRollup_7Day(t *testing.T) {
	now := fixedNow()
	costs := []RunCost{
		{RunID: "r1", Timestamp: now.Add(-1 * 24 * time.Hour), CostUSD: 0.01},
		{RunID: "r2", Timestamp: now.Add(-3 * 24 * time.Hour), CostUSD: 0.02},
		{RunID: "r3", Timestamp: now.Add(-8 * 24 * time.Hour), CostUSD: 0.03},
	}
	r := WindowRollup(costs, 7*24*time.Hour, now)
	if r.Summary.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2 (r3 is outside 7-day window)", r.Summary.RunCount)
	}
	if r.Summary.TotalCost != 0.03 {
		t.Errorf("TotalCost = %f, want 0.03", r.Summary.TotalCost)
	}
}

func TestWindowRollup_Empty(t *testing.T) {
	now := fixedNow()
	r := WindowRollup(nil, 24*time.Hour, now)
	if r.Summary.RunCount != 0 {
		t.Errorf("RunCount = %d, want 0", r.Summary.RunCount)
	}
	if r.Summary.TotalCost != 0 {
		t.Errorf("TotalCost = %f, want 0", r.Summary.TotalCost)
	}
}

func TestParseJSONLCosts_MultiLine(t *testing.T) {
	jsonl := `{"usage":{"cost":0.01,"input_tokens":100,"output_tokens":50},"timestamp":"2026-05-07T10:00:00Z"}
{"usage":{"cost":0.02,"input_tokens":200,"output_tokens":100},"timestamp":"2026-05-07T10:30:00Z"}
{"usage":{"cost":0.03,"input_tokens":300,"output_tokens":150},"timestamp":"2026-05-07T11:00:00Z"}`
	costs, err := ParseJSONLCosts(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(costs) != 3 {
		t.Fatalf("len(costs) = %d, want 3", len(costs))
	}
	if costs[0].CostUSD != 0.01 {
		t.Errorf("costs[0].CostUSD = %f, want 0.01", costs[0].CostUSD)
	}
	if costs[2].CostUSD != 0.03 {
		t.Errorf("costs[2].CostUSD = %f, want 0.03", costs[2].CostUSD)
	}
}

func TestParseJSONLCosts_EmptyLines(t *testing.T) {
	jsonl := `{"usage":{"cost":0.01,"input_tokens":100},"timestamp":"2026-05-07T10:00:00Z"}

{"usage":{"cost":0.02,"input_tokens":200},"timestamp":"2026-05-07T10:30:00Z"}`
	costs, err := ParseJSONLCosts(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(costs) != 2 {
		t.Fatalf("len(costs) = %d, want 2", len(costs))
	}
}

func TestRunCost_UnknownCostSentinel(t *testing.T) {
	rc := RunCost{CostUSD: UnknownCost}
	if !rc.IsUnknownCost() {
		t.Error("IsUnknownCost should return true for UnknownCost sentinel")
	}
	rc2 := RunCost{CostUSD: 0.01}
	if rc2.IsUnknownCost() {
		t.Error("IsUnknownCost should return false for valid cost")
	}
}
