package tools

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/checkpoint"
)

const checkpointWorkdirMarker = "GORMES_WORKDIR"

// CheckpointManagerOptions configures the startup shadow-repo GC contract.
type CheckpointManagerOptions = checkpoint.CheckpointManagerOptions

// CheckpointManager owns the read model for Gormes checkpoint rollback state.
type CheckpointManager = checkpoint.CheckpointManager

// CheckpointStatus reports degraded-mode evidence from checkpoint startup GC.
type CheckpointStatus = checkpoint.CheckpointStatus

// CheckpointEvidence names a cleanup condition and the affected shadow repos.
type CheckpointEvidence = checkpoint.CheckpointEvidence

// NewCheckpointManager performs deterministic startup cleanup before callers
// can depend on rollback state.
func NewCheckpointManager(opts CheckpointManagerOptions) (*CheckpointManager, error) {
	return checkpoint.NewCheckpointManager(opts)
}

// DefaultCheckpointRoot returns Gormes' XDG-owned rollback directory.
func DefaultCheckpointRoot() string {
	return checkpoint.DefaultCheckpointRoot()
}

// StoreStatusResult is the read model returned by StoreStatus, mirroring
// Hermes' store_status() dict shape.
type StoreStatusResult = checkpoint.StoreStatusResult

// StoreStatusProject is one checkpoint shadow-repo entry in the status table.
type StoreStatusProject = checkpoint.StoreStatusProject

// StoreStatusArchive is a legacy archive visible to clear-legacy.
type StoreStatusArchive = checkpoint.StoreStatusArchive

// StoreStatus builds a read-only snapshot of the checkpoint store under root.
// It does not mutate any files. A non-existent root returns an empty result
// with no error.
func StoreStatus(root string) (StoreStatusResult, error) {
	return checkpoint.StoreStatus(root)
}

// PruneResult is the summary returned by PruneCheckpoints.
type PruneResult = checkpoint.PruneResult

// PruneCheckpoints deletes orphan (workdir missing) and stale (last touch older
// than retentionDays) shadow repos, then optionally enforces maxSizeMB.
func PruneCheckpoints(root string, retentionDays int, keepOrphans bool, maxSizeMB int, now func() time.Time) PruneResult {
	return checkpoint.PruneCheckpoints(root, retentionDays, keepOrphans, maxSizeMB, now)
}

// PruneCheckpointsDryRun mirrors PruneCheckpoints but skips the RemoveAll calls
// and marks the result as dry-run output.
func PruneCheckpointsDryRun(root string, retentionDays int, keepOrphans bool, maxSizeMB int, dryRun bool, now func() time.Time) PruneResult {
	return checkpoint.PruneCheckpointsDryRun(root, retentionDays, keepOrphans, maxSizeMB, dryRun, now)
}

// ClearResult reports what clear deleted.
type ClearResult = checkpoint.ClearResult

// ClearAll deletes the entire checkpoint root directory.
func ClearAll(root string) ClearResult {
	return checkpoint.ClearAll(root)
}

// ClearLegacy deletes only legacy-* archive directories under root.
func ClearLegacy(root string) ClearResult {
	return checkpoint.ClearLegacy(root)
}
