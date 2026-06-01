package modelcatalog

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/identity"
)

type ModelFactStatus string

const (
	ModelFactKnown   ModelFactStatus = "known"
	ModelFactUnknown ModelFactStatus = "unknown"
)

type ModelCapabilityFlag string

const (
	ModelCapabilitySupported   ModelCapabilityFlag = "supported"
	ModelCapabilityUnsupported ModelCapabilityFlag = "unsupported"
	ModelCapabilityUnknown     ModelCapabilityFlag = "unknown"
)

type ModelPricingSource string

const (
	ModelPricingSourceNone                 ModelPricingSource = "none"
	ModelPricingSourceOfficialDocsSnapshot ModelPricingSource = "official_docs_snapshot"
	ModelPricingSourceModelsDevSnapshot    ModelPricingSource = "models_dev_snapshot"
	ModelPricingSourceProviderModelsAPI    ModelPricingSource = "provider_models_api"
)

type ModelRegistrySource string

const (
	ModelRegistrySourceEmbedded ModelRegistrySource = "embedded"
	ModelRegistrySourceTestdata ModelRegistrySource = "testdata"
)

type ModelRegistryFreshness string

const (
	ModelRegistryFreshnessCurrent ModelRegistryFreshness = "current"
	ModelRegistryFreshnessStale   ModelRegistryFreshness = "stale"
)

type ModelPricing struct {
	Status                  ModelFactStatus
	InputUSDPerMillion      float64
	OutputUSDPerMillion     float64
	CacheReadUSDPerMillion  float64
	CacheWriteUSDPerMillion float64
	Source                  ModelPricingSource
	Version                 string
}

func (p ModelPricing) Known() bool {
	return p.Status == ModelFactKnown
}

type ModelCapabilityFlags struct {
	Status           ModelFactStatus
	Tools            ModelCapabilityFlag
	Vision           ModelCapabilityFlag
	Reasoning        ModelCapabilityFlag
	PDF              ModelCapabilityFlag
	AudioInput       ModelCapabilityFlag
	StructuredOutput ModelCapabilityFlag
	OpenWeights      ModelCapabilityFlag
}

func (c ModelCapabilityFlags) Known() bool {
	return c.Status == ModelFactKnown
}

type ModelRegistrySnapshot struct {
	Source    ModelRegistrySource
	Freshness ModelRegistryFreshness
	Version   string
	Reason    string
}

type ModelRegistryQuery struct {
	Provider string
	Model    string
}

type ModelRegistryKey struct {
	Provider string
	Model    string
}

type ModelRegistryEntry struct {
	Provider         string
	Model            string
	ProviderFamily   string
	ModelFamily      string
	RawContextWindow int
	MaxOutputTokens  int
	Pricing          ModelPricing
	Capabilities     ModelCapabilityFlags
}

const OllamaCloudProviderID = "ollama-cloud"

func NormalizeProviderModelID(provider, model string) string {
	model = strings.TrimSpace(model)
	if normalizeModelRegistryProvider(provider) == OllamaCloudProviderID {
		return NormalizeOllamaCloudModelID(model)
	}
	return model
}

func NormalizeOllamaCloudModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	lower := strings.ToLower(modelID)
	for _, suffix := range []string{":cloud", "-cloud"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimSpace(modelID[:len(modelID)-len(suffix)])
		}
	}
	return modelID
}

func MergeOllamaCloudModelEntries(live, modelsDev []ModelRegistryEntry) []ModelRegistryEntry {
	seen := make(map[string]struct{}, len(live)+len(modelsDev))
	merged := make([]ModelRegistryEntry, 0, len(live)+len(modelsDev))
	appendEntry := func(entry ModelRegistryEntry) {
		if strings.TrimSpace(entry.Provider) == "" {
			entry.Provider = OllamaCloudProviderID
		}
		entry.Provider = normalizeModelRegistryProvider(entry.Provider)
		if entry.Provider != OllamaCloudProviderID {
			return
		}
		entry.Model = NormalizeOllamaCloudModelID(entry.Model)
		entry = normalizeModelRegistryEntry(entry)
		if entry.Model == "" {
			return
		}
		if _, ok := seen[entry.Model]; ok {
			return
		}
		seen[entry.Model] = struct{}{}
		merged = append(merged, entry)
	}
	for _, entry := range live {
		appendEntry(entry)
	}
	for _, entry := range modelsDev {
		appendEntry(entry)
	}
	return merged
}

type ModelMetadataResult struct {
	Found bool
	ModelRegistryEntry
	Registry ModelRegistrySnapshot
}

type ModelRegistry struct {
	snapshot ModelRegistrySnapshot
	entries  map[ModelRegistryKey]ModelRegistryEntry
}

func NewStaticModelRegistry(snapshot ModelRegistrySnapshot, entries []ModelRegistryEntry) ModelRegistry {
	registry := ModelRegistry{
		snapshot: normalizeModelRegistrySnapshot(snapshot),
		entries:  make(map[ModelRegistryKey]ModelRegistryEntry, len(entries)),
	}
	for _, entry := range entries {
		entry = normalizeModelRegistryEntry(entry)
		key := ModelRegistryKey{Provider: entry.Provider, Model: entry.Model}
		if key.Provider == "" || key.Model == "" {
			continue
		}
		registry.entries[key] = entry
	}
	return registry
}

func DefaultModelRegistry() ModelRegistry {
	return defaultModelRegistry
}

func (r ModelRegistry) IsZero() bool {
	return r.entries == nil
}

func LookupModelMetadata(query ModelRegistryQuery) ModelMetadataResult {
	return DefaultModelRegistry().Lookup(query)
}

func (r ModelRegistry) Entries() []ModelRegistryEntry {
	entries := make([]ModelRegistryEntry, 0, len(r.entries))
	for _, entry := range r.entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Provider == entries[j].Provider {
			return entries[i].Model < entries[j].Model
		}
		return entries[i].Provider < entries[j].Provider
	})
	return entries
}

func (r ModelRegistry) Lookup(query ModelRegistryQuery) ModelMetadataResult {
	result := ModelMetadataResult{
		Registry: r.Snapshot(),
		Found:    false,
		ModelRegistryEntry: ModelRegistryEntry{
			Pricing:      unknownModelPricing(),
			Capabilities: unknownModelCapabilities(),
		},
	}
	key := ModelRegistryKey{
		Provider: normalizeModelRegistryProvider(query.Provider),
		Model:    normalizeModelRegistryText(query.Model),
	}
	if key.Provider == "" || key.Model == "" {
		return result
	}
	entry, ok := r.entries[key]
	if !ok {
		entry, ok = r.lookupStaticAlias(key)
		if !ok {
			return result
		}
	}
	result.Found = true
	result.ModelRegistryEntry = entry
	return result
}

type staticModelAlias struct {
	provider string
	prefix   string
}

func (r ModelRegistry) lookupStaticAlias(key ModelRegistryKey) (ModelRegistryEntry, bool) {
	alias, ok := staticModelAliases[key.Model]
	if !ok {
		return ModelRegistryEntry{}, false
	}
	provider := key.Provider
	if provider == "" {
		provider = alias.provider
	}
	if provider == "" {
		return ModelRegistryEntry{}, false
	}

	var candidates []ModelRegistryEntry
	for _, entry := range r.entries {
		if entry.Provider != provider {
			continue
		}
		if strings.HasPrefix(entry.Model, alias.prefix) {
			candidates = append(candidates, entry)
		}
	}
	if len(candidates) == 0 {
		return ModelRegistryEntry{}, false
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Model > candidates[j].Model
	})
	return candidates[0], true
}

func (r ModelRegistry) Snapshot() ModelRegistrySnapshot {
	return normalizeModelRegistrySnapshot(r.snapshot)
}

func normalizeModelRegistrySnapshot(snapshot ModelRegistrySnapshot) ModelRegistrySnapshot {
	if snapshot.Source == "" {
		snapshot.Source = ModelRegistrySourceEmbedded
	}
	if snapshot.Freshness == "" {
		snapshot.Freshness = ModelRegistryFreshnessCurrent
	}
	return snapshot
}

func normalizeModelRegistryEntry(entry ModelRegistryEntry) ModelRegistryEntry {
	entry.Provider = normalizeModelRegistryProvider(entry.Provider)
	entry.Model = normalizeModelRegistryText(entry.Model)
	if entry.ProviderFamily == "" {
		entry.ProviderFamily = entry.Provider
	}
	entry.Pricing = normalizeModelPricing(entry.Pricing)
	entry.Capabilities = normalizeModelCapabilities(entry.Capabilities)
	return entry
}

func normalizeModelRegistryProvider(provider string) string {
	return identity.Provider(provider)
}

func normalizeModelRegistryText(value string) string {
	return identity.Text(value)
}

func normalizeModelPricing(pricing ModelPricing) ModelPricing {
	if pricing.Status == "" {
		pricing.Status = ModelFactUnknown
	}
	if pricing.Source == "" {
		pricing.Source = ModelPricingSourceNone
	}
	return pricing
}

func normalizeModelCapabilities(capabilities ModelCapabilityFlags) ModelCapabilityFlags {
	if capabilities.Status == "" {
		capabilities.Status = ModelFactUnknown
	}
	capabilities.Tools = normalizeModelCapabilityFlag(capabilities.Tools)
	capabilities.Vision = normalizeModelCapabilityFlag(capabilities.Vision)
	capabilities.Reasoning = normalizeModelCapabilityFlag(capabilities.Reasoning)
	capabilities.PDF = normalizeModelCapabilityFlag(capabilities.PDF)
	capabilities.AudioInput = normalizeModelCapabilityFlag(capabilities.AudioInput)
	capabilities.StructuredOutput = normalizeModelCapabilityFlag(capabilities.StructuredOutput)
	capabilities.OpenWeights = normalizeModelCapabilityFlag(capabilities.OpenWeights)
	return capabilities
}

func normalizeModelCapabilityFlag(flag ModelCapabilityFlag) ModelCapabilityFlag {
	if flag == "" {
		return ModelCapabilityUnknown
	}
	return flag
}

func unknownModelPricing() ModelPricing {
	return normalizeModelPricing(ModelPricing{Status: ModelFactUnknown})
}

func unknownModelCapabilities() ModelCapabilityFlags {
	return normalizeModelCapabilities(ModelCapabilityFlags{Status: ModelFactUnknown})
}

func knownModelPricing(input, output, cacheRead, cacheWrite float64, source ModelPricingSource, version string) ModelPricing {
	return ModelPricing{
		Status:                  ModelFactKnown,
		InputUSDPerMillion:      input,
		OutputUSDPerMillion:     output,
		CacheReadUSDPerMillion:  cacheRead,
		CacheWriteUSDPerMillion: cacheWrite,
		Source:                  source,
		Version:                 version,
	}
}

func knownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights ModelCapabilityFlag) ModelCapabilityFlags {
	return normalizeModelCapabilities(ModelCapabilityFlags{
		Status:           ModelFactKnown,
		Tools:            tools,
		Vision:           vision,
		Reasoning:        reasoning,
		PDF:              pdf,
		AudioInput:       audioInput,
		StructuredOutput: structuredOutput,
		OpenWeights:      openWeights,
	})
}

var staticModelAliases = map[string]staticModelAlias{
	"sonnet": {provider: "anthropic", prefix: "claude-sonnet"},
	"opus":   {provider: "anthropic", prefix: "claude-opus"},
	"haiku":  {provider: "anthropic", prefix: "claude-haiku"},
	"claude": {provider: "anthropic", prefix: "claude"},
}

var defaultModelRegistry = NewStaticModelRegistry(ModelRegistrySnapshot{
	Source:    ModelRegistrySourceEmbedded,
	Freshness: ModelRegistryFreshnessCurrent,
	Version:   "models.dev-fixture-2026-04-25",
}, []ModelRegistryEntry{
	{
		Provider:         "openai",
		Model:            "gpt-4o-mini",
		ProviderFamily:   "openai",
		ModelFamily:      "gpt-4o",
		RawContextWindow: 128_000,
		MaxOutputTokens:  16_384,
		Pricing: knownModelPricing(
			0.15,
			0.60,
			0.075,
			0,
			ModelPricingSourceOfficialDocsSnapshot,
			"openai-pricing-2026-03-16",
		),
		Capabilities: knownModelCapabilities(
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
		),
	},
	{
		Provider:         "anthropic",
		Model:            "claude-opus-4-20250514",
		ProviderFamily:   "anthropic",
		ModelFamily:      "claude-opus-4",
		RawContextWindow: 200_000,
		MaxOutputTokens:  32_000,
		Pricing: knownModelPricing(
			15.00,
			75.00,
			1.50,
			18.75,
			ModelPricingSourceOfficialDocsSnapshot,
			"anthropic-prompt-caching-2026-03-16",
		),
		Capabilities: knownModelCapabilities(
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
		),
	},
	{
		Provider:         "anthropic",
		Model:            "claude-sonnet-4-20250514",
		ProviderFamily:   "anthropic",
		ModelFamily:      "claude-sonnet-4",
		RawContextWindow: 200_000,
		MaxOutputTokens:  32_000,
		Pricing:          unknownModelPricing(),
		Capabilities: knownModelCapabilities(
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
		),
	},
	{
		Provider:         "openai-codex",
		Model:            "gpt-5.5",
		ProviderFamily:   "openai",
		ModelFamily:      "gpt-5",
		RawContextWindow: 1_050_000,
		MaxOutputTokens:  128_000,
		Pricing:          unknownModelPricing(),
		Capabilities: knownModelCapabilities(
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
		),
	},
	{
		Provider:         "openai-codex",
		Model:            "gpt-5.3-codex-spark",
		ProviderFamily:   "openai",
		ModelFamily:      "gpt-5.3-codex",
		RawContextWindow: 128_000,
		MaxOutputTokens:  32_000,
		Pricing:          unknownModelPricing(),
		Capabilities: knownModelCapabilities(
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
			ModelCapabilityUnsupported,
			ModelCapabilitySupported,
			ModelCapabilityUnsupported,
		),
	},
})

func UnknownModelPricing() ModelPricing {
	return unknownModelPricing()
}

func UnknownModelCapabilities() ModelCapabilityFlags {
	return unknownModelCapabilities()
}

func KnownModelPricing(input, output, cacheRead, cacheWrite float64, source ModelPricingSource, version string) ModelPricing {
	return knownModelPricing(input, output, cacheRead, cacheWrite, source, version)
}

func KnownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights ModelCapabilityFlag) ModelCapabilityFlags {
	return knownModelCapabilities(tools, vision, reasoning, pdf, audioInput, structuredOutput, openWeights)
}
