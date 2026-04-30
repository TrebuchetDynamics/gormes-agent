package insights

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestSelfMonitoringUsageFeedsExistingDailyRollupSchema(t *testing.T) {
	finished := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)
	usage := telemetry.UsageEvidence{
		SessionID:        "sess-1",
		Provider:         "openai-codex",
		Model:            "gpt-5.5",
		InputTokens:      100,
		CacheReadTokens:  20,
		CacheWriteTokens: 5,
		OutputTokens:     40,
		ReasoningTokens:  8,
		EstimatedCostUSD: 0.0123456,
		ToolCalls:        3,
		ToolErrors:       1,
		FinishedAt:       finished,
	}

	session := SessionUsageFromSelfMonitoring(usage)
	if session.SessionID != "sess-1" || session.Model != "gpt-5.5" {
		t.Fatalf("session usage identity = %+v, want sess-1/gpt-5.5", session)
	}
	if session.TokensIn != 125 {
		t.Fatalf("TokensIn = %d, want input + cache read + cache write = 125", session.TokensIn)
	}
	if session.TokensOut != 48 {
		t.Fatalf("TokensOut = %d, want output + reasoning tokens = 48", session.TokensOut)
	}

	rollup := DailyRollupForDate(finished, []SessionUsage{session})
	raw, err := json.Marshal(rollup)
	if err != nil {
		t.Fatalf("Marshal rollup: %v", err)
	}
	for _, existingKey := range []string{"date", "session_count", "total_tokens_in", "total_tokens_out", "estimated_cost_usd", "model_breakdown"} {
		if !strings.Contains(string(raw), `"`+existingKey+`"`) {
			t.Fatalf("rollup JSON missing existing key %q: %s", existingKey, raw)
		}
	}
	for _, forbiddenKey := range []string{"provider", "cache_read_tokens", "reasoning_tokens", "tool_calls", "tool_errors"} {
		if strings.Contains(string(raw), forbiddenKey) {
			t.Fatalf("rollup JSON added self-monitoring-only key %q: %s", forbiddenKey, raw)
		}
	}
}

func TestSelfMonitoringUsageRecorderAdaptsToSessionUsageSink(t *testing.T) {
	sink := &recordingSessionUsageSink{}
	recorder := SelfMonitoringUsageRecorder{Sink: sink}
	finished := time.Date(2026, 4, 29, 13, 0, 0, 0, time.UTC)

	if err := recorder.RecordSelfMonitoringUsage(context.Background(), telemetry.UsageEvidence{
		SessionID:    "sess-2",
		Model:        "claude-opus",
		InputTokens:  10,
		OutputTokens: 4,
		FinishedAt:   finished,
	}); err != nil {
		t.Fatalf("RecordSelfMonitoringUsage: %v", err)
	}

	if len(sink.usage) != 1 {
		t.Fatalf("sink usage count = %d, want 1", len(sink.usage))
	}
	if sink.usage[0].SessionID != "sess-2" || sink.usage[0].TokensIn != 10 || sink.usage[0].TokensOut != 4 {
		t.Fatalf("sink usage = %+v, want sess-2 10/4", sink.usage[0])
	}
}

type recordingSessionUsageSink struct {
	usage []SessionUsage
}

func (s *recordingSessionUsageSink) RecordSessionUsage(_ context.Context, usage SessionUsage) error {
	s.usage = append(s.usage, usage)
	return nil
}
