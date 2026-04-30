package insights

import (
	"context"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

type SessionUsageSink interface {
	RecordSessionUsage(context.Context, SessionUsage) error
}

type SelfMonitoringUsageRecorder struct {
	Sink SessionUsageSink
}

func (r SelfMonitoringUsageRecorder) RecordSelfMonitoringUsage(ctx context.Context, usage telemetry.UsageEvidence) error {
	if r.Sink == nil {
		return nil
	}
	return r.Sink.RecordSessionUsage(ctx, SessionUsageFromSelfMonitoring(usage))
}

func SessionUsageFromSelfMonitoring(usage telemetry.UsageEvidence) SessionUsage {
	return SessionUsage{
		SessionID:        strings.TrimSpace(usage.SessionID),
		Model:            strings.TrimSpace(usage.Model),
		TokensIn:         clampPositive(usage.InputTokens) + clampPositive(usage.CacheReadTokens) + clampPositive(usage.CacheWriteTokens),
		TokensOut:        clampPositive(usage.OutputTokens) + clampPositive(usage.ReasoningTokens),
		EstimatedCostUSD: usage.EstimatedCostUSD,
		FinishedAt:       usage.FinishedAt,
	}
}

func clampPositive(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
