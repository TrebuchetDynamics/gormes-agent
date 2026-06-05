package toolsets

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/toolsets/platformconfig"

type PlatformToolsetIssueKind = platformconfig.PlatformToolsetIssueKind

const (
	PlatformToolsetIssueIgnoredDefaultSuperset    = platformconfig.PlatformToolsetIssueIgnoredDefaultSuperset
	PlatformToolsetIssueHomeAssistantTokenMissing = platformconfig.PlatformToolsetIssueHomeAssistantTokenMissing
	PlatformToolsetIssueNumericKeyNormalized      = platformconfig.PlatformToolsetIssueNumericKeyNormalized
	PlatformToolsetIssueNumericEntryNormalized    = platformconfig.PlatformToolsetIssueNumericEntryNormalized
	PlatformToolsetIssueNoMCPSuppression          = platformconfig.PlatformToolsetIssueNoMCPSuppression
	PlatformToolsetIssueRestrictedToolset         = platformconfig.PlatformToolsetIssueRestrictedToolset
)

type PlatformToolsetIssue = platformconfig.PlatformToolsetIssue
type PlatformToolsetReport = platformconfig.PlatformToolsetReport
type PlatformToolsetConfig = platformconfig.PlatformToolsetConfig
type MCPServerConfig = platformconfig.MCPServerConfig

func ParsePlatformToolsetConfig(raw any) (PlatformToolsetConfig, PlatformToolsetReport) {
	return platformconfig.ParsePlatformToolsetConfig(raw)
}
