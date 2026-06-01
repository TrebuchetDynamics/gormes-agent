package diagnostics

import (
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics/paste"
)

// DefaultSlowBracketedPasteThreshold mirrors the upstream Hermes 500ms signal.
const DefaultSlowBracketedPasteThreshold = paste.DefaultSlowBracketedPasteThreshold

// PasteEventLogger is the narrow logging seam the diagnostic helper uses.
type PasteEventLogger = paste.PasteEventLogger

// SlowBracketedPasteSample carries the per-paste evidence.
type SlowBracketedPasteSample = paste.SlowBracketedPasteSample

// SlowBracketedPasteDiagnostic emits a redacted "paste_handler_slow" event.
type SlowBracketedPasteDiagnostic = paste.SlowBracketedPasteDiagnostic

// NewSlowBracketedPasteDiagnostic returns a recorder that forwards slow paste events to logger.
func NewSlowBracketedPasteDiagnostic(logger PasteEventLogger, threshold time.Duration) *SlowBracketedPasteDiagnostic {
	return paste.NewSlowBracketedPasteDiagnostic(logger, threshold)
}
