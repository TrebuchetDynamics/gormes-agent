package routing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/identity"
)

type ProviderDefaultModelSource string

const (
	ProviderDefaultModelSourceUnknown         ProviderDefaultModelSource = "unknown_provider"
	ProviderDefaultModelSourceCodexConfig     ProviderDefaultModelSource = "codex_config"
	ProviderDefaultModelSourceCodexCache      ProviderDefaultModelSource = "codex_cache"
	ProviderDefaultModelSourceCuratedFallback ProviderDefaultModelSource = "curated_fallback"
	ProviderDefaultModelSourceStaticCatalog   ProviderDefaultModelSource = "static_catalog"
)

type ProviderDefaultModelOptions struct {
	CodexHome string
}

type ProviderDefaultModelResolution struct {
	Provider string
	Model    string
	Source   ProviderDefaultModelSource
}

var openAICodexDefaultModels = []string{
	"gpt-5.5",
	"gpt-5.4-mini",
	"gpt-5.4",
	"gpt-5.3-codex",
	"gpt-5.3-codex-spark",
	"gpt-5.2-codex",
	"gpt-5.1-codex-max",
	"gpt-5.1-codex-mini",
}

var providerDefaultModelFloor = map[string]string{
	"ai-gateway":          "moonshotai/kimi-k2.6",
	"alibaba":             "qwen3.6-plus",
	"alibaba-coding-plan": "qwen3.6-plus",
	"anthropic":           "claude-opus-4-7",
	"bedrock":             "us.anthropic.claude-sonnet-4-6",
	"copilot":             "gpt-5.4",
	"copilot-acp":         "copilot-acp",
	"deepseek":            "deepseek-v4-pro",
	"gemini":              "gemini-2.5-flash",
	"gmi":                 "zai-org/GLM-5.1-FP8",
	"google-gemini-cli":   "gemini-3.1-pro-preview",
	"groq":                "llama-3.3-70b-versatile",
	"huggingface":         "moonshotai/Kimi-K2.5",
	"kilocode":            "anthropic/claude-opus-4.6",
	"minimax-cn":          "MiniMax-M2.7",
	"nous":                "moonshotai/kimi-k2.6",
	"novita":              "moonshotai/kimi-k2.5",
	"nvidia":              "nvidia/nemotron-3-super-120b-a12b",
	"openai":              "gpt-5.4",
	"openrouter":          "deepseek/deepseek-chat-v3-0324:free",
	"opencode-go":         "kimi-k2.6",
	"opencode-zen":        "kimi-k2.5",
	"stepfun":             "step-3.5-flash",
	"tencent-tokenhub":    "hy3-preview",
	"xai":                 "grok-4.20-0309-reasoning",
	"xiaomi":              "mimo-v2.5-pro",
	"zai":                 "glm-5.1",
}

func ResolveProviderDefaultModel(provider string, opts ProviderDefaultModelOptions) ProviderDefaultModelResolution {
	normalized := normalizeProviderDefaultModelProvider(provider)
	resolution := ProviderDefaultModelResolution{
		Provider: normalized,
		Source:   ProviderDefaultModelSourceUnknown,
	}
	switch normalized {
	case "openai-codex":
		codexHome := resolveCodexHome(opts.CodexHome)
		if model := readCodexDefaultModel(codexHome); model != "" {
			resolution.Model = model
			resolution.Source = ProviderDefaultModelSourceCodexConfig
			return resolution
		}
		if model := readCodexCachedDefaultModel(codexHome); model != "" {
			resolution.Model = model
			resolution.Source = ProviderDefaultModelSourceCodexCache
			return resolution
		}
		resolution.Model = openAICodexDefaultModels[0]
		resolution.Source = ProviderDefaultModelSourceCuratedFallback
		return resolution
	default:
		if model := providerDefaultModelFloor[normalized]; model != "" {
			resolution.Model = model
			resolution.Source = ProviderDefaultModelSourceStaticCatalog
		}
		return resolution
	}
}

func resolveCodexHome(explicit string) string {
	if explicit = strings.TrimSpace(explicit); explicit != "" {
		return explicit
	}
	if fromEnv := strings.TrimSpace(os.Getenv("CODEX_HOME")); fromEnv != "" {
		return fromEnv
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".codex")
}

func readCodexDefaultModel(codexHome string) string {
	if strings.TrimSpace(codexHome) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		return ""
	}
	var payload struct {
		Model string `toml:"model"`
	}
	if err := toml.Unmarshal(data, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Model)
}

func readCodexCachedDefaultModel(codexHome string) string {
	if strings.TrimSpace(codexHome) == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(codexHome, "models_cache.json"))
	if err != nil {
		return ""
	}
	var payload struct {
		Models []struct {
			Slug       string   `json:"slug"`
			Visibility string   `json:"visibility"`
			Priority   *float64 `json:"priority"`
		} `json:"models"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	type candidate struct {
		slug     string
		priority float64
	}
	var candidates []candidate
	for _, item := range payload.Models {
		slug := strings.TrimSpace(item.Slug)
		if slug == "" {
			continue
		}
		// supported_in_api describes public OpenAI API availability, not
		// Codex OAuth backend availability; visible CLI-only models still count.
		visibility := strings.ToLower(strings.TrimSpace(item.Visibility))
		if visibility == "hide" || visibility == "hidden" {
			continue
		}
		priority := 10_000.0
		if item.Priority != nil {
			priority = *item.Priority
		}
		candidates = append(candidates, candidate{slug: slug, priority: priority})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority == candidates[j].priority {
			return candidates[i].slug < candidates[j].slug
		}
		return candidates[i].priority < candidates[j].priority
	})
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].slug
}

func normalizeProviderDefaultModelProvider(provider string) string {
	return identity.Provider(provider)
}
