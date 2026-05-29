package skills

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSkillEffectivenessLedgerAppendsReloadsAndReportsInvalidLines(t *testing.T) {
	root := t.TempDir()
	path := SkillEffectivenessLedgerPath(root)
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	ledger := NewSkillEffectivenessLedger(path, func() time.Time { return now })

	for _, event := range []SkillEffectivenessEvent{
		{
			SkillName:        "planning",
			SessionID:        "sess-1",
			TurnID:           "turn-1",
			Prompt:           "please plan this with api key sk-live-secret",
			SelectionSource:  "lexical",
			LexicalScore:     42,
			TotalScore:       0.42,
			Outcome:          SkillOutcomePositive,
			OperatorFeedback: "great recovery, token ghp_secret must stay hidden",
			FeedbackReason:   "operator_explicit",
		},
		{
			SkillName:       "debugging",
			SessionID:       "sess-1",
			TurnID:          "turn-2",
			Prompt:          "debug failing test",
			SelectionSource: "hybrid",
			LexicalScore:    20,
			SemanticScore:   0.75,
			TotalScore:      1.15,
			Outcome:         SkillOutcomeNeutral,
		},
	} {
		if err := ledger.Record(context.Background(), event); err != nil {
			t.Fatalf("Record(%s): %v", event.SkillName, err)
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile(%q): %v", path, err)
	}
	if _, err := f.WriteString("{not-json}\n"); err != nil {
		f.Close()
		t.Fatalf("append corrupt line: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close corrupt append: %v", err)
	}

	loaded, err := ledger.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if got, want := len(loaded.Records), 2; got != want {
		t.Fatalf("loaded records = %d, want %d", got, want)
	}
	if got, want := len(loaded.Invalid), 1; got != want {
		t.Fatalf("invalid records = %d, want %d", got, want)
	}
	if loaded.Invalid[0].Line != 3 || loaded.Invalid[0].Error == "" {
		t.Fatalf("invalid evidence = %+v, want line 3 with error", loaded.Invalid[0])
	}
	first := loaded.Records[0]
	if first.SkillName != "planning" || first.SessionID != "sess-1" || first.TurnID != "turn-1" {
		t.Fatalf("first record identity = %+v", first)
	}
	if first.PromptSHA256 == "" || first.PromptBytes == 0 || first.RedactedInputCount == 0 {
		t.Fatalf("first redacted prompt evidence = %+v", first)
	}
	if first.OperatorFeedbackCount != 1 || first.FeedbackReason != "operator_explicit" {
		t.Fatalf("first feedback evidence = %+v", first)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	serialized := string(raw)
	for _, forbidden := range []string{"sk-live-secret", "ghp_secret", "please plan this", "great recovery"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("effectiveness ledger leaked raw prompt or feedback %q:\n%s", forbidden, serialized)
		}
	}
}

func TestSkillEffectivenessScoresFeedbackDecayAndStableTies(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	records := []SkillEffectivenessRecord{
		{SkillName: "fresh", Outcome: SkillOutcomePositive, OperatorFeedbackCount: 1, RecordedAt: now.Add(-time.Hour)},
		{SkillName: "bad", Outcome: SkillOutcomeNegative, OperatorFeedbackCount: 1, RecordedAt: now.Add(-time.Hour)},
		{SkillName: "stale", Outcome: SkillOutcomePositive, RecordedAt: now.Add(-15 * 24 * time.Hour)},
		{SkillName: "tie-b", Outcome: SkillOutcomeNeutral, RecordedAt: now.Add(-time.Hour)},
		{SkillName: "tie-a", Outcome: SkillOutcomeNeutral, RecordedAt: now.Add(-time.Hour)},
	}

	scores := ScoreSkillEffectiveness(records, SkillEffectivenessScoreOptions{Now: now, StaleAfter: 7 * 24 * time.Hour})
	byName := map[string]SkillEffectivenessScore{}
	for _, score := range scores {
		byName[score.SkillName] = score
	}
	if byName["fresh"].Score <= byName["stale"].Score {
		t.Fatalf("fresh score %.2f must exceed stale score %.2f", byName["fresh"].Score, byName["stale"].Score)
	}
	if byName["bad"].Score >= 0 {
		t.Fatalf("bad score = %.2f, want negative", byName["bad"].Score)
	}
	if !containsString(byName["fresh"].ReasonCodes, SkillEffectivenessReasonOperatorFeedback) {
		t.Fatalf("fresh reasons = %v, want operator feedback", byName["fresh"].ReasonCodes)
	}
	if !containsString(byName["stale"].ReasonCodes, SkillEffectivenessReasonStaleDecay) {
		t.Fatalf("stale reasons = %v, want stale decay", byName["stale"].ReasonCodes)
	}
	if indexScore(scores, "tie-a") > indexScore(scores, "tie-b") {
		t.Fatalf("tie scores sorted unstably by name: %+v", scores)
	}
}

func TestSkillEffectivenessLedgerDoesNotMutateUsageCounters(t *testing.T) {
	root := t.TempDir()
	if err := MarkAgentCreated(root, "planner"); err != nil {
		t.Fatalf("MarkAgentCreated: %v", err)
	}
	if err := BumpUse(root, "planner"); err != nil {
		t.Fatalf("BumpUse: %v", err)
	}
	before, err := GetUsageRecord(root, "planner")
	if err != nil {
		t.Fatalf("GetUsageRecord before: %v", err)
	}

	ledger := NewSkillEffectivenessLedger(filepath.Join(root, "skills.effectiveness.jsonl"), func() time.Time {
		return time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	})
	if err := ledger.Record(context.Background(), SkillEffectivenessEvent{
		SkillName:       "planner",
		SessionID:       "sess-1",
		TurnID:          "turn-1",
		Prompt:          "plan this safely",
		SelectionSource: "lexical",
		LexicalScore:    10,
		TotalScore:      0.10,
		Outcome:         SkillOutcomePositive,
	}); err != nil {
		t.Fatalf("Record(): %v", err)
	}
	after, err := GetUsageRecord(root, "planner")
	if err != nil {
		t.Fatalf("GetUsageRecord after: %v", err)
	}
	if before.UseCount != after.UseCount || !before.LastUsedAt.Equal(after.LastUsedAt) || before.PatchCount != after.PatchCount || before.ViewCount != after.ViewCount {
		t.Fatalf("usage counters changed after effectiveness record: before=%+v after=%+v", before, after)
	}
}

func indexScore(scores []SkillEffectivenessScore, name string) int {
	for i, score := range scores {
		if score.SkillName == name {
			return i
		}
	}
	return len(scores)
}
