package llm

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const OpenRouterDefaultBaseURL = "https://openrouter.ai/api/v1"

type OpenRouterRuntimeSource string

const (
	OpenRouterRuntimeSourceExplicit   OpenRouterRuntimeSource = "explicit"
	OpenRouterRuntimeSourceEnvDefault OpenRouterRuntimeSource = "env_default"
)

type OpenRouterRuntimeRequest struct {
	Provider  string
	BaseURL   string
	APIKey    string
	LookupEnv func(string) (string, bool)
}

type OpenRouterRuntime struct {
	Provider          string
	BaseURL           string
	APIKey            string
	Source            OpenRouterRuntimeSource
	IsOpenRouterRoute bool
	MissingAPIKey     bool
}

func ResolveOpenRouterRuntime(req OpenRouterRuntimeRequest) OpenRouterRuntime {
	lookupEnv := req.LookupEnv
	if lookupEnv == nil {
		lookupEnv = os.LookupEnv
	}
	requestedProvider := normalizeOpenRouterProvider(req.Provider)
	if requestedProvider == "" {
		requestedProvider = "openrouter"
	}

	baseURL := cleanOpenRouterBaseURL(req.BaseURL)
	source := OpenRouterRuntimeSourceExplicit
	if baseURL == "" {
		source = OpenRouterRuntimeSourceEnvDefault
		baseURL = cleanOpenRouterBaseURL(lookupEnvValue(lookupEnv, "OPENROUTER_BASE_URL"))
	}
	if baseURL == "" {
		baseURL = OpenRouterDefaultBaseURL
	}

	provider := "openrouter"
	if requestedProvider == "custom" {
		provider = "custom"
	}
	isOpenRouterRoute := provider == "openrouter" || IsOpenRouterBaseURL(baseURL)

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		source = OpenRouterRuntimeSourceEnvDefault
		if isOpenRouterRoute {
			apiKey = firstOpenRouterRuntimeValue(
				lookupEnvValue(lookupEnv, "OPENROUTER_API_KEY"),
				lookupEnvValue(lookupEnv, "OPENAI_API_KEY"),
			)
		} else {
			apiKey = firstOpenRouterRuntimeValue(
				lookupEnvValue(lookupEnv, "OPENAI_API_KEY"),
				lookupEnvValue(lookupEnv, "OPENROUTER_API_KEY"),
			)
		}
	}

	return OpenRouterRuntime{
		Provider:          provider,
		BaseURL:           baseURL,
		APIKey:            apiKey,
		Source:            source,
		IsOpenRouterRoute: isOpenRouterRoute,
		MissingAPIKey:     isOpenRouterRoute && apiKey == "",
	}
}

func IsOpenRouterBaseURL(baseURL string) bool {
	return baseURLHostMatches(baseURL, "openrouter.ai")
}

func IsOpenRouterRoute(provider, baseURL string) bool {
	provider = normalizeOpenRouterProvider(provider)
	return provider == "openrouter" || IsOpenRouterBaseURL(baseURL)
}

func ApplyOpenRouterAttributionHeaders(req *http.Request, provider, baseURL string) {
	if req == nil || !IsOpenRouterRoute(provider, baseURL) {
		return
	}
	for key, value := range OpenRouterAttributionHeaders() {
		req.Header.Set(key, value)
	}
}

func ApplyOpenRouterGrokPromptCacheAffinityHeader(req *http.Request, provider, baseURL, model, sessionID string) {
	if req == nil || !IsOpenRouterRoute(provider, baseURL) {
		return
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" || !openRouterModelIsGrok(model) {
		return
	}
	req.Header.Set("x-grok-conv-id", sessionID)
}

func OpenRouterAttributionHeaders() map[string]string {
	return map[string]string{
		"HTTP-Referer": "https://gormes.ai",
		// X-Title is the canonical attribution header OpenRouter's dashboard reads;
		// X-OpenRouter-Title was not recognized there. Mirrors Hermes 6430d6756.
		"X-Title":                 "Gormes Agent",
		"X-OpenRouter-Categories": "productivity,cli-agent",
	}
}

func openRouterModelIsGrok(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "x-ai/grok-") || strings.HasPrefix(model, "xai/grok-")
}

// openRouterAdaptiveAnthropicVerbosity returns the verbosity string that
// OpenRouter should receive for Claude 4.6+ models (adaptive/mandatory
// thinking). These models ignore reasoning.effort and use the top-level
// verbosity field instead. Returns "" when no verbosity should be emitted
// (model is not adaptive, or effort is none/empty).
//
// Mirrors Hermes fix(openrouter): route reasoning_effort to verbosity for
// adaptive Anthropic models (183d86b3e).
func openRouterAdaptiveAnthropicVerbosity(provider, baseURL, model string, effort *ReasoningEffort) string {
	if !IsOpenRouterRoute(provider, baseURL) {
		return ""
	}
	m := strings.ToLower(strings.TrimSpace(model))
	// Must be an Anthropic model.
	isAnthropic := strings.HasPrefix(m, "anthropic/") || strings.HasPrefix(m, "claude") || strings.Contains(m, "claude")
	if !isAnthropic {
		return ""
	}
	// Older models with optional reasoning — not adaptive.
	optional := []string{
		"claude-3",
		"claude-opus-4-0", "claude-opus-4.0", "claude-opus-4-1", "claude-opus-4.1",
		"claude-sonnet-4-0", "claude-sonnet-4.0",
		"claude-opus-4-2025", "claude-sonnet-4-2025",
		"claude-opus-4-5", "claude-opus-4.5",
		"claude-sonnet-4-5", "claude-sonnet-4.5",
		"claude-haiku-4-5", "claude-haiku-4.5",
	}
	for _, sub := range optional {
		if strings.Contains(m, sub) {
			return ""
		}
	}
	// Adaptive model confirmed. Map effort to Anthropic verbosity tiers.
	if effort == nil {
		return ""
	}
	switch *effort {
	case ReasoningEffortNone:
		return "" // disabled — can't truly disable adaptive thinking, so omit verbosity
	case ReasoningEffortMinimal, ReasoningEffortLow:
		return "low"
	case ReasoningEffortMedium:
		return "medium"
	case ReasoningEffortHigh, ReasoningEffortXHigh:
		return "high"
	default:
		return ""
	}
}

func buildOpenRouterParetoExtraBody(provider, baseURL, model, score string) map[string]any {
	if !IsOpenRouterRoute(provider, baseURL) || strings.TrimSpace(model) != "openrouter/pareto-code" {
		return nil
	}
	normalized, ok := normalizeOpenRouterParetoScore(score)
	if !ok {
		return nil
	}
	return map[string]any{
		"plugins": []map[string]any{{
			"id":               "pareto-router",
			"min_coding_score": normalized,
		}},
	}
}

func normalizeOpenRouterParetoScore(raw string) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	score, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(score) || math.IsInf(score, 0) || score < 0 || score > 1 {
		return 0, false
	}
	return score, true
}

func ParseOpenRouterModelRegistry(data []byte, version string) ([]ModelRegistryEntry, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var payload struct {
		Data []openRouterModelPayload `json:"data"`
	}
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	entries := make([]ModelRegistryEntry, 0, len(payload.Data))
	for _, item := range payload.Data {
		model := strings.TrimSpace(item.ID)
		if model == "" {
			continue
		}
		entries = append(entries, ModelRegistryEntry{
			Provider:         "openrouter",
			Model:            model,
			ProviderFamily:   openRouterProviderFamily(model),
			ModelFamily:      openRouterModelFamily(model),
			RawContextWindow: openRouterJSONInt(item.ContextLength),
			MaxOutputTokens:  openRouterJSONInt(item.TopProvider.MaxCompletionTokens),
			Pricing: knownModelPricing(
				openRouterPricePerMillion(item.Pricing["prompt"]),
				openRouterPricePerMillion(item.Pricing["completion"]),
				openRouterPricePerMillion(firstOpenRouterPricingField(item.Pricing, "input_cache_read", "cache_read", "prompt_cache_read")),
				openRouterPricePerMillion(firstOpenRouterPricingField(item.Pricing, "input_cache_write", "cache_write", "prompt_cache_write")),
				ModelPricingSourceProviderModelsAPI,
				version,
			),
			Capabilities: openRouterCapabilities(item.SupportedParameters),
		})
	}
	return entries, nil
}

type openRouterModelPayload struct {
	ID                  string                `json:"id"`
	ContextLength       any                   `json:"context_length"`
	TopProvider         openRouterTopProvider `json:"top_provider"`
	Pricing             map[string]any        `json:"pricing"`
	SupportedParameters []string              `json:"supported_parameters"`
}

type openRouterTopProvider struct {
	MaxCompletionTokens any `json:"max_completion_tokens"`
}

func normalizeOpenRouterProvider(provider string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
}

func cleanOpenRouterBaseURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/")
}

func lookupEnvValue(lookup func(string) (string, bool), name string) string {
	value, ok := lookup(name)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstOpenRouterRuntimeValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func openRouterProviderFamily(model string) string {
	model = strings.TrimSpace(model)
	if before, _, ok := strings.Cut(model, "/"); ok && before != "" {
		return before
	}
	return "openrouter"
}

func openRouterModelFamily(model string) string {
	model = strings.TrimSpace(model)
	if _, after, ok := strings.Cut(model, "/"); ok {
		model = after
	}
	if before, _, ok := strings.Cut(model, ":"); ok {
		model = before
	}
	return model
}

func firstOpenRouterPricingField(pricing map[string]any, names ...string) any {
	for _, name := range names {
		if value, ok := pricing[name]; ok {
			return value
		}
	}
	return nil
}

func openRouterPricePerMillion(value any) float64 {
	price := openRouterJSONFloat(value)
	if price <= 0 {
		return 0
	}
	return price * 1_000_000
}

func openRouterJSONInt(value any) int {
	switch v := value.(type) {
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
		f, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return int(f)
		}
	case float64:
		return int(v)
	case int:
		return v
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return int(f)
		}
	}
	return 0
}

func openRouterJSONFloat(value any) float64 {
	switch v := value.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(v.String(), 64)
		if err == nil {
			return f
		}
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func openRouterCapabilities(params []string) ModelCapabilityFlags {
	seen := make(map[string]bool, len(params))
	for _, param := range params {
		seen[strings.ToLower(strings.TrimSpace(param))] = true
	}
	tools := ModelCapabilityUnsupported
	if seen["tools"] {
		tools = ModelCapabilitySupported
	}
	structured := ModelCapabilityUnsupported
	if seen["response_format"] || seen["structured_outputs"] {
		structured = ModelCapabilitySupported
	}
	return knownModelCapabilities(
		tools,
		ModelCapabilityUnknown,
		ModelCapabilityUnknown,
		ModelCapabilityUnknown,
		ModelCapabilityUnknown,
		structured,
		ModelCapabilityUnknown,
	)
}
