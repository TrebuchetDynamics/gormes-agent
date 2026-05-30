package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type ModelContextSource = routing.ModelContextSource

const (
	ModelContextSourceProviderCap = routing.ModelContextSourceProviderCap
	ModelContextSourceModelsDev   = routing.ModelContextSourceModelsDev
	ModelContextSourceUnknown     = routing.ModelContextSourceUnknown
)

type ModelContextMetadata = routing.ModelContextMetadata
type ModelContextQuery = routing.ModelContextQuery
type ModelContextResolution = routing.ModelContextResolution
type ModelContextLookup = routing.ModelContextLookup
type ModelContextLookupFunc = routing.ModelContextLookupFunc
type ModelContextKey = routing.ModelContextKey
type StaticModelContextCaps = routing.StaticModelContextCaps
type ModelContextResolver = routing.ModelContextResolver

func NewModelContextResolver(providerCaps ModelContextLookup) ModelContextResolver {
	return routing.NewModelContextResolver(providerCaps)
}

func DefaultModelContextResolver() ModelContextResolver {
	return routing.DefaultModelContextResolver()
}

func ResolveDisplayContextLength(query ModelContextQuery) ModelContextResolution {
	return routing.ResolveDisplayContextLength(query)
}

func normalizeModelContextProvider(provider string) string {
	return routing.NormalizeModelContextProvider(provider)
}

func normalizeModelContextText(value string) string {
	return routing.NormalizeModelContextText(value)
}
