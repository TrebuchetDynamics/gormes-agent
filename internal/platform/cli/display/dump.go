package display

import dumpmodel "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display/dump"

// DumpInput is the pure input model for a deterministic support-summary dump.
// Callers populate the fields from runtime sources at the boundary so
// RenderDumpSummary can stay file/clock/env/network inert.
type DumpInput = dumpmodel.Input

// RenderDumpSummary returns a deterministic, plain-text summary of the supplied
// DumpInput.
func RenderDumpSummary(in DumpInput) string { return dumpmodel.RenderSummary(in) }
