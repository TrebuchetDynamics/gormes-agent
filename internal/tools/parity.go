package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/parity"

// upstreamToolParityManifestJSON is kept for legacy root-package tests that
// inspect the frozen donor fixture directly.
var upstreamToolParityManifestJSON = parity.RawUpstreamToolParityManifestJSON()

// ErrMissingToolParityRow is returned when a handler port is marked complete
// before its upstream descriptor has a parity fixture row.
var ErrMissingToolParityRow = parity.ErrMissingToolParityRow
var ErrUnknownToolsetDistribution = parity.ErrUnknownToolsetDistribution

type ToolParityIssueKind = parity.ToolParityIssueKind
type ToolsetDistributionIssueKind = parity.ToolsetDistributionIssueKind

const (
	ToolParityIssueDisabledTool            = parity.ToolParityIssueDisabledTool
	ToolParityIssueMissingDependency       = parity.ToolParityIssueMissingDependency
	ToolParityIssueSchemaDrift             = parity.ToolParityIssueSchemaDrift
	ToolParityIssueUnavailableProviderPath = parity.ToolParityIssueUnavailableProviderPath
	ToolParityIssueStaleSourceCommit       = parity.ToolParityIssueStaleSourceCommit
	ToolParityIssueMissingToolParityRow    = parity.ToolParityIssueMissingToolParityRow
	ToolParityIssueMissingSchemaProperty   = parity.ToolParityIssueMissingSchemaProperty
	ToolParityIssueToolsetMismatch         = parity.ToolParityIssueToolsetMismatch

	ToolsetDistributionIssueUnknownDistribution   = parity.ToolsetDistributionIssueUnknownDistribution
	ToolsetDistributionIssueInvalidToolsetSkipped = parity.ToolsetDistributionIssueInvalidToolsetSkipped
	ToolsetDistributionIssueFallbackSelected      = parity.ToolsetDistributionIssueFallbackSelected
)

type UpstreamToolParityManifest = parity.UpstreamToolParityManifest
type ToolParitySource = parity.ToolParitySource
type UpstreamToolParityRow = parity.UpstreamToolParityRow
type ToolSchemaProvenance = parity.ToolSchemaProvenance
type ToolDescriptorMetadata = parity.ToolDescriptorMetadata
type ToolProviderPath = parity.ToolProviderPath
type ToolResultEnvelope = parity.ToolResultEnvelope
type ToolDegradedModeStatus = parity.ToolDegradedModeStatus
type UpstreamToolsetRow = parity.UpstreamToolsetRow
type ToolPlatformRestrictions = parity.ToolPlatformRestrictions
type ToolParityDoctorOptions = parity.ToolParityDoctorOptions
type ToolParityDoctorReport = parity.ToolParityDoctorReport
type ToolParityIssue = parity.ToolParityIssue
type ToolsetDistributionEntry = parity.ToolsetDistributionEntry
type ToolsetDistribution = parity.ToolsetDistribution
type ToolsetDistributionIssue = parity.ToolsetDistributionIssue
type ToolsetDistributionSampleOptions = parity.ToolsetDistributionSampleOptions
type ToolsetDistributionSample = parity.ToolsetDistributionSample

// LoadUpstreamToolParityManifest returns the embedded upstream descriptor
// inventory fixture.
func LoadUpstreamToolParityManifest() (UpstreamToolParityManifest, error) {
	return parity.LoadUpstreamToolParityManifest()
}

// ListToolsetDistributions returns the ordered Hermes distribution manifest.
func ListToolsetDistributions() []ToolsetDistribution {
	return parity.ListToolsetDistributions()
}

// GetToolsetDistribution returns one distribution by name.
func GetToolsetDistribution(name string) (ToolsetDistribution, bool) {
	return parity.GetToolsetDistribution(name)
}

// SampleToolsetsFromDistribution samples each toolset independently using the
// Hermes probability contract.
func SampleToolsetsFromDistribution(name string, opts ToolsetDistributionSampleOptions) (ToolsetDistributionSample, error) {
	return parity.SampleToolsetsFromDistribution(name, opts)
}
