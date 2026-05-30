package gateway

import gatewaysteer "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/steercmd"

// Steer evidence strings are stable degraded-mode reasons surfaced before any
// future queue or running-agent dispatch path can handle /steer.
type SteerEvidence = gatewaysteer.Evidence

const (
	SteerEvidenceUsage              = gatewaysteer.EvidenceUsage
	SteerEvidencePayloadUnsupported = gatewaysteer.EvidencePayloadUnsupported
	SteerEvidenceQueued             = gatewaysteer.EvidenceQueued
	SteerEvidenceInjected           = gatewaysteer.EvidenceInjected
	SteerEvidenceUnavailable        = gatewaysteer.EvidenceUnavailable
	SteerEvidencePreview            = gatewaysteer.EvidencePreview
)

const SteerPreviewMaxRunes = gatewaysteer.PreviewMaxRunes

// SteerPayloadMetadata carries synthetic media counts for the pure parser.
// Platform adapters keep their own attachment details out of this slice.
type SteerPayloadMetadata = gatewaysteer.PayloadMetadata

// SteerCommand is the parsed shape of a /steer invocation.
type SteerCommand = gatewaysteer.Command

// ParseSteerCommand turns raw /steer slash text plus payload metadata into a
// pure parser result. It performs no queueing, session mutation, or dispatch.
func ParseSteerCommand(raw string, payload SteerPayloadMetadata) SteerCommand {
	return gatewaysteer.Parse(raw, payload)
}

// SteerPreview returns deterministic, bounded guidance text for acknowledgments
// and evidence. Truncation is marked with an ASCII suffix.
func SteerPreview(guidance string) string {
	return gatewaysteer.Preview(guidance)
}
