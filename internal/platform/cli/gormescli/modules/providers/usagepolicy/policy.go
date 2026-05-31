package usagepolicy

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// InferProvider infers the account-usage provider from configured
// provider/model settings when `gormes usage --provider` is not passed.
func InferProvider(configuredProvider, model string) string {
	provider := strings.TrimSpace(configuredProvider)
	if provider != "" {
		return provider
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}
	for _, candidate := range []string{"openai-codex", "anthropic", "openai", "openrouter"} {
		if metadata := llm.LookupModelMetadata(llm.ModelRegistryQuery{Provider: candidate, Model: model}); metadata.Found {
			return metadata.Provider
		}
	}
	lower := strings.ToLower(model)
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return "openai-codex"
	}
	if strings.Contains(lower, "claude") {
		return "anthropic"
	}
	return ""
}

// FirstNonBlank returns the first non-blank value.
func FirstNonBlank(values ...string) string {
	return textvalue.FirstNonBlank(values...)
}
