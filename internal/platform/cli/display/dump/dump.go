package dump

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/redaction"
)

// Input is the pure input model for a deterministic support-summary dump.
// Callers populate the fields from runtime sources at the boundary so
// RenderSummary can stay file/clock/env/network inert.
type Input struct {
	Version         string
	OS              string
	Arch            string
	ProfileName     string
	Toolsets        []string
	SecretsLikeKeys []string
}

// RenderSummary returns a deterministic, plain-text summary of the supplied
// Input. Lines are emitted in the fixed order version, os, arch, profile,
// toolsets and every literal occurrence of any SecretsLikeKeys entry is
// replaced with "[redacted]" before returning. Empty scalar fields render as
// "unknown" and an empty Toolsets renders as "(none)".
func RenderSummary(in Input) string {
	var b strings.Builder
	writeLine(&b, "version", scalarOrUnknown(in.Version))
	writeLine(&b, "os", scalarOrUnknown(in.OS))
	writeLine(&b, "arch", scalarOrUnknown(in.Arch))
	writeLine(&b, "profile", scalarOrUnknown(in.ProfileName))
	writeLine(&b, "toolsets", toolsetsValue(in.Toolsets))
	return redaction.RedactLiterals(b.String(), in.SecretsLikeKeys, "[redacted]")
}

func writeLine(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteByte('\n')
}

func scalarOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func toolsetsValue(toolsets []string) string {
	if len(toolsets) == 0 {
		return "(none)"
	}
	return strings.Join(toolsets, ", ")
}
