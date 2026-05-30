package diagnostics

import "time"

// DefaultSlowBracketedPasteThreshold mirrors the upstream Hermes 500ms
// signal that bracketed-paste handling stalled long enough to warrant a
// structured diagnostic event. Hermes ac0325c2 introduced this threshold to
// catch terminals where prompt_toolkit hangs on large pastes.
const DefaultSlowBracketedPasteThreshold = 500 * time.Millisecond

// PasteEventLogger is the narrow logging seam the diagnostic helper uses.
// Implementations route the structured event into Gormes audit/telemetry
// without ever receiving raw paste content.
type PasteEventLogger interface {
	Log(name string, fields map[string]any)
}

// SlowBracketedPasteSample carries the per-paste evidence. Only the
// Duration is read for the diagnostic decision; PastedContent and
// PastedFilePaths are accepted so callers can pass the full paste record
// without having to redact upstream — the helper drops them.
type SlowBracketedPasteSample struct {
	Duration        time.Duration
	PastedContent   string
	PastedFilePaths []string
}

// SlowBracketedPasteDiagnostic emits a redacted "paste_handler_slow" event
// when bracketed-paste handling exceeds the configured threshold.
type SlowBracketedPasteDiagnostic struct {
	logger    PasteEventLogger
	threshold time.Duration
}

// NewSlowBracketedPasteDiagnostic returns a recorder that forwards slow
// paste events to logger. A non-positive threshold falls back to
// DefaultSlowBracketedPasteThreshold.
func NewSlowBracketedPasteDiagnostic(logger PasteEventLogger, threshold time.Duration) *SlowBracketedPasteDiagnostic {
	if threshold <= 0 {
		threshold = DefaultSlowBracketedPasteThreshold
	}
	return &SlowBracketedPasteDiagnostic{logger: logger, threshold: threshold}
}

// Threshold reports the active threshold for diagnostic emission.
func (d *SlowBracketedPasteDiagnostic) Threshold() time.Duration {
	if d == nil {
		return DefaultSlowBracketedPasteThreshold
	}
	return d.threshold
}

// Record inspects one paste sample and emits a structured event when the
// duration meets or exceeds the threshold. The pasted content and any
// pasted file paths are deliberately omitted so the log never carries
// secrets or filesystem fingerprints.
func (d *SlowBracketedPasteDiagnostic) Record(sample SlowBracketedPasteSample) {
	if d == nil || d.logger == nil {
		return
	}
	if sample.Duration < d.threshold {
		return
	}
	d.logger.Log("paste_handler_slow", map[string]any{
		"duration_ms":  sample.Duration.Milliseconds(),
		"threshold_ms": d.threshold.Milliseconds(),
	})
}
