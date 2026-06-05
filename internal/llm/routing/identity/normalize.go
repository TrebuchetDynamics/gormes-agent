package identity

import "strings"

// Text normalizes provider/model identifiers that participate in routing keys.
func Text(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// Provider normalizes provider aliases into canonical routing provider IDs.
func Provider(provider string) string {
	switch Text(provider) {
	case "codex", "openai-codex":
		return "openai-codex"
	case "copilot", "copilot-acp", "github", "github-copilot", "github-models":
		return "copilot"
	case "google", "google-ai-studio", "google-gemini":
		return "gemini"
	case "open-router", "openrouter-free", "or":
		return "openrouter"
	case "ollama_cloud", "ollama-cloud":
		return "ollama-cloud"
	default:
		return Text(provider)
	}
}

// ModelBase returns the provider-neutral base model slug used by routing policy checks.
func ModelBase(model string) string {
	raw := Text(model)
	if raw == "" {
		return ""
	}
	if slash := strings.Index(raw, "/"); slash >= 0 {
		raw = raw[slash+1:]
	}
	if colon := strings.Index(raw, ":"); colon >= 0 {
		raw = raw[:colon]
	}
	return raw
}
