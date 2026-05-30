package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/diagnostics"

// RedactLine replaces known secret-shaped byte spans with "[REDACTED]".
// It returns the input slice unchanged when no redactions are applied.
func RedactLine(line []byte) ([]byte, int) { return diagnostics.RedactLine(line) }
