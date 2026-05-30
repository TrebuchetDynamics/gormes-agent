// Package channels contains adapter-neutral channel compatibility facades.
package channels

import channelcaps "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/capabilities"

// CapabilityOptions configures a read-only channel capability report.
type CapabilityOptions = channelcaps.CapabilityOptions

// CapabilitySupport mirrors the manifest surface statuses that operators need
// when deciding whether a channel supports a workflow.
type CapabilitySupport = channelcaps.CapabilitySupport

// CapabilityReport is a redacted, live-SDK-free channel capability row.
type CapabilityReport = channelcaps.CapabilityReport

// UnknownChannelError reports an operator-requested channel that is absent
// from the source-backed platform manifest.
type UnknownChannelError = channelcaps.UnknownChannelError

// BuildCapabilityReports returns source-backed channel capability metadata in
// stable order. It deliberately reads only the checked-in gateway manifest and
// redacted configured-channel names supplied by the caller.
func BuildCapabilityReports(opts CapabilityOptions) ([]CapabilityReport, error) {
	return channelcaps.BuildCapabilityReports(opts)
}
