package provideridentity

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

// NormalizeAuthProvider canonicalizes provider names used by provider auth and
// logout commands.
func NormalizeAuthProvider(provider string) string {
	normalized := strings.ReplaceAll(textvalue.LowerTrim(provider), "_", "-")
	switch normalized {
	case "or", "open-router":
		return "openrouter"
	case "novita-ai", "novitaai":
		return "novita"
	case "minimax-global", "minimax-portal", "minimax-oauth":
		return "minimax-oauth"
	default:
		return normalized
	}
}

// DisplayName returns the operator-facing display name for a provider ID.
func DisplayName(provider string) string {
	switch textvalue.LowerTrim(provider) {
	case "openai-codex":
		return "OpenAI Codex"
	case "openrouter":
		return "OpenRouter"
	case "xai":
		return "xAI"
	case "gmi":
		return "GMI"
	case "lmstudio":
		return "LM Studio"
	case "qwen-oauth":
		return "Qwen OAuth"
	case "google-gemini-cli":
		return "Google Gemini CLI"
	case "ai-gateway", "vercel":
		return "Vercel AI Gateway"
	}
	parts := strings.Fields(strings.ReplaceAll(provider, "-", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
