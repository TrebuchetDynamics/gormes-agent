package display

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display/sanitizing"

// StripANSI removes terminal escape sequences from user-entered text.
func StripANSI(s string) string { return sanitizing.StripANSI(s) }
