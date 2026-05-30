package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock"

var ErrBedrockRuntimeClientMissing = bedrock.ErrBedrockRuntimeClientMissing

type BedrockRuntimeRequest = bedrock.BedrockRuntimeRequest
type BedrockRuntimeResponse = bedrock.BedrockRuntimeResponse
type BedrockRuntimeStreamResponse = bedrock.BedrockRuntimeStreamResponse
type BedrockRuntimeClient = bedrock.BedrockRuntimeClient
type BedrockRuntimeClientFactory = bedrock.BedrockRuntimeClientFactory
type BedrockRuntimeCache = bedrock.BedrockRuntimeCache

func NewBedrockRuntimeCache(newClient BedrockRuntimeClientFactory) *BedrockRuntimeCache {
	return bedrock.NewBedrockRuntimeCache(newClient)
}
