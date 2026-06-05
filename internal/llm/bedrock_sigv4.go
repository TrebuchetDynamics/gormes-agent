package llm

import (
	"net/http"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock"
)

const (
	BedrockAuthStatePresent = bedrock.BedrockAuthStatePresent
	BedrockAuthStateMissing = bedrock.BedrockAuthStateMissing

	BedrockAuthStatusCredentialsMissing = bedrock.BedrockAuthStatusCredentialsMissing
	BedrockAuthStatusBearerSelected     = bedrock.BedrockAuthStatusBearerSelected
	BedrockAuthStatusProfileSelected    = bedrock.BedrockAuthStatusProfileSelected
	BedrockAuthStatusStaticKeySelected  = bedrock.BedrockAuthStatusStaticKeySelected
	BedrockAuthStatusSigV4Unavailable   = bedrock.BedrockAuthStatusSigV4Unavailable
)

type BedrockAuthEvidence = bedrock.BedrockAuthEvidence
type StaticAWSCredentials = bedrock.StaticAWSCredentials

func ResolveBedrockAuth(env map[string]string) BedrockAuthEvidence {
	return bedrock.ResolveBedrockAuth(env)
}

func ResolveBedrockRegion(env map[string]string) string {
	return bedrock.ResolveBedrockRegion(env)
}

func SignBedrockRequest(req *http.Request, creds StaticAWSCredentials, now time.Time) error {
	return bedrock.SignBedrockRequest(req, creds, now)
}
