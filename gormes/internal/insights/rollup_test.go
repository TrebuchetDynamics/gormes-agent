package insights

import (
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

func TestDailyRollupForDate_AggregatesRuntimeUsageByDay(t *testing.T) {
	day := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)

	got := DailyRollupForDate(day, []SessionUsage{
		{
			SessionID:        "sess-1",
			Model:            "gpt-4",
			TokensIn:         100,
			TokensOut:        40,
			EstimatedCostUSD: 0.0125,
			FinishedAt:       time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
		},
		{
			SessionID:        "sess-1",
			Model:            "gpt-4",
			TokensIn:         20,
			TokensOut:        5,
			EstimatedCostUSD: 0.0015,
			FinishedAt:       time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
		},
		{
			SessionID:        "sess-2",
			Model:            "claude-opus",
			TokensIn:         70,
			TokensOut:        30,
			EstimatedCostUSD: 0.02,
			FinishedAt:       time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
		},
		{
			SessionID:        "sess-3",
			Model:            "gpt-4",
			TokensIn:         999,
			TokensOut:        999,
			EstimatedCostUSD: 9.99,
			FinishedAt:       time.Date(2026, 4, 23, 0, 1, 0, 0, time.UTC),
		},
	})

	if got.Date != "2026-04-22" {
		t.Fatalf("Date = %q, want %q", got.Date, "2026-04-22")
	}
	if got.SessionCount != 2 {
		t.Fatalf("SessionCount = %d, want 2", got.SessionCount)
	}
	if got.TotalTokensIn != 190 {
		t.Fatalf("TotalTokensIn = %d, want 190", got.TotalTokensIn)
	}
	if got.TotalTokensOut != 75 {
		t.Fatalf("TotalTokensOut = %d, want 75", got.TotalTokensOut)
	}
	if got.EstimatedCostUSD != 0.034 {
		t.Fatalf("EstimatedCostUSD = %v, want 0.034", got.EstimatedCostUSD)
	}
	wantBreakdown := map[string]int{
		"claude-opus": 1,
		"gpt-4":       1,
	}
	if len(got.ModelBreakdown) != len(wantBreakdown) {
		t.Fatalf("ModelBreakdown len = %d, want %d", len(got.ModelBreakdown), len(wantBreakdown))
	}
	for model, want := range wantBreakdown {
		if got.ModelBreakdown[model] != want {
			t.Fatalf("ModelBreakdown[%q] = %d, want %d", model, got.ModelBreakdown[model], want)
		}
	}
}

func TestDailyRollupForDate_UsesLatestModelPerSessionInBreakdown(t *testing.T) {
	day := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)

	got := DailyRollupForDate(day, []SessionUsage{
		{
			SessionID:        "sess-1",
			Model:            "gpt-4",
			TokensIn:         10,
			TokensOut:        5,
			EstimatedCostUSD: 0.002,
			FinishedAt:       time.Date(2026, 4, 22, 8, 0, 0, 0, time.UTC),
		},
		{
			SessionID:        "sess-1",
			Model:            "claude-opus",
			TokensIn:         15,
			TokensOut:        6,
			EstimatedCostUSD: 0.003,
			FinishedAt:       time.Date(2026, 4, 22, 9, 0, 0, 0, time.UTC),
		},
		{
			SessionID:        "sess-2",
			Model:            "",
			TokensIn:         5,
			TokensOut:        1,
			EstimatedCostUSD: 0,
			FinishedAt:       time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC),
		},
	})

	if got.SessionCount != 2 {
		t.Fatalf("SessionCount = %d, want 2", got.SessionCount)
	}
	if len(got.ModelBreakdown) != 1 {
		t.Fatalf("ModelBreakdown len = %d, want 1", len(got.ModelBreakdown))
	}
	if got.ModelBreakdown["claude-opus"] != 1 {
		t.Fatalf("ModelBreakdown[claude-opus] = %d, want 1", got.ModelBreakdown["claude-opus"])
	}
	if got.ModelBreakdown["gpt-4"] != 0 {
		t.Fatalf("ModelBreakdown[gpt-4] = %d, want 0", got.ModelBreakdown["gpt-4"])
	}
}

func TestSessionUsageFromTelemetryCopiesRuntimeCounters(t *testing.T) {
	finishedAt := time.Date(2026, 4, 22, 11, 30, 0, 0, time.UTC)
	snap := telemetry.Snapshot{
		Model:          "claude-opus",
		TokensInTotal:  120,
		TokensOutTotal: 45,
	}

	got := SessionUsageFromTelemetry("sess-9", finishedAt, snap, 0.018)

	if got.SessionID != "sess-9" {
		t.Fatalf("SessionID = %q, want %q", got.SessionID, "sess-9")
	}
	if got.Model != "claude-opus" {
		t.Fatalf("Model = %q, want %q", got.Model, "claude-opus")
	}
	if got.TokensIn != 120 {
		t.Fatalf("TokensIn = %d, want 120", got.TokensIn)
	}
	if got.TokensOut != 45 {
		t.Fatalf("TokensOut = %d, want 45", got.TokensOut)
	}
	if got.EstimatedCostUSD != 0.018 {
		t.Fatalf("EstimatedCostUSD = %v, want 0.018", got.EstimatedCostUSD)
	}
	if !got.FinishedAt.Equal(finishedAt) {
		t.Fatalf("FinishedAt = %v, want %v", got.FinishedAt, finishedAt)
	}
}
