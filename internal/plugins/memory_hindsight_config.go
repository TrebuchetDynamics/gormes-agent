package plugins

// HindsightConfig holds the memory hindsight plugin configuration.
type HindsightConfig struct {
	Mode        string            `json:"mode"`
	LLMProvider string            `json:"llm_provider"`
	LLMBaseURL  string            `json:"llm_base_url"`
	LLMModel    string            `json:"llm_model"`
	Timeout     int               `json:"timeout"`
	IdleTimeout int               `json:"idle_timeout"`
	APIKey      string            `json:"api_key,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// PatchHindsightConfig applies new values while preserving existing values
// when the new input is blank. Never overwrites API-key references with
// empty input. Returns the merged config.
func PatchHindsightConfig(existing HindsightConfig, incoming HindsightConfig) HindsightConfig {
	out := existing

	if incoming.Mode != "" {
		out.Mode = normalizeHindsightMode(incoming.Mode, existing.Mode)
	}
	if incoming.LLMProvider != "" {
		out.LLMProvider = incoming.LLMProvider
	}
	if incoming.LLMBaseURL != "" {
		out.LLMBaseURL = incoming.LLMBaseURL
	}
	if incoming.LLMModel != "" {
		out.LLMModel = incoming.LLMModel
	}
	if incoming.Timeout > 0 {
		out.Timeout = incoming.Timeout
	}
	if incoming.IdleTimeout > 0 {
		out.IdleTimeout = incoming.IdleTimeout
	}
	// APIKey: preserve existing reference, never overwrite with blank
	if incoming.APIKey != "" {
		out.APIKey = incoming.APIKey
	}

	if out.Extra == nil {
		out.Extra = make(map[string]string)
	}
	for k, v := range incoming.Extra {
		if v != "" {
			out.Extra[k] = v
		}
	}

	return out
}

func normalizeHindsightMode(incoming, existing string) string {
	switch incoming {
	case "local_embedded", "openai_compatible", "ollama", "disabled":
		return incoming
	default:
		if existing != "" {
			return existing
		}
		return "local_embedded"
	}
}
