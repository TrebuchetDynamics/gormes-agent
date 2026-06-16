package manifest

import contract "github.com/TrebuchetDynamics/gormes-agent/internal/tools/parity/manifest/contract"

// ErrMissingToolParityRow is returned when a handler port is marked complete
// before its upstream descriptor has a parity fixture row.
var ErrMissingToolParityRow = contract.ErrMissingToolParityRow

// ToolParityIssueKind identifies a degraded-mode doctor finding.
type ToolParityIssueKind = contract.ToolParityIssueKind

const (
	ToolParityIssueDisabledTool            = contract.ToolParityIssueDisabledTool
	ToolParityIssueMissingDependency       = contract.ToolParityIssueMissingDependency
	ToolParityIssueSchemaDrift             = contract.ToolParityIssueSchemaDrift
	ToolParityIssueUnavailableProviderPath = contract.ToolParityIssueUnavailableProviderPath
	ToolParityIssueStaleSourceCommit       = contract.ToolParityIssueStaleSourceCommit
	ToolParityIssueMissingToolParityRow    = contract.ToolParityIssueMissingToolParityRow
	ToolParityIssueMissingSchemaProperty   = contract.ToolParityIssueMissingSchemaProperty
	ToolParityIssueToolsetMismatch         = contract.ToolParityIssueToolsetMismatch
)

// UpstreamToolParityManifest is the frozen donor descriptor inventory used to
// gate later handler ports.
type UpstreamToolParityManifest = contract.UpstreamToolParityManifest

// ToolParitySource records the donor files used to capture the fixture.
type ToolParitySource = contract.ToolParitySource

// UpstreamToolParityRow captures the model-visible descriptor plus the
// operational metadata that must exist before porting a handler.
type UpstreamToolParityRow = contract.UpstreamToolParityRow

// ToolSchemaProvenance records dynamic schema replacement seams in the donor.
type ToolSchemaProvenance = contract.ToolSchemaProvenance

// ToolDescriptorMetadata captures descriptor-only parity notes that are not
// part of OpenAI function schemas.
type ToolDescriptorMetadata = contract.ToolDescriptorMetadata

// ToolProviderPath captures optional provider-specific availability gates.
type ToolProviderPath = contract.ToolProviderPath

// ToolResultEnvelope captures the JSON fields the donor returns on success or
// failure. Handler ports can refine these rows before they claim completion.
type ToolResultEnvelope = contract.ToolResultEnvelope

// ToolDegradedModeStatus captures how doctor should report degraded tools.
type ToolDegradedModeStatus = contract.ToolDegradedModeStatus

// UpstreamToolsetRow captures static and resolved donor toolset membership.
type UpstreamToolsetRow = contract.UpstreamToolsetRow

// ToolPlatformRestrictions records platform-scoped toolset availability from
// the donor CLI configuration tests.
type ToolPlatformRestrictions = contract.ToolPlatformRestrictions

// ToolParityDoctorOptions controls degraded-mode inventory checks.
type ToolParityDoctorOptions = contract.ToolParityDoctorOptions

// ToolParityDoctorReport is the aggregate doctor output for descriptor parity.
type ToolParityDoctorReport = contract.ToolParityDoctorReport

// ToolParityIssue is one degraded-mode doctor finding.
type ToolParityIssue = contract.ToolParityIssue

// LoadUpstreamToolParityManifest returns the embedded upstream descriptor
// inventory fixture.
func LoadUpstreamToolParityManifest() (UpstreamToolParityManifest, error) {
	return contract.LoadUpstreamToolParityManifest()
}

// RawUpstreamToolParityManifestJSON returns a defensive copy of the embedded upstream descriptor inventory fixture.
func RawUpstreamToolParityManifestJSON() []byte {
	return contract.RawUpstreamToolParityManifestJSON()
}
