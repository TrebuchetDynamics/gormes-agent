package display

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"

// StripANSI removes terminal escape sequences from user-entered text.
func StripANSI(s string) string {
	return redaction.StripANSI(s)
}
