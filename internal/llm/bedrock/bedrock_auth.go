package bedrock

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock/auth"

const (
	BedrockAuthStatePresent = auth.BedrockAuthStatePresent
	BedrockAuthStateMissing = auth.BedrockAuthStateMissing

	BedrockAuthStatusCredentialsMissing = auth.BedrockAuthStatusCredentialsMissing
	BedrockAuthStatusBearerSelected     = auth.BedrockAuthStatusBearerSelected
	BedrockAuthStatusProfileSelected    = auth.BedrockAuthStatusProfileSelected
	BedrockAuthStatusStaticKeySelected  = auth.BedrockAuthStatusStaticKeySelected
	BedrockAuthStatusSigV4Unavailable   = auth.BedrockAuthStatusSigV4Unavailable
)

type BedrockAuthEvidence = auth.BedrockAuthEvidence

func ResolveBedrockAuth(env map[string]string) BedrockAuthEvidence {
	return auth.ResolveBedrockAuth(env)
}

func ResolveBedrockRegion(env map[string]string) string {
	return auth.ResolveBedrockRegion(env)
}
