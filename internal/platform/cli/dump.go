package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display"

// DumpInput is the pure input model for a deterministic support-summary dump.
// Callers populate the fields from runtime sources at the boundary so
// RenderDumpSummary can stay file/clock/env/network inert.
type DumpInput = display.DumpInput

// RenderDumpSummary returns a deterministic, plain-text summary of the
// supplied DumpInput. Lines are emitted in the fixed order
// version, os, arch, profile, toolsets and every literal occurrence of any
// SecretsLikeKeys entry is replaced with "[redacted]" before returning.
// Empty scalar fields render as "unknown" and an empty Toolsets renders as
// "(none)".
func RenderDumpSummary(in DumpInput) string { return display.RenderDumpSummary(in) }
