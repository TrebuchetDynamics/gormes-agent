package modelrouting

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

// ShouldSwapProvider reports whether a session-level provider override requires
// replacing the active client. Empty next providers keep the current client.
func ShouldSwapProvider(currentProvider, configuredProvider, nextProvider string) bool {
	current := strings.TrimSpace(FirstNonEmpty(currentProvider, configuredProvider))
	next := strings.TrimSpace(nextProvider)
	return next != "" && next != current
}

// Route resolves a provider/model pair into the provider client route used for
// cross-provider session model switches.
func Route(provider, model string) (llm.ModelRoute, bool) {
	entry, ok := llm.ResolveProviderManifestEntry(provider)
	if !ok {
		return llm.ModelRoute{}, false
	}
	if entry.ImplementationStatus != llm.ProviderImplemented && entry.ImplementationStatus != llm.ProviderOwned {
		return llm.ModelRoute{}, false
	}
	route := llm.ModelRoute{
		Provider:  strings.ToLower(strings.TrimSpace(entry.ID)),
		Model:     strings.TrimSpace(model),
		BaseURL:   strings.TrimSpace(entry.BaseURLOverride),
		APIMode:   APIMode(entry.TransportFamily),
		APIKeyEnv: FirstNonEmpty(entry.EnvVars...),
		KeyEnv:    strings.TrimSpace(entry.BaseURLEnvVar),
	}
	return route, route.Provider != "" && route.Model != ""
}

// APIMode normalizes provider manifest transport names into llm.ModelRoute API
// modes.
func APIMode(transport string) string {
	switch strings.TrimSpace(transport) {
	case "openai_chat":
		return "chat_completions"
	default:
		return strings.TrimSpace(transport)
	}
}

func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func MatchesAny(model string, needles []string) bool {
	for _, needle := range needles {
		needle = strings.ToLower(strings.TrimSpace(needle))
		if needle != "" && strings.Contains(model, needle) {
			return true
		}
	}
	return false
}

func SelectTurnModel(residentModel, override string) string {
	if model := strings.TrimSpace(override); model != "" {
		return model
	}
	return residentModel
}

func SelectTurnReasoningEffort(residentEffort, override string, status llm.ProviderStatus) llm.ReasoningEffortEvidence {
	if effort := strings.TrimSpace(override); effort != "" {
		return llm.ResolveReasoningEffort(effort, llm.ReasoningEffortSourceTurnOverride, status)
	}
	return llm.ResolveReasoningEffort(residentEffort, llm.ReasoningEffortSourceConfigDefault, status)
}
