package modelcatalog

import (
	"regexp"
	"strings"
)

var modelVendorPrefixes = map[string]string{
	"claude":   "anthropic",
	"gpt":      "openai",
	"o1":       "openai",
	"o3":       "openai",
	"o4":       "openai",
	"gemini":   "google",
	"gemma":    "google",
	"deepseek": "deepseek",
	"glm":      "z-ai",
	"kimi":     "moonshotai",
	"minimax":  "minimax",
	"grok":     "x-ai",
	"qwen":     "qwen",
	"mimo":     "xiaomi",
	"trinity":  "arcee-ai",
	"nemotron": "nvidia",
	"llama":    "meta-llama",
	"step":     "stepfun",
}

var modelNormalizeProviderAliases = map[string]string{
	"glm":                 "zai",
	"z-ai":                "zai",
	"z.ai":                "zai",
	"zhipu":               "zai",
	"github":              "copilot",
	"github-copilot":      "copilot",
	"github-models":       "copilot",
	"github-model":        "copilot",
	"github-copilot-acp":  "copilot-acp",
	"copilot-acp-agent":   "copilot-acp",
	"codex":               "openai-codex",
	"google":              "gemini",
	"google-gemini":       "gemini",
	"google-ai-studio":    "gemini",
	"kimi":                "kimi-coding",
	"moonshot":            "kimi-coding",
	"kimi-cn":             "kimi-coding-cn",
	"moonshot-cn":         "kimi-coding-cn",
	"step":                "stepfun",
	"stepfun-coding-plan": "stepfun",
	"arcee-ai":            "arcee",
	"arceeai":             "arcee",
	"minimax-china":       "minimax-cn",
	"minimax_cn":          "minimax-cn",
	"minimax-portal":      "minimax-oauth",
	"minimax-global":      "minimax-oauth",
	"minimax_oauth":       "minimax-oauth",
	"claude":              "anthropic",
	"claude-code":         "anthropic",
	"deep-seek":           "deepseek",
	"opencode":            "opencode-zen",
	"zen":                 "opencode-zen",
	"go":                  "opencode-go",
	"opencode-go-sub":     "opencode-go",
	"kilo":                "kilocode",
	"kilo-code":           "kilocode",
	"kilo-gateway":        "kilocode",
	"dashscope":           "alibaba",
	"aliyun":              "alibaba",
	"qwen":                "alibaba",
	"alibaba-cloud":       "alibaba",
	"qwen-portal":         "qwen-oauth",
	"hf":                  "huggingface",
	"hugging-face":        "huggingface",
	"huggingface-hub":     "huggingface",
	"mimo":                "xiaomi",
	"xiaomi-mimo":         "xiaomi",
	"tencent":             "tencent-tokenhub",
	"tokenhub":            "tencent-tokenhub",
	"tencent-cloud":       "tencent-tokenhub",
	"aws":                 "bedrock",
	"aws-bedrock":         "bedrock",
	"amazon-bedrock":      "bedrock",
	"amazon":              "bedrock",
	"grok":                "xai",
	"x-ai":                "xai",
	"x.ai":                "xai",
	"nim":                 "nvidia",
	"nvidia-nim":          "nvidia",
	"build-nvidia":        "nvidia",
	"nemotron":            "nvidia",
	"ollama_cloud":        "ollama-cloud",
	"open-router":         "openrouter",
	"openrouter-free":     "openrouter",
	"or":                  "openrouter",
}

var aggregatorModelProviders = map[string]struct{}{
	"openrouter": {},
	"nous":       {},
	"kilocode":   {},
}

var dotToHyphenModelProviders = map[string]struct{}{
	"anthropic": {},
}

var stripVendorOnlyModelProviders = map[string]struct{}{
	"copilot":      {},
	"copilot-acp":  {},
	"openai-codex": {},
}

var authoritativeNativeModelProviders = map[string]struct{}{
	"gemini":      {},
	"huggingface": {},
}

var matchingPrefixStripModelProviders = map[string]struct{}{
	"zai":            {},
	"kimi-coding":    {},
	"kimi-coding-cn": {},
	"minimax":        {},
	"minimax-oauth":  {},
	"minimax-cn":     {},
	"alibaba":        {},
	"qwen-oauth":     {},
	"xiaomi":         {},
	"arcee":          {},
	"ollama-cloud":   {},
	"custom":         {},
}

var lowercaseModelProviders = map[string]struct{}{
	"xiaomi": {},
}

var deepSeekReasonerKeywords = []string{"reasoner", "r1", "think", "reasoning", "cot"}

var deepSeekCanonicalModels = map[string]struct{}{
	"deepseek-chat":     {},
	"deepseek-reasoner": {},
	"deepseek-v4-pro":   {},
	"deepseek-v4-flash": {},
}

var deepSeekVSeriesPattern = regexp.MustCompile(`^deepseek-v\d+([-.].+)?$`)

var copilotModelAliases = map[string]string{
	"openai/gpt-5":                "gpt-5-mini",
	"openai/gpt-5-chat":           "gpt-5-mini",
	"openai/gpt-5-mini":           "gpt-5-mini",
	"openai/gpt-5-nano":           "gpt-5-mini",
	"openai/gpt-4.1":              "gpt-4.1",
	"openai/gpt-4.1-mini":         "gpt-4.1",
	"openai/gpt-4.1-nano":         "gpt-4.1",
	"openai/gpt-4o":               "gpt-4o",
	"openai/gpt-4o-mini":          "gpt-4o-mini",
	"openai/o1":                   "gpt-5.2",
	"openai/o1-mini":              "gpt-5-mini",
	"openai/o1-preview":           "gpt-5.2",
	"openai/o3":                   "gpt-5.3-codex",
	"openai/o3-mini":              "gpt-5-mini",
	"openai/o4-mini":              "gpt-5-mini",
	"anthropic/claude-opus-4.6":   "claude-opus-4.6",
	"anthropic/claude-sonnet-4.6": "claude-sonnet-4.6",
	"anthropic/claude-sonnet-4":   "claude-sonnet-4",
	"anthropic/claude-sonnet-4.5": "claude-sonnet-4.5",
	"anthropic/claude-haiku-4.5":  "claude-haiku-4.5",
	"claude-opus-4-6":             "claude-opus-4.6",
	"claude-sonnet-4-6":           "claude-sonnet-4.6",
	"claude-sonnet-4-0":           "claude-sonnet-4",
	"claude-sonnet-4-5":           "claude-sonnet-4.5",
	"claude-haiku-4-5":            "claude-haiku-4.5",
	"anthropic/claude-opus-4-6":   "claude-opus-4.6",
	"anthropic/claude-sonnet-4-6": "claude-sonnet-4.6",
	"anthropic/claude-sonnet-4-0": "claude-sonnet-4",
	"anthropic/claude-sonnet-4-5": "claude-sonnet-4.5",
	"anthropic/claude-haiku-4-5":  "claude-haiku-4.5",
}

// NormalizeModelForProvider translates a user-facing model identifier into the
// model name expected by the target provider. It mirrors
// hermes-agent/hermes_cli/model_normalize.py normalize_model_for_provider.
func NormalizeModelForProvider(modelInput, targetProvider string) string {
	name := strings.TrimSpace(modelInput)
	if name == "" {
		return name
	}
	provider := normalizeModelProviderAlias(targetProvider)

	if _, ok := aggregatorModelProviders[provider]; ok {
		return prependModelVendor(name)
	}

	if provider == "opencode-zen" || provider == "opencode-go" {
		if strings.Contains(name, "/") {
			_, bare, _ := strings.Cut(name, "/")
			if strings.TrimSpace(bare) != "" {
				name = strings.TrimSpace(bare)
			}
		}
		if provider == "opencode-zen" && strings.HasPrefix(strings.ToLower(name), "claude-") {
			return dotsToHyphens(name)
		}
		return name
	}

	if _, ok := dotToHyphenModelProviders[provider]; ok {
		bare := stripMatchingProviderPrefix(name, provider)
		if strings.Contains(bare, "/") {
			return bare
		}
		return dotsToHyphens(bare)
	}

	if provider == "copilot" || provider == "copilot-acp" {
		if normalized := normalizeCopilotModelID(name); normalized != "" {
			return normalized
		}
	}

	if _, ok := stripVendorOnlyModelProviders[provider]; ok {
		stripped := stripMatchingProviderPrefix(name, provider)
		if stripped == name && strings.HasPrefix(name, "openai/") {
			_, bare, _ := strings.Cut(name, "/")
			return bare
		}
		return stripped
	}

	if provider == "deepseek" {
		bare := stripMatchingProviderPrefix(name, provider)
		if strings.Contains(bare, "/") {
			return bare
		}
		return NormalizeDeepSeekModelID(bare)
	}

	if _, ok := matchingPrefixStripModelProviders[provider]; ok {
		result := stripMatchingProviderPrefix(name, provider)
		if _, lowercase := lowercaseModelProviders[provider]; lowercase {
			result = strings.ToLower(result)
		}
		return result
	}

	if _, ok := authoritativeNativeModelProviders[provider]; ok {
		return name
	}

	return name
}

// DetectModelVendor detects the aggregator vendor slug for a model name.
func DetectModelVendor(modelName string) string {
	name := strings.TrimSpace(modelName)
	if name == "" {
		return ""
	}
	if vendor, _, ok := strings.Cut(name, "/"); ok {
		return strings.ToLower(vendor)
	}

	lower := strings.ToLower(name)
	firstToken := strings.Split(lower, "-")[0]
	if vendor, ok := modelVendorPrefixes[firstToken]; ok {
		return vendor
	}
	for prefix, vendor := range modelVendorPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return vendor
		}
	}
	return ""
}

func NormalizeDeepSeekModelID(modelName string) string {
	bare := strings.ToLower(stripAnyModelVendorPrefix(modelName))
	if _, ok := deepSeekCanonicalModels[bare]; ok {
		return bare
	}
	if deepSeekVSeriesPattern.MatchString(bare) {
		return bare
	}
	for _, keyword := range deepSeekReasonerKeywords {
		if strings.Contains(bare, keyword) {
			return "deepseek-reasoner"
		}
	}
	return "deepseek-chat"
}

func normalizeModelProviderAlias(providerName string) string {
	raw := strings.ToLower(strings.TrimSpace(providerName))
	if raw == "" {
		return raw
	}
	if canonical, ok := modelNormalizeProviderAliases[raw]; ok {
		return canonical
	}
	return raw
}

func stripAnyModelVendorPrefix(modelName string) string {
	name := strings.TrimSpace(modelName)
	if _, bare, ok := strings.Cut(name, "/"); ok {
		return strings.TrimSpace(bare)
	}
	return name
}

func dotsToHyphens(modelName string) string {
	return strings.ReplaceAll(modelName, ".", "-")
}

func prependModelVendor(modelName string) string {
	if strings.Contains(modelName, "/") {
		return modelName
	}
	if vendor := DetectModelVendor(modelName); vendor != "" {
		return vendor + "/" + modelName
	}
	return modelName
}

func stripMatchingProviderPrefix(modelName, targetProvider string) string {
	prefix, remainder, ok := strings.Cut(modelName, "/")
	if !ok {
		return modelName
	}
	prefix = strings.TrimSpace(prefix)
	remainder = strings.TrimSpace(remainder)
	if prefix == "" || remainder == "" {
		return modelName
	}
	if normalizeModelProviderAlias(prefix) == normalizeModelProviderAlias(targetProvider) {
		return remainder
	}
	return modelName
}

func normalizeCopilotModelID(modelID string) string {
	raw := strings.TrimSpace(modelID)
	if raw == "" {
		return ""
	}
	if alias, ok := copilotModelAliases[raw]; ok {
		return alias
	}

	candidates := []string{raw}
	if _, bare, ok := strings.Cut(raw, "/"); ok {
		candidates = append(candidates, strings.TrimSpace(bare))
	}
	for _, suffix := range []string{"-mini", "-nano", "-chat"} {
		if strings.HasSuffix(raw, suffix) {
			candidates = append(candidates, strings.TrimSuffix(raw, suffix))
		}
	}

	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if alias, ok := copilotModelAliases[candidate]; ok {
			return alias
		}
	}

	if _, bare, ok := strings.Cut(raw, "/"); ok {
		return strings.TrimSpace(bare)
	}
	return raw
}
