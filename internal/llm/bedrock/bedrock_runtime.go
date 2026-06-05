package bedrock

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/bedrock/runtime"

var ErrBedrockRuntimeClientMissing = runtime.ErrBedrockRuntimeClientMissing

type BedrockRuntimeRequest = runtime.BedrockRuntimeRequest
type BedrockRuntimeResponse = runtime.BedrockRuntimeResponse
type BedrockRuntimeStreamResponse = runtime.BedrockRuntimeStreamResponse
type BedrockRuntimeClient = runtime.BedrockRuntimeClient
type BedrockRuntimeClientFactory = runtime.BedrockRuntimeClientFactory
type BedrockRuntimeCache = runtime.BedrockRuntimeCache

func NewBedrockRuntimeCache(newClient BedrockRuntimeClientFactory) *BedrockRuntimeCache {
	return runtime.NewBedrockRuntimeCache(newClient)
}
