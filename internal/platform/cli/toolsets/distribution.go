package toolsets

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools"

var ErrUnknownToolsetDistribution = tools.ErrUnknownToolsetDistribution

type ToolsetDistributionIssueKind = tools.ToolsetDistributionIssueKind

const (
	ToolsetDistributionIssueUnknownDistribution   = tools.ToolsetDistributionIssueUnknownDistribution
	ToolsetDistributionIssueInvalidToolsetSkipped = tools.ToolsetDistributionIssueInvalidToolsetSkipped
	ToolsetDistributionIssueFallbackSelected      = tools.ToolsetDistributionIssueFallbackSelected
)

type ToolsetDistributionEntry = tools.ToolsetDistributionEntry
type ToolsetDistribution = tools.ToolsetDistribution
type ToolsetDistributionIssue = tools.ToolsetDistributionIssue
type ToolsetDistributionSampleOptions = tools.ToolsetDistributionSampleOptions
type ToolsetDistributionSample = tools.ToolsetDistributionSample

// ListToolsetDistributions returns the ordered Hermes distribution manifest.
func ListToolsetDistributions() []ToolsetDistribution {
	return tools.ListToolsetDistributions()
}

// GetToolsetDistribution returns one distribution by name.
func GetToolsetDistribution(name string) (ToolsetDistribution, bool) {
	return tools.GetToolsetDistribution(name)
}

// SampleToolsetsFromDistribution samples each toolset independently using the
// Hermes probability contract.
func SampleToolsetsFromDistribution(name string, opts ToolsetDistributionSampleOptions) (ToolsetDistributionSample, error) {
	return tools.SampleToolsetsFromDistribution(name, opts)
}
