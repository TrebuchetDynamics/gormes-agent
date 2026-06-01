package diagnostics

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics/logs"

// LogClass identifies which Gormes log file a snapshot section belongs to.
type LogClass = logs.LogClass

const (
	LogClassMain      = logs.LogClassMain
	LogClassToolAudit = logs.LogClassToolAudit
)

// LogSnapshotRoots carries the resolved log file paths supplied by the caller.
type LogSnapshotRoots = logs.LogSnapshotRoots

// SnapshotOpts controls how many bytes from the head and tail of each log
// file are kept.
type SnapshotOpts = logs.SnapshotOpts

// LogSection records the redacted head/tail of a single log file plus
// degraded-mode evidence flags.
type LogSection = logs.LogSection

// Snapshot is the ordered collection of LogSections SnapshotLogs returns.
type Snapshot = logs.Snapshot

// SnapshotLogs reads the configured log files, applies head/tail truncation,
// and runs each non-empty line through RedactLine.
func SnapshotLogs(roots LogSnapshotRoots, opts SnapshotOpts) Snapshot {
	return logs.SnapshotLogs(roots, opts)
}
