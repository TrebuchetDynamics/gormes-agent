package tools

import (
	"strings"
	"testing"
)

func TestBlockerRecordFormatMatchesFleetStandard(t *testing.T) {
	record := BlockerRecord{
		Title:        "Telegram deploy",
		Type:         BlockerTypeInfra,
		RecordedAt:   "2026-05-01T12:00:00-06:00",
		Blocker:      "gateway binary is locked",
		Evidence:     "sessions.db: database is locked",
		UnblocksWhen: "gateway process exits",
		Owner:        "operator",
		Pivot:        "continue provider parity tests",
		NextCheck:    "2026-05-01T12:30:00-06:00",
	}

	got := FormatBlockerRecord(record)
	for _, want := range []string{
		"[BLOCKED] Telegram deploy \u2014 2026-05-01T12:00:00-06:00",
		"  type: infra",
		"  blocker: gateway binary is locked",
		"  evidence: sessions.db: database is locked",
		"  unblocks when: gateway process exits",
		"  owner: operator",
		"  workaround/pivot: continue provider parity tests",
		"  next check: 2026-05-01T12:30:00-06:00",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatted blocker missing %q:\n%s", want, got)
		}
	}
}

func TestNormalizeBlockerRecordClassifiesUnknownAndMissingFields(t *testing.T) {
	got := NormalizeBlockerRecord(BlockerRecord{
		Title:        "Provider outage",
		Type:         "vendor",
		Blocker:      "upstream 503",
		UnblocksWhen: "provider recovers",
		Owner:        "provider",
		NextCheck:    "2026-05-01T13:00:00-06:00",
	})

	if got.Type != BlockerTypeUnknown {
		t.Fatalf("Type = %q, want %q", got.Type, BlockerTypeUnknown)
	}
	if got.Status != BlockerStatusUnclassified {
		t.Fatalf("Status = %q, want %q", got.Status, BlockerStatusUnclassified)
	}
	if !got.Degraded {
		t.Fatal("Degraded = false, want true")
	}
	if !containsString(got.MissingFields, "evidence") || !containsString(got.MissingFields, "pivot") {
		t.Fatalf("MissingFields = %v, want evidence and pivot", got.MissingFields)
	}
}

func TestSelectBlockerPivotChoosesFirstActionablePivot(t *testing.T) {
	got := SelectBlockerPivot([]BlockerRecord{
		{Title: "no pivot", Pivot: ""},
		{Title: "has pivot", Pivot: "run next unblocked P0 row"},
	})

	if got != "run next unblocked P0 row" {
		t.Fatalf("pivot = %q, want actionable pivot", got)
	}
}
