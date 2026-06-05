package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display"

// StripANSI removes terminal escape sequences from user-entered text.
func StripANSI(s string) string { return display.StripANSI(s) }
