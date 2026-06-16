package parity

import (
	manifestpkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools/parity/manifest"
	toolsetspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools/parity/toolsets"
)

var upstreamToolParityManifestJSON = manifestpkg.RawUpstreamToolParityManifestJSON()

var ErrMissingToolParityRow = manifestpkg.ErrMissingToolParityRow
var ErrUnknownToolsetDistribution = toolsetspkg.ErrUnknownToolsetDistribution

type ToolParityIssueKind = manifestpkg.ToolParityIssueKind
type ToolsetDistributionIssueKind = toolsetspkg.ToolsetDistributionIssueKind

const (
	ToolParityIssueDisabledTool            = manifestpkg.ToolParityIssueDisabledTool
	ToolParityIssueMissingDependency       = manifestpkg.ToolParityIssueMissingDependency
	ToolParityIssueSchemaDrift             = manifestpkg.ToolParityIssueSchemaDrift
	ToolParityIssueUnavailableProviderPath = manifestpkg.ToolParityIssueUnavailableProviderPath
	ToolParityIssueStaleSourceCommit       = manifestpkg.ToolParityIssueStaleSourceCommit
	ToolParityIssueMissingToolParityRow    = manifestpkg.ToolParityIssueMissingToolParityRow
	ToolParityIssueMissingSchemaProperty   = manifestpkg.ToolParityIssueMissingSchemaProperty
	ToolParityIssueToolsetMismatch         = manifestpkg.ToolParityIssueToolsetMismatch

	ToolsetDistributionIssueUnknownDistribution   = toolsetspkg.ToolsetDistributionIssueUnknownDistribution
	ToolsetDistributionIssueInvalidToolsetSkipped = toolsetspkg.ToolsetDistributionIssueInvalidToolsetSkipped
	ToolsetDistributionIssueFallbackSelected      = toolsetspkg.ToolsetDistributionIssueFallbackSelected
)

type UpstreamToolParityManifest = manifestpkg.UpstreamToolParityManifest
type ToolParitySource = manifestpkg.ToolParitySource
type UpstreamToolParityRow = manifestpkg.UpstreamToolParityRow
type ToolSchemaProvenance = manifestpkg.ToolSchemaProvenance
type ToolDescriptorMetadata = manifestpkg.ToolDescriptorMetadata
type ToolProviderPath = manifestpkg.ToolProviderPath
type ToolResultEnvelope = manifestpkg.ToolResultEnvelope
type ToolDegradedModeStatus = manifestpkg.ToolDegradedModeStatus
type UpstreamToolsetRow = manifestpkg.UpstreamToolsetRow
type ToolPlatformRestrictions = manifestpkg.ToolPlatformRestrictions
type ToolParityDoctorOptions = manifestpkg.ToolParityDoctorOptions
type ToolParityDoctorReport = manifestpkg.ToolParityDoctorReport
type ToolParityIssue = manifestpkg.ToolParityIssue
type ToolsetDistributionEntry = toolsetspkg.ToolsetDistributionEntry
type ToolsetDistribution = toolsetspkg.ToolsetDistribution
type ToolsetDistributionIssue = toolsetspkg.ToolsetDistributionIssue
type ToolsetDistributionSampleOptions = toolsetspkg.ToolsetDistributionSampleOptions
type ToolsetDistributionSample = toolsetspkg.ToolsetDistributionSample

// LoadUpstreamToolParityManifest returns the embedded upstream descriptor
// inventory fixture.
func LoadUpstreamToolParityManifest() (UpstreamToolParityManifest, error) {
	return manifestpkg.LoadUpstreamToolParityManifest()
}

// ListToolsetDistributions returns the ordered Hermes distribution manifest.
func ListToolsetDistributions() []ToolsetDistribution {
	return toolsetspkg.ListToolsetDistributions()
}

// GetToolsetDistribution returns one distribution by name.
func GetToolsetDistribution(name string) (ToolsetDistribution, bool) {
	return toolsetspkg.GetToolsetDistribution(name)
}

// SampleToolsetsFromDistribution samples each toolset independently using the
// Hermes probability contract.
func SampleToolsetsFromDistribution(name string, opts ToolsetDistributionSampleOptions) (ToolsetDistributionSample, error) {
	if opts.ValidateToolset == nil {
		validator, err := upstreamToolsetValidator()
		if err != nil {
			return ToolsetDistributionSample{Distribution: name}, err
		}
		opts.ValidateToolset = validator
	}
	return toolsetspkg.SampleToolsetsFromDistribution(name, opts)
}

// RawUpstreamToolParityManifestJSON returns a defensive copy of the embedded upstream descriptor inventory fixture.
func RawUpstreamToolParityManifestJSON() []byte {
	return manifestpkg.RawUpstreamToolParityManifestJSON()
}

func upstreamToolsetValidator() (func(string) bool, error) {
	manifest, err := LoadUpstreamToolParityManifest()
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(manifest.Toolsets))
	for _, row := range manifest.Toolsets {
		valid[row.Name] = true
	}
	return func(name string) bool { return valid[name] }, nil
}
