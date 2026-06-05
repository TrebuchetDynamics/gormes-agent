package cli

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics"
)

// DefaultSlowBracketedPasteThreshold mirrors the upstream Hermes 500ms
// signal that bracketed-paste handling stalled long enough to warrant a
// structured diagnostic event.
const DefaultSlowBracketedPasteThreshold = diagnostics.DefaultSlowBracketedPasteThreshold

// PasteEventLogger is the narrow logging seam the diagnostic helper uses.
type PasteEventLogger = diagnostics.PasteEventLogger

// SlowBracketedPasteSample carries the per-paste evidence.
type SlowBracketedPasteSample = diagnostics.SlowBracketedPasteSample

// SlowBracketedPasteDiagnostic emits a redacted "paste_handler_slow" event
// when bracketed-paste handling exceeds the configured threshold.
type SlowBracketedPasteDiagnostic = diagnostics.SlowBracketedPasteDiagnostic

// NewSlowBracketedPasteDiagnostic returns a recorder that forwards slow
// paste events to logger. A non-positive threshold falls back to
// DefaultSlowBracketedPasteThreshold.
func NewSlowBracketedPasteDiagnostic(logger PasteEventLogger, threshold time.Duration) *SlowBracketedPasteDiagnostic {
	return diagnostics.NewSlowBracketedPasteDiagnostic(logger, threshold)
}
