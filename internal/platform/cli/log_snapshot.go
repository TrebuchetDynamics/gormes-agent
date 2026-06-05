package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics"

// LogClass identifies which Gormes log file a snapshot section belongs to.
type LogClass = diagnostics.LogClass

const (
	LogClassMain      = diagnostics.LogClassMain
	LogClassToolAudit = diagnostics.LogClassToolAudit
)

// LogSnapshotRoots carries the resolved log file paths supplied by the caller.
// SnapshotLogs does not resolve XDG paths itself so it stays unit-testable
// against a t.TempDir().
type LogSnapshotRoots = diagnostics.LogSnapshotRoots

// SnapshotOpts controls how many bytes from the head and tail of each log
// file are kept. Zero or negative values fall back to the defaults
// (64 KiB head, 16 KiB tail).
type SnapshotOpts = diagnostics.SnapshotOpts

// LogSection records the redacted head/tail of a single log file plus
// degraded-mode evidence flags so doctor/status can render output without
// failing when a file is missing or unreadable.
type LogSection = diagnostics.LogSection

// Snapshot is the ordered collection of LogSections SnapshotLogs returns.
// The order is always [LogClassMain, LogClassToolAudit].
type Snapshot = diagnostics.Snapshot

// SnapshotLogs reads the configured log files, applies head/tail truncation,
// and runs each non-empty line through RedactLine.
func SnapshotLogs(roots LogSnapshotRoots, opts SnapshotOpts) Snapshot {
	return diagnostics.SnapshotLogs(roots, opts)
}
