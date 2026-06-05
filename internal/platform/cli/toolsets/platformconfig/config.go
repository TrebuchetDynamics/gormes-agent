package platformconfig

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets/platformconfig/engine"

// PlatformToolsetIssueKind identifies a config-status degraded-mode finding.
type PlatformToolsetIssueKind = engine.PlatformToolsetIssueKind

const (
	PlatformToolsetIssueIgnoredDefaultSuperset    = engine.PlatformToolsetIssueIgnoredDefaultSuperset
	PlatformToolsetIssueHomeAssistantTokenMissing = engine.PlatformToolsetIssueHomeAssistantTokenMissing
	PlatformToolsetIssueNumericKeyNormalized      = engine.PlatformToolsetIssueNumericKeyNormalized
	PlatformToolsetIssueNumericEntryNormalized    = engine.PlatformToolsetIssueNumericEntryNormalized
	PlatformToolsetIssueNoMCPSuppression          = engine.PlatformToolsetIssueNoMCPSuppression
	PlatformToolsetIssueRestrictedToolset         = engine.PlatformToolsetIssueRestrictedToolset
)

// PlatformToolsetIssue records a normalization or degraded-mode decision.
type PlatformToolsetIssue = engine.PlatformToolsetIssue

// PlatformToolsetReport is the pure helper status surface used by future CLI
// config/setup commands.
type PlatformToolsetReport = engine.PlatformToolsetReport

// PlatformToolsetConfig is a YAML-shaped read/write model for Hermes-compatible
// platform_toolsets and mcp_servers config sections.
type PlatformToolsetConfig = engine.PlatformToolsetConfig

// MCPServerConfig captures only the fields needed for platform toolset
// resolution. Unknown MCP server names still round-trip through
// PlatformToolsets as passthrough entries.
type MCPServerConfig = engine.MCPServerConfig

// ParsePlatformToolsetConfig normalizes the config sections touched by
// platform toolset setup without performing file I/O.
func ParsePlatformToolsetConfig(raw any) (PlatformToolsetConfig, PlatformToolsetReport) {
	return engine.ParsePlatformToolsetConfig(raw)
}
