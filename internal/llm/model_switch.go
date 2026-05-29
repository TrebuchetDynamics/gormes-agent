package llm

import (
	"sort"
	"strings"
)

// ModelIdentity represents a vendor slug and family prefix used for catalog
// resolution. Mirrors hermes-agent/hermes_cli/model_switch.py ModelIdentity.
type ModelIdentity struct {
	Vendor string
	Family string
}

// ModelAliases is the built-in map of short alias names to ModelIdentity,
// matching hermes-agent/hermes_cli/model_switch.py MODEL_ALIASES.
var ModelAliases = map[string]ModelIdentity{
	"sonnet":   {Vendor: "anthropic", Family: "claude-sonnet"},
	"opus":     {Vendor: "anthropic", Family: "claude-opus"},
	"haiku":    {Vendor: "anthropic", Family: "claude-haiku"},
	"claude":   {Vendor: "anthropic", Family: "claude"},
	"gpt5":     {Vendor: "openai", Family: "gpt-5"},
	"gpt":      {Vendor: "openai", Family: "gpt"},
	"codex":    {Vendor: "openai", Family: "codex"},
	"o3":       {Vendor: "openai", Family: "o3"},
	"o4":       {Vendor: "openai", Family: "o4"},
	"gemini":   {Vendor: "google", Family: "gemini"},
	"deepseek": {Vendor: "deepseek", Family: "deepseek-chat"},
	"grok":     {Vendor: "x-ai", Family: "grok"},
	"llama":    {Vendor: "meta-llama", Family: "llama"},
	"qwen":     {Vendor: "qwen", Family: "qwen"},
	"minimax":  {Vendor: "minimax", Family: "minimax"},
	"nemotron": {Vendor: "nvidia", Family: "nemotron"},
	"kimi":     {Vendor: "moonshotai", Family: "kimi"},
	"glm":      {Vendor: "z-ai", Family: "glm"},
	"step":     {Vendor: "stepfun", Family: "step"},
	"mimo":     {Vendor: "xiaomi", Family: "mimo"},
	"trinity":  {Vendor: "arcee-ai", Family: "trinity"},
}

// DirectAlias is an exact model mapping that bypasses catalog resolution.
// Mirrors hermes-agent/hermes_cli/model_switch.py DirectAlias.
type DirectAlias struct {
	Model    string
	Provider string
	BaseURL  string
}

// ModelSwitchResult is the result of a model switch attempt.
// Mirrors hermes-agent/hermes_cli/model_switch.py ModelSwitchResult.
type ModelSwitchResult struct {
	Success          bool
	NewModel         string
	TargetProvider   string
	ProviderChanged  bool
	APIKey           string
	BaseURL          string
	APIMode          string
	ErrorMessage     string
	WarningMessage   string
	ProviderLabel    string
	ResolvedViaAlias string
	IsGlobal         bool
}

// ParseModelFlags parses --provider and --global flags from model command args.
// Mirrors hermes-agent/hermes_cli/model_switch.py parse_model_flags.
func ParseModelFlags(rawArgs string) (modelInput string, explicitProvider string, isGlobal bool) {
	// Normalize Unicode dashes (Telegram/iOS auto-converts -- to em/en dash)
	replacer := strings.NewReplacer(
		"\u2012", "--",
		"\u2013", "--",
		"\u2014", "--",
		"\u2015", "--",
	)
	normalized := replacer.Replace(rawArgs)

	isGlobal = strings.Contains(normalized, "--global")
	normalized = strings.ReplaceAll(normalized, "--global", "")
	normalized = strings.TrimSpace(normalized)

	// Extract --provider <name>
	parts := strings.Fields(normalized)
	var filtered []string
	for i := 0; i < len(parts); i++ {
		if parts[i] == "--provider" && i+1 < len(parts) {
			explicitProvider = parts[i+1]
			i++
		} else {
			filtered = append(filtered, parts[i])
		}
	}
	modelInput = strings.TrimSpace(strings.Join(filtered, " "))
	return
}

// ModelSortKey produces a deterministic sort key for model IDs.
// Mirrors hermes-agent/hermes_cli/model_switch.py _model_sort_key.
func ModelSortKey(modelID, prefix string) (int, string, string) {
	rest := modelID
	if prefix != "" && strings.HasPrefix(modelID, prefix) {
		rest = strings.TrimPrefix(modelID, prefix)
	}
	return 0, prefix, rest
}

// SortedModelAliases returns model alias keys sorted by ModelSortKey.
func SortedModelAliases() []string {
	keys := make([]string, 0, len(ModelAliases))
	for k := range ModelAliases {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		_, pi, ri := ModelSortKey(ModelAliases[keys[i]].Family, "")
		_, pj, rj := ModelSortKey(ModelAliases[keys[j]].Family, "")
		if pi != pj {
			return pi < pj
		}
		return ri < rj
	})
	return keys
}
