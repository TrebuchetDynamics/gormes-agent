package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"

type ModelFactStatus = routing.ModelFactStatus

const (
	ModelFactKnown   = routing.ModelFactKnown
	ModelFactUnknown = routing.ModelFactUnknown
)

type ModelCapabilityFlag = routing.ModelCapabilityFlag

const (
	ModelCapabilitySupported   = routing.ModelCapabilitySupported
	ModelCapabilityUnsupported = routing.ModelCapabilityUnsupported
	ModelCapabilityUnknown     = routing.ModelCapabilityUnknown
)

type ModelPricingSource = routing.ModelPricingSource

const (
	ModelPricingSourceNone                 = routing.ModelPricingSourceNone
	ModelPricingSourceOfficialDocsSnapshot = routing.ModelPricingSourceOfficialDocsSnapshot
	ModelPricingSourceModelsDevSnapshot    = routing.ModelPricingSourceModelsDevSnapshot
	ModelPricingSourceProviderModelsAPI    = routing.ModelPricingSourceProviderModelsAPI
)

type ModelRegistrySource = routing.ModelRegistrySource

const (
	ModelRegistrySourceEmbedded = routing.ModelRegistrySourceEmbedded
	ModelRegistrySourceTestdata = routing.ModelRegistrySourceTestdata
)

type ModelRegistryFreshness = routing.ModelRegistryFreshness

const (
	ModelRegistryFreshnessCurrent = routing.ModelRegistryFreshnessCurrent
	ModelRegistryFreshnessStale   = routing.ModelRegistryFreshnessStale
)

type ModelPricing = routing.ModelPricing
type ModelCapabilityFlags = routing.ModelCapabilityFlags
type ModelRegistrySnapshot = routing.ModelRegistrySnapshot
type ModelRegistryQuery = routing.ModelRegistryQuery
type ModelRegistryKey = routing.ModelRegistryKey
type ModelRegistryEntry = routing.ModelRegistryEntry

const OllamaCloudProviderID = routing.OllamaCloudProviderID

func NormalizeProviderModelID(provider, model string) string {
	return routing.NormalizeProviderModelID(provider, model)
}

func NormalizeModelForProvider(model, provider string) string {
	return routing.NormalizeModelForProvider(model, provider)
}

func DetectModelVendor(model string) string {
	return routing.DetectModelVendor(model)
}

func NormalizeOllamaCloudModelID(modelID string) string {
	return routing.NormalizeOllamaCloudModelID(modelID)
}

func MergeOllamaCloudModelEntries(live, modelsDev []ModelRegistryEntry) []ModelRegistryEntry {
	return routing.MergeOllamaCloudModelEntries(live, modelsDev)
}

type ModelMetadataResult = routing.ModelMetadataResult
type ModelRegistry = routing.ModelRegistry

func NewStaticModelRegistry(snapshot ModelRegistrySnapshot, entries []ModelRegistryEntry) ModelRegistry {
	return routing.NewStaticModelRegistry(snapshot, entries)
}

func DefaultModelRegistry() ModelRegistry {
	return routing.DefaultModelRegistry()
}

func LookupModelMetadata(query ModelRegistryQuery) ModelMetadataResult {
	return routing.LookupModelMetadata(query)
}

func unknownModelPricing() ModelPricing {
	return routing.UnknownModelPricing()
}

func unknownModelCapabilities() ModelCapabilityFlags {
	return routing.UnknownModelCapabilities()
}

func knownModelPricing(input, output, cacheRead, cacheWrite float64, source ModelPricingSource, version string) ModelPricing {
	return routing.KnownModelPricing(input, output, cacheRead, cacheWrite, source, version)
}

func knownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights ModelCapabilityFlag) ModelCapabilityFlags {
	return routing.KnownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights)
}
