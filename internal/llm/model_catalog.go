package llm

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/modelcatalog"
)

const SupportedModelCatalogSchemaVersion = modelcatalog.SupportedModelCatalogSchemaVersion

type ModelCatalogSource = modelcatalog.ModelCatalogSource

const (
	ModelCatalogSourceDisabled  ModelCatalogSource = modelcatalog.ModelCatalogSourceDisabled
	ModelCatalogSourceMemory    ModelCatalogSource = modelcatalog.ModelCatalogSourceMemory
	ModelCatalogSourceDisk      ModelCatalogSource = modelcatalog.ModelCatalogSourceDisk
	ModelCatalogSourceDiskStale ModelCatalogSource = modelcatalog.ModelCatalogSourceDiskStale
	ModelCatalogSourceNetwork   ModelCatalogSource = modelcatalog.ModelCatalogSourceNetwork
	ModelCatalogSourceOverride  ModelCatalogSource = modelcatalog.ModelCatalogSourceOverride
	ModelCatalogSourceEmpty     ModelCatalogSource = modelcatalog.ModelCatalogSourceEmpty
)

type ModelCatalogConfig = modelcatalog.ModelCatalogConfig
type ModelCatalogOptions = modelcatalog.ModelCatalogOptions
type ModelCatalogFetcher = modelcatalog.ModelCatalogFetcher
type ModelCatalog = modelcatalog.ModelCatalog
type ModelCatalogManifest = modelcatalog.ModelCatalogManifest
type ModelCatalogProvider = modelcatalog.ModelCatalogProvider
type ModelCatalogModel = modelcatalog.ModelCatalogModel
type ModelCatalogChoice = modelcatalog.ModelCatalogChoice
type ModelCatalogEvidence = modelcatalog.ModelCatalogEvidence
type AIGatewayPricing = modelcatalog.AIGatewayPricing

func NewModelCatalog(opts ModelCatalogOptions) *ModelCatalog {
	return modelcatalog.NewModelCatalog(opts)
}

func ValidateModelCatalogManifest(manifest ModelCatalogManifest) error {
	return modelcatalog.ValidateModelCatalogManifest(manifest)
}

func ParseAIGatewayModelPricing(payload []byte) (map[string]AIGatewayPricing, error) {
	return modelcatalog.ParseAIGatewayModelPricing(payload)
}

func MergeAIGatewayFreePromotions(curated []ModelCatalogChoice, payload []byte) ([]ModelCatalogChoice, error) {
	return modelcatalog.MergeAIGatewayFreePromotions(curated, payload)
}

func IsModelsDevPreferredProvider(provider string) bool {
	return modelcatalog.IsModelsDevPreferredProvider(provider)
}

func MergePreferredProviderModels(provider string, curated, modelsDev []string) []string {
	return modelcatalog.MergePreferredProviderModels(provider, curated, modelsDev)
}

func ProviderModelCatalogSuggestions(provider string, modelsDev []string) []string {
	return modelcatalog.ProviderModelCatalogSuggestions(provider, modelsDev)
}

func fetchModelCatalogHTTP(ctx context.Context, url string) ([]byte, error) {
	return modelcatalog.FetchModelCatalogHTTP(ctx, url)
}
