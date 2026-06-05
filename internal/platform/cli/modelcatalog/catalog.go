package modelcatalog

const (
	ProviderCatalogAuxConfig      = "aux-config"
	ProviderCatalogLeaveUnchanged = "cancel"
)

// ProviderEntry is one Hermes-compatible provider picker row.
// ID is the provider slug or setup action; Label is the user-visible picker
// text from upstream Hermes' CANONICAL_PROVIDERS/tui_desc taxonomy.
type ProviderEntry struct {
	ID    string
	Label string
}

// HermesProviderCatalog returns the curated provider order shared by Hermes
// setup/model. Keep labels and order aligned with hermes_cli/models.py
// CANONICAL_PROVIDERS plus the trailing setup actions.
func HermesProviderCatalog() []ProviderEntry {
	entries := []ProviderEntry{
		{ID: "nous", Label: "Nous Portal (Nous Research subscription)"},
		{ID: "openrouter", Label: "OpenRouter (100+ models, pay-per-use)"},
		{ID: "novita", Label: "NovitaAI (AI-native cloud: Model API, Agent Sandbox, GPU Cloud)"},
		{ID: "lmstudio", Label: "LM Studio (local desktop app with built-in model server)"},
		{ID: "anthropic", Label: "Anthropic (Claude models - API key or Claude Code)"},
		{ID: "openai-codex", Label: "OpenAI Codex"},
		{ID: "alibaba", Label: "Qwen Cloud / DashScope Coding (Qwen + multi-provider)"},
		{ID: "xai-oauth", Label: "xAI Grok OAuth (SuperGrok Subscription)"},
		{ID: "xiaomi", Label: "Xiaomi MiMo (MiMo-V2.5 and V2 models - pro, omni, flash)"},
		{ID: "tencent-tokenhub", Label: "Tencent TokenHub (Hy3 Preview - direct API via tokenhub.tencentmaas.com)"},
		{ID: "nvidia", Label: "NVIDIA NIM (Nemotron models - build.nvidia.com or local NIM)"},
		{ID: "copilot", Label: "GitHub Copilot (uses GITHUB_TOKEN or gh auth token)"},
		{ID: "copilot-acp", Label: "GitHub Copilot ACP (spawns `copilot --acp --stdio`)"},
		{ID: "huggingface", Label: "Hugging Face Inference Providers (20+ open models)"},
		{ID: "gemini", Label: "Google AI Studio (Gemini models - native Gemini API)"},
		{ID: "google-gemini-cli", Label: "Google Gemini via OAuth + Code Assist (free tier supported; no API key needed)"},
		{ID: "deepseek", Label: "DeepSeek (DeepSeek-V3, R1, coder - direct API)"},
		{ID: "xai", Label: "xAI (Grok models - direct API)"},
		{ID: "zai", Label: "Z.AI / GLM (Zhipu AI direct API)"},
		{ID: "kimi-coding", Label: "Kimi Coding Plan (api.kimi.com) & Moonshot API"},
		{ID: "kimi-coding-cn", Label: "Kimi / Moonshot China (Moonshot CN direct API)"},
		{ID: "stepfun", Label: "StepFun Step Plan (agent/coding models via Step Plan API)"},
		{ID: "minimax", Label: "MiniMax (global direct API)"},
		{ID: "minimax-oauth", Label: "MiniMax via OAuth browser login (Coding Plan, minimax.io)"},
		{ID: "minimax-cn", Label: "MiniMax China (domestic direct API)"},
		{ID: "ollama-cloud", Label: "Ollama Cloud (cloud-hosted open models - ollama.com)"},
		{ID: "arcee", Label: "Arcee AI (Trinity models - direct API)"},
		{ID: "gmi", Label: "GMI Cloud (multi-model direct API)"},
		{ID: "kilocode", Label: "Kilo Code (Kilo Gateway API)"},
		{ID: "opencode-zen", Label: "OpenCode Zen (35+ curated models, pay-as-you-go)"},
		{ID: "opencode-go", Label: "OpenCode Go (open models, $10/month subscription)"},
		{ID: "bedrock", Label: "AWS Bedrock (Claude, Nova, Llama, DeepSeek - IAM or API key)"},
		{ID: "azure-foundry", Label: "Azure Foundry (OpenAI-style or Anthropic-style endpoint - your Azure AI deployment)"},
		{ID: "ai-gateway", Label: "Vercel AI Gateway"},
		{ID: "qwen-oauth", Label: "Qwen OAuth (reuses local Qwen CLI login)"},
		{ID: "alibaba-coding-plan", Label: "Alibaba Cloud Coding Plan - dedicated coding tier"},
		{ID: "custom", Label: "custom (direct API)"},
		{ID: "custom-endpoint", Label: "Custom endpoint (enter URL manually)"},
		{ID: ProviderCatalogAuxConfig, Label: "Configure auxiliary models..."},
		{ID: ProviderCatalogLeaveUnchanged, Label: "Leave unchanged"},
	}
	return append([]ProviderEntry(nil), entries...)
}

// HermesModelProviderCatalog returns the provider rows that can be used by
// selection-only model pickers. Setup-only actions are intentionally excluded.
func HermesModelProviderCatalog() []ProviderEntry {
	catalog := HermesProviderCatalog()
	entries := make([]ProviderEntry, 0, len(catalog))
	for _, entry := range catalog {
		switch entry.ID {
		case ProviderCatalogAuxConfig, ProviderCatalogLeaveUnchanged, "custom-endpoint":
			continue
		default:
			entries = append(entries, entry)
		}
	}
	return entries
}
