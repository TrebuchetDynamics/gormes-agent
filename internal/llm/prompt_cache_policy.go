package llm

import "strings"

// PromptCacheLayout describes the provider wire shape that may carry Anthropic
// cache_control hints. Unsupported providers must strip the hints before
// request serialization.
type PromptCacheLayout string

const (
	PromptCacheLayoutUnsupported     PromptCacheLayout = "unsupported"
	PromptCacheLayoutNativeAnthropic PromptCacheLayout = "native_anthropic"
	PromptCacheLayoutEnvelope        PromptCacheLayout = "envelope"
	defaultPromptCacheControlType                      = "ephemeral"
	defaultPromptCacheTTL                              = "1h"
)

// PromptCachePolicyInput is the provider/model tuple used to decide whether a
// request may carry Anthropic-compatible cache_control markers.
type PromptCachePolicyInput struct {
	Provider string
	BaseURL  string
	APIMode  string
	Model    string
}

// PromptCachePolicy is the bounded cache-control decision shared by native
// Anthropic and OpenAI-compatible request serializers.
type PromptCachePolicy struct {
	ShouldCache bool
	Layout      PromptCacheLayout
	TTL         string
	Reason      string
}

// PromptCachePolicyFor ports Hermes' _anthropic_prompt_cache_policy provider
// matrix without reading live provider config or credentials.
func PromptCachePolicyFor(input PromptCachePolicyInput) PromptCachePolicy {
	provider := strings.ToLower(strings.TrimSpace(input.Provider))
	providerDash := strings.ReplaceAll(provider, "_", "-")
	baseURL := strings.TrimSpace(input.BaseURL)
	apiMode := strings.ToLower(strings.TrimSpace(input.APIMode))
	model := strings.ToLower(strings.TrimSpace(input.Model))
	isClaude := strings.Contains(model, "claude")
	isAnthropicWire := apiMode == "anthropic_messages"
	isNativeAnthropic := isAnthropicWire && (provider == "anthropic" || baseURLHostname(baseURL) == "api.anthropic.com")
	if isNativeAnthropic {
		return supportedPromptCachePolicy(PromptCacheLayoutNativeAnthropic, "prompt_cache_supported: native Anthropic messages cache_control")
	}
	if baseURLHostMatches(baseURL, "openrouter.ai") && isClaude {
		return supportedPromptCachePolicy(PromptCacheLayoutEnvelope, "prompt_cache_supported: OpenRouter Claude cache_control envelope")
	}
	// Nous Portal proxies to OpenRouter; Claude and Qwen both get envelope-layout
	// caching. Mirrors Hermes fix(cache): route Nous Portal Qwen through Portal-Claude
	// cache pathway (7993e03c0).
	isNousPortal := baseURLHostMatches(baseURL, "nousresearch.com") || providerDash == "nous"
	if isNousPortal && (isClaude || strings.Contains(model, "qwen")) {
		return supportedPromptCachePolicy(PromptCacheLayoutEnvelope, "prompt_cache_supported: Nous Portal Claude/Qwen cache_control envelope")
	}
	if isAnthropicWire && isClaude {
		return supportedPromptCachePolicy(PromptCacheLayoutNativeAnthropic, "prompt_cache_supported: third-party Anthropic messages Claude cache_control")
	}
	if isAnthropicWire && (providerDash == "minimax" || providerDash == "minimax-cn" || baseURLHostMatches(baseURL, "api.minimax.io") || baseURLHostMatches(baseURL, "api.minimaxi.com")) {
		return supportedPromptCachePolicy(PromptCacheLayoutNativeAnthropic, "prompt_cache_supported: MiniMax Anthropic-compatible cache_control")
	}
	if strings.Contains(model, "qwen") {
		switch providerDash {
		case "opencode", "opencode-zen", "opencode-go", "alibaba":
			return supportedPromptCachePolicy(PromptCacheLayoutEnvelope, "prompt_cache_supported: Qwen/Alibaba OpenAI-wire cache_control envelope")
		}
	}
	return PromptCachePolicy{ShouldCache: false, Layout: PromptCacheLayoutUnsupported, TTL: defaultPromptCacheTTL, Reason: "prompt_cache_stripped: provider/model does not advertise cache_control support"}
}

func supportedPromptCachePolicy(layout PromptCacheLayout, reason string) PromptCachePolicy {
	return PromptCachePolicy{ShouldCache: true, Layout: layout, TTL: defaultPromptCacheTTL, Reason: reason}
}

func openAICompatiblePromptCacheStatus(provider, baseURL string) CapabilityStatus {
	providerLower := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(provider)), "_", "-")
	if baseURLHostMatches(baseURL, "openrouter.ai") {
		return CapabilityStatus{Available: true, Reason: "prompt_cache_supported: OpenRouter Claude requests serialize cache_control when model policy allows it"}
	}
	if baseURLHostMatches(baseURL, "nousresearch.com") || providerLower == "nous" {
		return CapabilityStatus{Available: true, Reason: "prompt_cache_supported: Nous Portal requests serialize cache_control when model policy allows it"}
	}
	switch providerLower {
	case "opencode", "opencode-zen", "opencode-go", "alibaba":
		return CapabilityStatus{Available: true, Reason: "prompt_cache_supported: Qwen/Alibaba requests serialize cache_control when model policy allows it"}
	}
	return unavailableCapability("cache_control stripped: prompt_cache_stripped unsupported OpenAI-compatible request mapping omits cache_control")
}

// markToolsForLongLivedCache marks the last tool in the list with a 1h
// cache_control marker so the entire tools array is cached cross-session for
// providers that support Anthropic prompt caching. Anthropic's cache order is
// tools → system → messages; the last-tool marker caches everything before it.
// Mirrors Hermes feat(prompt-cache): cross-session 1h prefix cache (7b7636655).
func markToolsForLongLivedCache(tools []ToolDescriptor, policy PromptCachePolicy) []ToolDescriptor {
	if !policy.ShouldCache || len(tools) == 0 {
		return tools
	}
	if policy.Layout != PromptCacheLayoutNativeAnthropic && policy.Layout != PromptCacheLayoutEnvelope {
		return tools
	}
	out := make([]ToolDescriptor, len(tools))
	copy(out, tools)
	last := out[len(out)-1]
	last.CacheControl = &CacheControl{Type: defaultPromptCacheControlType, TTL: defaultPromptCacheTTL}
	out[len(out)-1] = last
	return out
}

// ApplyPromptCacheControl returns a deep copy with the system_and_3 cache
// strategy: the first system message plus the last three non-system messages.
func ApplyPromptCacheControl(messages []Message, policy PromptCachePolicy) []Message {
	out := clonePromptCacheMessages(messages)
	if len(out) == 0 {
		return out
	}
	if !policy.ShouldCache {
		for idx := range out {
			out[idx].CacheControl = nil
		}
		return out
	}
	ttl := strings.TrimSpace(policy.TTL)
	if ttl == "" {
		ttl = defaultPromptCacheTTL
	}
	marker := CacheControl{Type: defaultPromptCacheControlType, TTL: ttl}
	breakpoints := 0
	if out[0].Role == "system" {
		out[0].CacheControl = cloneCacheControl(&marker)
		breakpoints++
	}
	remaining := 4 - breakpoints
	if remaining <= 0 {
		return out
	}
	indices := make([]int, 0, len(out))
	for idx := range out {
		if out[idx].Role != "system" {
			indices = append(indices, idx)
		}
	}
	if len(indices) > remaining {
		indices = indices[len(indices)-remaining:]
	}
	for _, idx := range indices {
		if policy.Layout == PromptCacheLayoutEnvelope && out[idx].Role == "tool" {
			continue
		}
		out[idx].CacheControl = cloneCacheControl(&marker)
	}
	return out
}

func clonePromptCacheMessages(messages []Message) []Message {
	return cloneMessages(messages)
}
