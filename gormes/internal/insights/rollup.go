package insights

import (
	"math"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/telemetry"
)

// SessionUsage is the minimal per-session runtime envelope that the lightweight
// Phase 3.E.5 rollups aggregate before the append-only JSONL writer lands.
type SessionUsage struct {
	SessionID        string
	Model            string
	TokensIn         int
	TokensOut        int
	EstimatedCostUSD float64
	FinishedAt       time.Time
}

// DailyRollup is the operator-facing daily aggregate that later writers will
// persist to usage.jsonl.
type DailyRollup struct {
	Date             string         `json:"date"`
	SessionCount     int            `json:"session_count"`
	TotalTokensIn    int            `json:"total_tokens_in"`
	TotalTokensOut   int            `json:"total_tokens_out"`
	EstimatedCostUSD float64        `json:"estimated_cost_usd"`
	ModelBreakdown   map[string]int `json:"model_breakdown"`
}

type modelStamp struct {
	model string
	at    time.Time
}

// DailyRollupForDate folds per-session runtime usage into one UTC-day summary.
// SessionCount is de-duplicated by SessionID, while token and cost totals sum
// every included usage sample. ModelBreakdown attributes each session to the
// latest non-empty model observed for that session on the target day.
func DailyRollupForDate(day time.Time, usage []SessionUsage) DailyRollup {
	start := time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	rollup := DailyRollup{
		Date:           start.Format("2006-01-02"),
		ModelBreakdown: map[string]int{},
	}

	sessions := make(map[string]struct{})
	models := make(map[string]modelStamp)

	for _, sample := range usage {
		finished := sample.FinishedAt.UTC()
		if finished.Before(start) || !finished.Before(end) {
			continue
		}

		rollup.TotalTokensIn += sample.TokensIn
		rollup.TotalTokensOut += sample.TokensOut
		rollup.EstimatedCostUSD += sample.EstimatedCostUSD

		if sample.SessionID != "" {
			sessions[sample.SessionID] = struct{}{}
			if sample.Model != "" {
				cur, ok := models[sample.SessionID]
				if !ok || finished.After(cur.at) {
					models[sample.SessionID] = modelStamp{model: sample.Model, at: finished}
				}
			}
		}
	}

	rollup.SessionCount = len(sessions)
	for _, stamp := range models {
		rollup.ModelBreakdown[stamp.model]++
	}
	rollup.EstimatedCostUSD = roundUSD(rollup.EstimatedCostUSD)
	return rollup
}

func roundUSD(v float64) float64 {
	return math.Round(v*1_000_000) / 1_000_000
}

// SessionUsageFromTelemetry captures the local runtime counters that the
// kernel/TUI already maintain in memory and normalizes them into the rollup
// input shape used by Phase 3.E.5 aggregation.
func SessionUsageFromTelemetry(sessionID string, finishedAt time.Time, snap telemetry.Snapshot, estimatedCostUSD float64) SessionUsage {
	return SessionUsage{
		SessionID:        sessionID,
		Model:            snap.Model,
		TokensIn:         snap.TokensInTotal,
		TokensOut:        snap.TokensOutTotal,
		EstimatedCostUSD: estimatedCostUSD,
		FinishedAt:       finishedAt,
	}
}
