package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"

var logRedactedMarker = []byte("[REDACTED]")

// RedactLine replaces known secret-shaped byte spans with "[REDACTED]".
// It returns the input slice unchanged when no redactions are applied.
func RedactLine(line []byte) ([]byte, int) {
	out, total := redaction.RedactSecretsWithCount(string(line), string(logRedactedMarker))
	return []byte(out), total
}
