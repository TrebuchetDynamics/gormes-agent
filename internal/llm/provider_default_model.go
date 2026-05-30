package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type ProviderDefaultModelSource = routing.ProviderDefaultModelSource

const (
	ProviderDefaultModelSourceUnknown         = routing.ProviderDefaultModelSourceUnknown
	ProviderDefaultModelSourceCodexConfig     = routing.ProviderDefaultModelSourceCodexConfig
	ProviderDefaultModelSourceCodexCache      = routing.ProviderDefaultModelSourceCodexCache
	ProviderDefaultModelSourceCuratedFallback = routing.ProviderDefaultModelSourceCuratedFallback
	ProviderDefaultModelSourceStaticCatalog   = routing.ProviderDefaultModelSourceStaticCatalog
)

type ProviderDefaultModelOptions = routing.ProviderDefaultModelOptions
type ProviderDefaultModelResolution = routing.ProviderDefaultModelResolution

func ResolveProviderDefaultModel(provider string, opts ProviderDefaultModelOptions) ProviderDefaultModelResolution {
	return routing.ResolveProviderDefaultModel(provider, opts)
}
