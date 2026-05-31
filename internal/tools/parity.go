package tools

import "github.com/TrebuchetDynamics/gormes-agent/internal/tools/parity"

// upstreamToolParityManifestJSON is kept for legacy root-package tests that
// inspect the frozen donor fixture directly.
var upstreamToolParityManifestJSON = parity.RawUpstreamToolParityManifestJSON()

// ErrMissingToolParityRow is returned when a handler port is marked complete
// before its upstream descriptor has a parity fixture row.
var ErrMissingToolParityRow = parity.ErrMissingToolParityRow

type ToolParityIssueKind = parity.ToolParityIssueKind

const (
	ToolParityIssueDisabledTool            = parity.ToolParityIssueDisabledTool
	ToolParityIssueMissingDependency       = parity.ToolParityIssueMissingDependency
	ToolParityIssueSchemaDrift             = parity.ToolParityIssueSchemaDrift
	ToolParityIssueUnavailableProviderPath = parity.ToolParityIssueUnavailableProviderPath
	ToolParityIssueStaleSourceCommit       = parity.ToolParityIssueStaleSourceCommit
	ToolParityIssueMissingToolParityRow    = parity.ToolParityIssueMissingToolParityRow
	ToolParityIssueMissingSchemaProperty   = parity.ToolParityIssueMissingSchemaProperty
	ToolParityIssueToolsetMismatch         = parity.ToolParityIssueToolsetMismatch
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

// LoadUpstreamToolParityManifest returns the embedded upstream descriptor
// inventory fixture.
func LoadUpstreamToolParityManifest() (UpstreamToolParityManifest, error) {
	return parity.LoadUpstreamToolParityManifest()
}
