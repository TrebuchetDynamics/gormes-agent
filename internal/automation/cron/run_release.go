package cron

import "github.com/TrebuchetDynamics/gormes-agent/internal/automation/cron/release"

// ReleaseEvidenceCode classifies one resource-release outcome inside a cron
// run's release ledger. Kept as a root alias for cron package compatibility.
type ReleaseEvidenceCode = release.ReleaseEvidenceCode

const (
	ReleaseEvidenceSubprocessKilled     = release.ReleaseEvidenceSubprocessKilled
	ReleaseEvidenceSessionDBClosed      = release.ReleaseEvidenceSessionDBClosed
	ReleaseEvidenceHTTPIdleClosed       = release.ReleaseEvidenceHTTPIdleClosed
	ReleaseEvidenceHTTPIdleClosedFailed = release.ReleaseEvidenceHTTPIdleClosedFailed
	ReleaseEvidenceSkippedNoResource    = release.ReleaseEvidenceSkippedNoResource
)

// ReleaseEvidence is one entry in the per-run release log.
type ReleaseEvidence = release.ReleaseEvidence

// SubprocessKiller is the narrow seam used to terminate spawned tool subprocesses.
type SubprocessKiller = release.SubprocessKiller

// RunReleaseLedger records the per-run resources acquired by a cron run.
type RunReleaseLedger = release.RunReleaseLedger

// NewRunReleaseLedger constructs an empty release ledger.
func NewRunReleaseLedger() *RunReleaseLedger {
	return release.NewRunReleaseLedger()
}
