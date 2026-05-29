package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const SupportedModelCatalogSchemaVersion = 1

type ModelCatalogSource string

const (
	ModelCatalogSourceDisabled  ModelCatalogSource = "disabled"
	ModelCatalogSourceMemory    ModelCatalogSource = "memory"
	ModelCatalogSourceDisk      ModelCatalogSource = "disk"
	ModelCatalogSourceDiskStale ModelCatalogSource = "disk_stale"
	ModelCatalogSourceNetwork   ModelCatalogSource = "network"
	ModelCatalogSourceOverride  ModelCatalogSource = "override"
	ModelCatalogSourceEmpty     ModelCatalogSource = "empty"
)

type ModelCatalogConfig struct {
	Enabled              bool
	URL                  string
	TTL                  time.Duration
	CachePath            string
	ProviderOverrideURLs map[string]string
}

type ModelCatalogOptions struct {
	Config  ModelCatalogConfig
	Fetcher ModelCatalogFetcher
	Now     func() time.Time
}

type ModelCatalogFetcher func(context.Context, string) ([]byte, error)

type ModelCatalog struct {
	cfg     ModelCatalogConfig
	fetcher ModelCatalogFetcher
	now     func() time.Time

	memory      *ModelCatalogManifest
	memoryMTime time.Time
}

type ModelCatalogManifest struct {
	Version   int                             `json:"version"`
	UpdatedAt string                          `json:"updated_at,omitempty"`
	Metadata  map[string]any                  `json:"metadata,omitempty"`
	Providers map[string]ModelCatalogProvider `json:"providers"`
}

type ModelCatalogProvider struct {
	Metadata map[string]any      `json:"metadata,omitempty"`
	Models   []ModelCatalogModel `json:"models"`
}

type ModelCatalogModel struct {
	ID          string         `json:"id"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type ModelCatalogChoice struct {
	ID          string
	Description string
}

type ModelCatalogEvidence struct {
	Source         ModelCatalogSource
	URL            string
	DegradedReason string
}

type AIGatewayPricing struct {
	Prompt          string
	Completion      string
	InputCacheRead  string
	InputCacheWrite string
}

func NewModelCatalog(opts ModelCatalogOptions) *ModelCatalog {
	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = fetchModelCatalogHTTP
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &ModelCatalog{cfg: opts.Config, fetcher: fetcher, now: now}
}

func (c *ModelCatalog) Get(ctx context.Context, forceRefresh bool) (ModelCatalogManifest, ModelCatalogEvidence, error) {
	if !c.cfg.Enabled {
		return emptyModelCatalogManifest(), ModelCatalogEvidence{
			Source:         ModelCatalogSourceDisabled,
			DegradedReason: "model_catalog_disabled",
		}, nil
	}

	ttl := c.cfg.TTL
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	disk, diskMTime, diskOK := c.readDiskCache()
	diskFresh := diskOK && c.now().Sub(diskMTime) < ttl
	if !forceRefresh && c.memory != nil && diskOK && diskFresh && diskMTime.Equal(c.memoryMTime) {
		return cloneModelCatalogManifest(*c.memory), ModelCatalogEvidence{Source: ModelCatalogSourceMemory, URL: c.cfg.URL}, nil
	}
	if !forceRefresh && diskFresh {
		c.memory = cloneModelCatalogManifestPtr(disk)
		c.memoryMTime = diskMTime
		return disk, ModelCatalogEvidence{Source: ModelCatalogSourceDisk, URL: c.cfg.URL}, nil
	}

	fetched, err := c.fetchManifest(ctx, c.cfg.URL)
	if err == nil {
		writeErr := c.writeDiskCache(fetched)
		if writeErr == nil {
			if diskAfterWrite, mtime, ok := c.readDiskCache(); ok {
				c.memory = cloneModelCatalogManifestPtr(diskAfterWrite)
				c.memoryMTime = mtime
				return diskAfterWrite, ModelCatalogEvidence{Source: ModelCatalogSourceNetwork, URL: c.cfg.URL}, nil
			}
		}
		c.memory = cloneModelCatalogManifestPtr(fetched)
		c.memoryMTime = c.now()
		evidence := ModelCatalogEvidence{Source: ModelCatalogSourceNetwork, URL: c.cfg.URL}
		if writeErr != nil {
			evidence.DegradedReason = "model_catalog_cache_write_failed"
		}
		return fetched, evidence, nil
	}

	if diskOK {
		c.memory = cloneModelCatalogManifestPtr(disk)
		c.memoryMTime = diskMTime
		return disk, ModelCatalogEvidence{
			Source:         ModelCatalogSourceDiskStale,
			URL:            c.cfg.URL,
			DegradedReason: "model_catalog_fetch_failed",
		}, nil
	}
	return emptyModelCatalogManifest(), ModelCatalogEvidence{
		Source:         ModelCatalogSourceEmpty,
		URL:            c.cfg.URL,
		DegradedReason: "model_catalog_fetch_failed",
	}, nil
}

func (c *ModelCatalog) CuratedOpenRouterModels(ctx context.Context) ([]ModelCatalogChoice, ModelCatalogEvidence, error) {
	return c.curatedProviderModels(ctx, "openrouter")
}

func (c *ModelCatalog) CuratedNousModels(ctx context.Context) ([]string, ModelCatalogEvidence, error) {
	choices, evidence, err := c.curatedProviderModels(ctx, "nous")
	if err != nil {
		return nil, evidence, err
	}
	out := make([]string, 0, len(choices))
	for _, choice := range choices {
		out = append(out, choice.ID)
	}
	return out, evidence, nil
}

func (c *ModelCatalog) curatedProviderModels(ctx context.Context, provider string) ([]ModelCatalogChoice, ModelCatalogEvidence, error) {
	provider = normalizeCatalogProvider(provider)
	if overrideURL := strings.TrimSpace(c.cfg.ProviderOverrideURLs[provider]); overrideURL != "" && c.cfg.Enabled {
		if manifest, err := c.fetchManifest(ctx, overrideURL); err == nil {
			if block, ok := manifest.Providers[provider]; ok {
				return choicesFromCatalogProvider(block), ModelCatalogEvidence{
					Source: ModelCatalogSourceOverride,
					URL:    overrideURL,
				}, nil
			}
		}
	}
	manifest, evidence, err := c.Get(ctx, false)
	if err != nil {
		return nil, evidence, err
	}
	block, ok := manifest.Providers[provider]
	if !ok {
		return nil, evidence, nil
	}
	return choicesFromCatalogProvider(block), evidence, nil
}

func ValidateModelCatalogManifest(manifest ModelCatalogManifest) error {
	if manifest.Version < 1 || manifest.Version > SupportedModelCatalogSchemaVersion {
		return fmt.Errorf("model_catalog_invalid_version")
	}
	if manifest.Providers == nil {
		return fmt.Errorf("model_catalog_missing_providers")
	}
	for provider, block := range manifest.Providers {
		if strings.TrimSpace(provider) == "" {
			return fmt.Errorf("model_catalog_invalid_provider")
		}
		if block.Models == nil {
			return fmt.Errorf("model_catalog_invalid_models")
		}
		for _, model := range block.Models {
			if strings.TrimSpace(model.ID) == "" {
				return fmt.Errorf("model_catalog_invalid_model_id")
			}
		}
	}
	return nil
}

func ParseAIGatewayModelPricing(payload []byte) (map[string]AIGatewayPricing, error) {
	var parsed struct {
		Data []struct {
			ID      string         `json:"id"`
			Pricing map[string]any `json:"pricing"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	out := make(map[string]AIGatewayPricing)
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || item.Pricing == nil {
			continue
		}
		out[id] = AIGatewayPricing{
			Prompt:          catalogNumberString(item.Pricing["input"]),
			Completion:      catalogNumberString(item.Pricing["output"]),
			InputCacheRead:  catalogNumberString(firstCatalogValue(item.Pricing, "input_cache_read", "cache_read", "prompt_cache_read")),
			InputCacheWrite: catalogNumberString(firstCatalogValue(item.Pricing, "input_cache_write", "cache_write", "prompt_cache_write")),
		}
	}
	return out, nil
}

func MergeAIGatewayFreePromotions(curated []ModelCatalogChoice, payload []byte) ([]ModelCatalogChoice, error) {
	pricing, err := ParseAIGatewayModelPricing(payload)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(curated)+len(pricing))
	out := make([]ModelCatalogChoice, 0, len(curated)+len(pricing))
	for id, price := range pricing {
		if !isFreeAIGatewayPricing(price) || !isMoonshotOrKimiModel(id) {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(id))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ModelCatalogChoice{ID: id, Description: "recommended"})
	}
	for _, choice := range curated {
		key := strings.ToLower(strings.TrimSpace(choice.ID))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, choice)
	}
	return out, nil
}

func IsModelsDevPreferredProvider(provider string) bool {
	_, ok := modelsDevPreferredProviders[normalizeCatalogProvider(provider)]
	return ok
}

func MergePreferredProviderModels(provider string, curated, modelsDev []string) []string {
	if !IsModelsDevPreferredProvider(provider) || len(modelsDev) == 0 {
		return append([]string(nil), curated...)
	}
	seen := make(map[string]struct{}, len(curated)+len(modelsDev))
	merged := make([]string, 0, len(curated)+len(modelsDev))
	appendModel := func(model string) {
		model = strings.TrimSpace(model)
		key := strings.ToLower(model)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		merged = append(merged, model)
	}
	for _, model := range modelsDev {
		appendModel(model)
	}
	for _, model := range curated {
		appendModel(model)
	}
	return merged
}

func ProviderModelCatalogSuggestions(provider string, modelsDev []string) []string {
	provider = normalizeCatalogProvider(provider)
	curated := providerModelCatalogFloor[provider]
	return MergePreferredProviderModels(provider, curated, modelsDev)
}

func (c *ModelCatalog) fetchManifest(ctx context.Context, url string) (ModelCatalogManifest, error) {
	raw, err := c.fetcher(ctx, url)
	if err != nil {
		return ModelCatalogManifest{}, err
	}
	var manifest ModelCatalogManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ModelCatalogManifest{}, err
	}
	if err := ValidateModelCatalogManifest(manifest); err != nil {
		return ModelCatalogManifest{}, err
	}
	return normalizeModelCatalogManifest(manifest), nil
}

func (c *ModelCatalog) readDiskCache() (ModelCatalogManifest, time.Time, bool) {
	if strings.TrimSpace(c.cfg.CachePath) == "" {
		return ModelCatalogManifest{}, time.Time{}, false
	}
	info, err := os.Stat(c.cfg.CachePath)
	if err != nil {
		return ModelCatalogManifest{}, time.Time{}, false
	}
	raw, err := os.ReadFile(c.cfg.CachePath)
	if err != nil {
		return ModelCatalogManifest{}, time.Time{}, false
	}
	var manifest ModelCatalogManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return ModelCatalogManifest{}, time.Time{}, false
	}
	if err := ValidateModelCatalogManifest(manifest); err != nil {
		return ModelCatalogManifest{}, time.Time{}, false
	}
	return normalizeModelCatalogManifest(manifest), info.ModTime(), true
}

func (c *ModelCatalog) writeDiskCache(manifest ModelCatalogManifest) error {
	if strings.TrimSpace(c.cfg.CachePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.cfg.CachePath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp, err := os.CreateTemp(filepath.Dir(c.cfg.CachePath), filepath.Base(c.cfg.CachePath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, c.cfg.CachePath); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Chmod(c.cfg.CachePath, 0o600)
}

func fetchModelCatalogHTTP(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "gormes-agent/model-catalog")
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("model_catalog_http_status_%d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func choicesFromCatalogProvider(block ModelCatalogProvider) []ModelCatalogChoice {
	out := make([]ModelCatalogChoice, 0, len(block.Models))
	for _, model := range block.Models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		out = append(out, ModelCatalogChoice{ID: id, Description: model.Description})
	}
	return out
}

func normalizeModelCatalogManifest(manifest ModelCatalogManifest) ModelCatalogManifest {
	if manifest.Providers == nil {
		manifest.Providers = map[string]ModelCatalogProvider{}
	}
	normalized := ModelCatalogManifest{
		Version:   manifest.Version,
		UpdatedAt: strings.TrimSpace(manifest.UpdatedAt),
		Metadata:  manifest.Metadata,
		Providers: make(map[string]ModelCatalogProvider, len(manifest.Providers)),
	}
	for provider, block := range manifest.Providers {
		provider = normalizeCatalogProvider(provider)
		models := make([]ModelCatalogModel, 0, len(block.Models))
		for _, model := range block.Models {
			model.ID = strings.TrimSpace(model.ID)
			if model.ID == "" {
				continue
			}
			model.Description = strings.TrimSpace(model.Description)
			models = append(models, model)
		}
		normalized.Providers[provider] = ModelCatalogProvider{Metadata: block.Metadata, Models: models}
	}
	return normalized
}

func emptyModelCatalogManifest() ModelCatalogManifest {
	return ModelCatalogManifest{Providers: map[string]ModelCatalogProvider{}}
}

func cloneModelCatalogManifestPtr(manifest ModelCatalogManifest) *ModelCatalogManifest {
	clone := cloneModelCatalogManifest(manifest)
	return &clone
}

func cloneModelCatalogManifest(manifest ModelCatalogManifest) ModelCatalogManifest {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return manifest
	}
	var clone ModelCatalogManifest
	if err := json.Unmarshal(raw, &clone); err != nil {
		return manifest
	}
	return clone
}

func catalogNumberString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	default:
		return fmt.Sprint(typed)
	}
}

func firstCatalogValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func isFreeAIGatewayPricing(pricing AIGatewayPricing) bool {
	return catalogPriceIsZero(pricing.Prompt) && catalogPriceIsZero(pricing.Completion)
}

func catalogPriceIsZero(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed == 0
}

func isMoonshotOrKimiModel(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "moonshot") || strings.Contains(id, "kimi")
}

func normalizeCatalogProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	switch provider {
	case "google", "google-ai-studio", "google-gemini":
		return "gemini"
	case "open-router", "openrouter-free", "or":
		return "openrouter"
	default:
		return provider
	}
}

var modelsDevPreferredProviders = map[string]struct{}{
	"opencode-go":  {},
	"opencode-zen": {},
	"deepseek":     {},
	"kilocode":     {},
	"fireworks":    {},
	"mistral":      {},
	"togetherai":   {},
	"cohere":       {},
	"perplexity":   {},
	"groq":         {},
	"nvidia":       {},
	"huggingface":  {},
	"zai":          {},
	"gemini":       {},
	"google":       {},
}

var providerModelCatalogFloor = map[string][]string{
	"openrouter": {
		"deepseek/deepseek-chat-v3-0324:free",
		"deepseek/deepseek-r1:free",
		"meta-llama/llama-4-maverick:free",
		"qwen/qwen3-235b-a22b:free",
		"anthropic/claude-opus-4.7",
		"anthropic/claude-opus-4.6",
		"anthropic/claude-sonnet-4.6",
		"moonshotai/kimi-k2.6",
		"openrouter/pareto-code",
		"qwen/qwen3.6-plus",
		"anthropic/claude-haiku-4.5",
		"openai/gpt-5.5",
		"openai/gpt-5.5-pro",
		"openai/gpt-5.4-mini",
		"openai/gpt-5.4-nano",
		"openai/gpt-5.3-codex",
		"xiaomi/mimo-v2.5-pro",
		"tencent/hy3-preview",
		"google/gemini-3-pro-image-preview",
		"google/gemini-3-flash-preview",
		"google/gemini-3.1-pro-preview",
		"google/gemini-3.1-flash-lite-preview",
		"qwen/qwen3.6-35b-a3b",
		"stepfun/step-3.5-flash",
		"minimax/minimax-m2.7",
		"z-ai/glm-5.1",
		"x-ai/grok-4.20",
		"x-ai/grok-4.3",
		"nvidia/nemotron-3-super-120b-a12b",
		"deepseek/deepseek-v4-pro",
		"openrouter/elephant-alpha",
		"openrouter/owl-alpha",
		"tencent/hy3-preview:free",
		"nvidia/nemotron-3-super-120b-a12b:free",
		"inclusionai/ring-2.6-1t:free",
	},
	"gemini": {
		"gemini-2.5-flash",
	},
	"openai-codex": {
		"gpt-5.5",
		"gpt-5.4-mini",
		"gpt-5.4",
		"gpt-5.3-codex",
		"gpt-5.3-codex-spark",
		"gpt-5.2-codex",
		"gpt-5.1-codex-max",
		"gpt-5.1-codex-mini",
	},
	"opencode-go": {
		"kimi-k2.6",
		"kimi-k2.5",
		"glm-5.1",
		"glm-5",
		"mimo-v2-pro",
		"mimo-v2-omni",
		"minimax-m2.7",
		"minimax-m2.5",
	},
	"opencode-zen": {
		"kimi-k2.6",
		"glm-5.1",
		"claude-opus-4-7",
	},
	"groq": {
		"llama-3.3-70b-versatile",
	},
}
