package kernel

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Provenance is the in-memory audit trail for one turn. In Phase 1 it is
// log-backed (slog) and never persisted — Python's state.db is the canonical
// record. In Phase 3, a subset of these fields is promoted to the runs
// table schema from the deterministic-kernel spec §7.2.
type Provenance struct {
	LocalRunID      string
	ServerSessionID string
	Endpoint        string
	StartedAt       time.Time
	FinishReason    string
	TokensIn        int
	TokensOut       int
	LatencyMs       int
	ErrorClass      string
	ErrorText       string
}

func newProvenance(endpoint string) Provenance {
	return Provenance{
		LocalRunID: uuid.NewString(),
		Endpoint:   endpoint,
		StartedAt:  time.Now(),
	}
}

func (p Provenance) LogAdmitted(log *slog.Logger) {
	log.Info("turn admitted", "local_run_id", p.LocalRunID)
}

func (p Provenance) LogPOSTSent(log *slog.Logger) {
	log.Info("turn POST sent", "local_run_id", p.LocalRunID, "endpoint", p.Endpoint)
}

func (p Provenance) LogSSEStart(log *slog.Logger) {
	log.Info("turn SSE start", "local_run_id", p.LocalRunID, "server_session_id", p.ServerSessionID)
}

func (p Provenance) LogDone(log *slog.Logger) {
	log.Info("turn done",
		"local_run_id", p.LocalRunID,
		"server_session_id", p.ServerSessionID,
		"finish", p.FinishReason,
		"tokens_in", p.TokensIn,
		"tokens_out", p.TokensOut,
		"latency_ms", p.LatencyMs)
}

func (p Provenance) LogError(log *slog.Logger) {
	log.Info("turn error",
		"local_run_id", p.LocalRunID,
		"class", p.ErrorClass,
		"err", p.ErrorText)
}
