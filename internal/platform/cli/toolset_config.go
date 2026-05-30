package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets"

type PlatformToolsetIssueKind = toolsets.PlatformToolsetIssueKind

const (
	PlatformToolsetIssueIgnoredDefaultSuperset    = toolsets.PlatformToolsetIssueIgnoredDefaultSuperset
	PlatformToolsetIssueHomeAssistantTokenMissing = toolsets.PlatformToolsetIssueHomeAssistantTokenMissing
	PlatformToolsetIssueNumericKeyNormalized      = toolsets.PlatformToolsetIssueNumericKeyNormalized
	PlatformToolsetIssueNumericEntryNormalized    = toolsets.PlatformToolsetIssueNumericEntryNormalized
	PlatformToolsetIssueNoMCPSuppression          = toolsets.PlatformToolsetIssueNoMCPSuppression
	PlatformToolsetIssueRestrictedToolset         = toolsets.PlatformToolsetIssueRestrictedToolset
)

type PlatformToolsetIssue = toolsets.PlatformToolsetIssue
type PlatformToolsetReport = toolsets.PlatformToolsetReport
type PlatformToolsetConfig = toolsets.PlatformToolsetConfig
type MCPServerConfig = toolsets.MCPServerConfig

func ParsePlatformToolsetConfig(raw any) (PlatformToolsetConfig, PlatformToolsetReport) {
	return toolsets.ParsePlatformToolsetConfig(raw)
}
