package routing

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/modelcatalog"

type ModelFactStatus = modelcatalog.ModelFactStatus

const (
	ModelFactKnown   ModelFactStatus = modelcatalog.ModelFactKnown
	ModelFactUnknown ModelFactStatus = modelcatalog.ModelFactUnknown
)

type ModelCapabilityFlag = modelcatalog.ModelCapabilityFlag

const (
	ModelCapabilitySupported   ModelCapabilityFlag = modelcatalog.ModelCapabilitySupported
	ModelCapabilityUnsupported ModelCapabilityFlag = modelcatalog.ModelCapabilityUnsupported
	ModelCapabilityUnknown     ModelCapabilityFlag = modelcatalog.ModelCapabilityUnknown
)

type ModelPricingSource = modelcatalog.ModelPricingSource

const (
	ModelPricingSourceNone                 ModelPricingSource = modelcatalog.ModelPricingSourceNone
	ModelPricingSourceOfficialDocsSnapshot ModelPricingSource = modelcatalog.ModelPricingSourceOfficialDocsSnapshot
	ModelPricingSourceModelsDevSnapshot    ModelPricingSource = modelcatalog.ModelPricingSourceModelsDevSnapshot
	ModelPricingSourceProviderModelsAPI    ModelPricingSource = modelcatalog.ModelPricingSourceProviderModelsAPI
)

type ModelRegistrySource = modelcatalog.ModelRegistrySource

const (
	ModelRegistrySourceEmbedded ModelRegistrySource = modelcatalog.ModelRegistrySourceEmbedded
	ModelRegistrySourceTestdata ModelRegistrySource = modelcatalog.ModelRegistrySourceTestdata
)

type ModelRegistryFreshness = modelcatalog.ModelRegistryFreshness

const (
	ModelRegistryFreshnessCurrent ModelRegistryFreshness = modelcatalog.ModelRegistryFreshnessCurrent
	ModelRegistryFreshnessStale   ModelRegistryFreshness = modelcatalog.ModelRegistryFreshnessStale
)

type ModelPricing = modelcatalog.ModelPricing
type ModelCapabilityFlags = modelcatalog.ModelCapabilityFlags
type ModelRegistrySnapshot = modelcatalog.ModelRegistrySnapshot
type ModelRegistryQuery = modelcatalog.ModelRegistryQuery
type ModelRegistryKey = modelcatalog.ModelRegistryKey
type ModelRegistryEntry = modelcatalog.ModelRegistryEntry
type ModelMetadataResult = modelcatalog.ModelMetadataResult
type ModelRegistry = modelcatalog.ModelRegistry

const OllamaCloudProviderID = modelcatalog.OllamaCloudProviderID

func NormalizeProviderModelID(provider, model string) string {
	return modelcatalog.NormalizeProviderModelID(provider, model)
}

func NormalizeModelForProvider(model, provider string) string {
	return modelcatalog.NormalizeModelForProvider(model, provider)
}

func DetectModelVendor(model string) string {
	return modelcatalog.DetectModelVendor(model)
}

func NormalizeOllamaCloudModelID(modelID string) string {
	return modelcatalog.NormalizeOllamaCloudModelID(modelID)
}

func MergeOllamaCloudModelEntries(live, modelsDev []ModelRegistryEntry) []ModelRegistryEntry {
	return modelcatalog.MergeOllamaCloudModelEntries(live, modelsDev)
}

func NewStaticModelRegistry(snapshot ModelRegistrySnapshot, entries []ModelRegistryEntry) ModelRegistry {
	return modelcatalog.NewStaticModelRegistry(snapshot, entries)
}

func DefaultModelRegistry() ModelRegistry {
	return modelcatalog.DefaultModelRegistry()
}

func LookupModelMetadata(query ModelRegistryQuery) ModelMetadataResult {
	return modelcatalog.LookupModelMetadata(query)
}

func UnknownModelPricing() ModelPricing {
	return modelcatalog.UnknownModelPricing()
}

func UnknownModelCapabilities() ModelCapabilityFlags {
	return modelcatalog.UnknownModelCapabilities()
}

func KnownModelPricing(input, output, cacheRead, cacheWrite float64, source ModelPricingSource, version string) ModelPricing {
	return modelcatalog.KnownModelPricing(input, output, cacheRead, cacheWrite, source, version)
}

func KnownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights ModelCapabilityFlag) ModelCapabilityFlags {
	return modelcatalog.KnownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights)
}
