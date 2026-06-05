package bedrock

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock/stale"

const (
	BedrockStaleTransportStatus      = stale.BedrockStaleTransportStatus
	BedrockNonRetryableRequestStatus = stale.BedrockNonRetryableRequestStatus
)

var (
	ErrBedrockConnectionClosed = stale.ErrBedrockConnectionClosed
	ErrBedrockProtocolError    = stale.ErrBedrockProtocolError
	ErrBedrockReadTimeout      = stale.ErrBedrockReadTimeout
	ErrBedrockUnexpectedEOF    = stale.ErrBedrockUnexpectedEOF
)

type BedrockRuntimeErrorKind = stale.BedrockRuntimeErrorKind

const (
	BedrockRuntimeErrorAssertion          = stale.BedrockRuntimeErrorAssertion
	BedrockRuntimeErrorValidation         = stale.BedrockRuntimeErrorValidation
	BedrockRuntimeErrorAuth               = stale.BedrockRuntimeErrorAuth
	BedrockRuntimeErrorMissingCredentials = stale.BedrockRuntimeErrorMissingCredentials
	BedrockRuntimeErrorMalformedRequest   = stale.BedrockRuntimeErrorMalformedRequest
)

type BedrockRuntimeError = stale.BedrockRuntimeError
type BedrockStaleErrorClassification = stale.BedrockStaleErrorClassification

func ClassifyBedrockStaleError(err error) BedrockStaleErrorClassification {
	return stale.ClassifyBedrockStaleError(err)
}
