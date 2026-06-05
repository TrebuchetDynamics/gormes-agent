package llm

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock"
)

const (
	BedrockStaleTransportStatus      = bedrock.BedrockStaleTransportStatus
	BedrockNonRetryableRequestStatus = bedrock.BedrockNonRetryableRequestStatus
)

var (
	ErrBedrockConnectionClosed = bedrock.ErrBedrockConnectionClosed
	ErrBedrockProtocolError    = bedrock.ErrBedrockProtocolError
	ErrBedrockReadTimeout      = bedrock.ErrBedrockReadTimeout
	ErrBedrockUnexpectedEOF    = bedrock.ErrBedrockUnexpectedEOF
)

type BedrockRuntimeErrorKind = bedrock.BedrockRuntimeErrorKind

const (
	BedrockRuntimeErrorAssertion          = bedrock.BedrockRuntimeErrorAssertion
	BedrockRuntimeErrorValidation         = bedrock.BedrockRuntimeErrorValidation
	BedrockRuntimeErrorAuth               = bedrock.BedrockRuntimeErrorAuth
	BedrockRuntimeErrorMissingCredentials = bedrock.BedrockRuntimeErrorMissingCredentials
	BedrockRuntimeErrorMalformedRequest   = bedrock.BedrockRuntimeErrorMalformedRequest
)

type BedrockRuntimeError = bedrock.BedrockRuntimeError
type BedrockStaleErrorClassification = bedrock.BedrockStaleErrorClassification

func ClassifyBedrockStaleError(err error) BedrockStaleErrorClassification {
	return bedrock.ClassifyBedrockStaleError(err)
}
