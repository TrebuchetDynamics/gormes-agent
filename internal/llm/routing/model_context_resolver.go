package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/contextwindow"

type ModelContextSource = contextwindow.ModelContextSource

const (
	ModelContextSourceProviderCap ModelContextSource = contextwindow.ModelContextSourceProviderCap
	ModelContextSourceModelsDev   ModelContextSource = contextwindow.ModelContextSourceModelsDev
	ModelContextSourceUnknown     ModelContextSource = contextwindow.ModelContextSourceUnknown
)

type ModelContextMetadata = contextwindow.ModelContextMetadata
type ModelContextQuery = contextwindow.ModelContextQuery
type ModelContextResolution = contextwindow.ModelContextResolution
type ModelContextLookup = contextwindow.ModelContextLookup
type ModelContextLookupFunc = contextwindow.ModelContextLookupFunc
type ModelContextKey = contextwindow.ModelContextKey
type StaticModelContextCaps = contextwindow.StaticModelContextCaps
type ModelContextResolver = contextwindow.ModelContextResolver

func NewModelContextResolver(providerCaps ModelContextLookup) ModelContextResolver {
	return contextwindow.NewModelContextResolver(providerCaps)
}

func DefaultModelContextResolver() ModelContextResolver {
	return contextwindow.DefaultModelContextResolver()
}

func ResolveDisplayContextLength(query ModelContextQuery) ModelContextResolution {
	return contextwindow.ResolveDisplayContextLength(query)
}

func normalizeModelContextProvider(provider string) string {
	return contextwindow.NormalizeModelContextProvider(provider)
}

func normalizeModelContextText(value string) string {
	return contextwindow.NormalizeModelContextText(value)
}

func NormalizeModelContextProvider(provider string) string {
	return contextwindow.NormalizeModelContextProvider(provider)
}

func NormalizeModelContextText(value string) string {
	return contextwindow.NormalizeModelContextText(value)
}
