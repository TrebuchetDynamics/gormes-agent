// Code generated from Hermes provider inventory by Mineru; edit deliberately.
package hermes

import "strings"

type ProviderImplementationStatus string

const (
	ProviderImplemented ProviderImplementationStatus = "implemented"
	ProviderOwned       ProviderImplementationStatus = "owned"
	ProviderRowBacked   ProviderImplementationStatus = "row_backed"
	ProviderExcluded    ProviderImplementationStatus = "excluded"
)

type ProviderManifestEntry struct {
	ID                   string
	Aliases              []string
	ModelsDevID          string
	TransportFamily      string
	AuthType             string
	Aggregator           bool
	EnvVars              []string
	BaseURLOverride      string
	BaseURLEnvVar        string
	ImplementationStatus ProviderImplementationStatus
}

func HermesProviderRegistryManifest() []ProviderManifestEntry {
	entries := []ProviderManifestEntry{
		{ID: "alibaba", Aliases: []string{"alibaba-cloud", "aliyun", "dashscope", "qwen"}, ModelsDevID: "alibaba", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"DASHSCOPE_API_KEY"}, BaseURLOverride: "https://dashscope-intl.aliyuncs.com/compatible-mode/v1", BaseURLEnvVar: "DASHSCOPE_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "alibaba-coding-plan", Aliases: []string{"alibaba-coding", "alibaba_coding", "alibaba_coding_plan"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"ALIBABA_CODING_PLAN_API_KEY", "DASHSCOPE_API_KEY"}, BaseURLOverride: "https://coding-intl.dashscope.aliyuncs.com/v1", BaseURLEnvVar: "ALIBABA_CODING_PLAN_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "anthropic", Aliases: []string{"claude", "claude-code"}, ModelsDevID: "anthropic", TransportFamily: "anthropic_messages", AuthType: "api_key", Aggregator: false, EnvVars: []string{"ANTHROPIC_API_KEY", "ANTHROPIC_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN"}, BaseURLOverride: "https://api.anthropic.com", BaseURLEnvVar: "ANTHROPIC_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "arcee", Aliases: []string{"arcee-ai", "arceeai"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"ARCEEAI_API_KEY"}, BaseURLOverride: "https://api.arcee.ai/api/v1", BaseURLEnvVar: "ARCEE_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "azure-foundry", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"AZURE_FOUNDRY_API_KEY"}, BaseURLOverride: "", BaseURLEnvVar: "AZURE_FOUNDRY_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "bedrock", Aliases: []string{"amazon", "amazon-bedrock", "aws", "aws-bedrock"}, ModelsDevID: "", TransportFamily: "bedrock_converse", AuthType: "aws_sdk", Aggregator: false, EnvVars: nil, BaseURLOverride: "https://bedrock-runtime.us-east-1.amazonaws.com", BaseURLEnvVar: "BEDROCK_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "cohere", Aliases: nil, ModelsDevID: "cohere", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "copilot-acp", Aliases: []string{"github-copilot-acp"}, ModelsDevID: "", TransportFamily: "codex_responses", AuthType: "external_process", Aggregator: false, EnvVars: nil, BaseURLOverride: "acp://copilot", BaseURLEnvVar: "COPILOT_ACP_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "custom", Aliases: []string{"ollama"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderImplemented},
		{ID: "deepseek", Aliases: []string{"deep-seek"}, ModelsDevID: "deepseek", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"DEEPSEEK_API_KEY"}, BaseURLOverride: "https://api.deepseek.com/v1", BaseURLEnvVar: "DEEPSEEK_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "fireworks", Aliases: nil, ModelsDevID: "fireworks-ai", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "gemini", Aliases: nil, ModelsDevID: "google", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"GOOGLE_API_KEY", "GEMINI_API_KEY"}, BaseURLOverride: "https://generativelanguage.googleapis.com/v1beta", BaseURLEnvVar: "GEMINI_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "github-copilot", Aliases: []string{"copilot", "github"}, ModelsDevID: "github-copilot", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"COPILOT_GITHUB_TOKEN", "GH_TOKEN", "GITHUB_TOKEN"}, BaseURLOverride: "", BaseURLEnvVar: "COPILOT_API_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "github-models", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "gmi", Aliases: []string{"gmi-cloud", "gmicloud"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"GMI_API_KEY"}, BaseURLOverride: "https://api.gmi-serving.com/v1", BaseURLEnvVar: "GMI_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "google", Aliases: nil, ModelsDevID: "google", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "google-ai-studio", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "google-gemini", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "google-gemini-cli", Aliases: []string{"gemini-cli", "gemini-oauth"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "oauth_external", Aggregator: false, EnvVars: nil, BaseURLOverride: "cloudcode-pa://google", BaseURLEnvVar: "", ImplementationStatus: ProviderOwned},
		{ID: "groq", Aliases: nil, ModelsDevID: "groq", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "huggingface", Aliases: []string{"hf", "hugging-face", "huggingface-hub"}, ModelsDevID: "huggingface", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"HF_TOKEN"}, BaseURLOverride: "https://router.huggingface.co/v1", BaseURLEnvVar: "HF_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "kilo", Aliases: []string{"kilo-code", "kilo-gateway", "kilocode"}, ModelsDevID: "kilo", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"KILOCODE_API_KEY"}, BaseURLOverride: "https://api.kilo.ai/api/gateway", BaseURLEnvVar: "KILOCODE_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "kimi-cn", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "kimi-for-coding", Aliases: []string{"kimi", "kimi-coding", "kimi-coding-cn", "moonshot"}, ModelsDevID: "kimi-for-coding", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"KIMI_API_KEY", "KIMI_CODING_API_KEY"}, BaseURLOverride: "https://api.moonshot.ai/v1", BaseURLEnvVar: "KIMI_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "lmstudio", Aliases: []string{"lm-studio", "lm_studio", "lmstudio"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"LM_API_KEY"}, BaseURLOverride: "http://127.0.0.1:1234/v1", BaseURLEnvVar: "LM_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "local", Aliases: []string{"llama-cpp", "llama.cpp", "llamacpp", "vllm"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderImplemented},
		{ID: "minimax", Aliases: nil, ModelsDevID: "minimax", TransportFamily: "anthropic_messages", AuthType: "api_key", Aggregator: false, EnvVars: []string{"MINIMAX_API_KEY"}, BaseURLOverride: "https://api.minimax.io/anthropic", BaseURLEnvVar: "MINIMAX_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "minimax-cn", Aliases: []string{"minimax-china", "minimax_cn"}, ModelsDevID: "minimax-cn", TransportFamily: "anthropic_messages", AuthType: "api_key", Aggregator: false, EnvVars: []string{"MINIMAX_CN_API_KEY"}, BaseURLOverride: "https://api.minimaxi.com/anthropic", BaseURLEnvVar: "MINIMAX_CN_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "mistral", Aliases: nil, ModelsDevID: "mistral", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "moonshot-cn", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "nous", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "oauth_device_code", Aggregator: false, EnvVars: nil, BaseURLOverride: "https://inference-api.nousresearch.com/v1", BaseURLEnvVar: "", ImplementationStatus: ProviderOwned},
		{ID: "nvidia", Aliases: []string{"build-nvidia", "nemotron", "nim", "nvidia-nim"}, ModelsDevID: "nvidia", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"NVIDIA_API_KEY"}, BaseURLOverride: "https://integrate.api.nvidia.com/v1", BaseURLEnvVar: "NVIDIA_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "ollama-cloud", Aliases: nil, ModelsDevID: "ollama-cloud", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"OLLAMA_API_KEY"}, BaseURLOverride: "", BaseURLEnvVar: "OLLAMA_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "openai-codex", Aliases: nil, ModelsDevID: "openai", TransportFamily: "codex_responses", AuthType: "oauth_external", Aggregator: false, EnvVars: nil, BaseURLOverride: "https://chatgpt.com/backend-api/codex", BaseURLEnvVar: "", ImplementationStatus: ProviderImplemented},
		{ID: "opencode", Aliases: []string{"opencode-zen", "zen"}, ModelsDevID: "opencode", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"OPENCODE_ZEN_API_KEY"}, BaseURLOverride: "https://opencode.ai/zen/v1", BaseURLEnvVar: "OPENCODE_ZEN_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "opencode-go", Aliases: []string{"go", "opencode-go-sub"}, ModelsDevID: "opencode-go", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"OPENCODE_GO_API_KEY"}, BaseURLOverride: "https://opencode.ai/zen/go/v1", BaseURLEnvVar: "OPENCODE_GO_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "openrouter", Aliases: []string{"openai"}, ModelsDevID: "openrouter", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"OPENROUTER_API_KEY", "OPENAI_API_KEY"}, BaseURLOverride: "", BaseURLEnvVar: "OPENROUTER_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "perplexity", Aliases: nil, ModelsDevID: "perplexity", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "qwen-oauth", Aliases: nil, ModelsDevID: "alibaba", TransportFamily: "openai_chat", AuthType: "oauth_external", Aggregator: false, EnvVars: nil, BaseURLOverride: "https://portal.qwen.ai/v1", BaseURLEnvVar: "HERMES_QWEN_BASE_URL", ImplementationStatus: ProviderOwned},
		{ID: "qwen-portal", Aliases: nil, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "stepfun", Aliases: []string{"step", "stepfun-coding-plan"}, ModelsDevID: "stepfun", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"STEPFUN_API_KEY"}, BaseURLOverride: "https://api.stepfun.ai/step_plan/v1", BaseURLEnvVar: "STEPFUN_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "tencent-tokenhub", Aliases: []string{"tencent", "tencent-cloud", "tencentmaas", "tokenhub"}, ModelsDevID: "", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"TOKENHUB_API_KEY"}, BaseURLOverride: "https://tokenhub.tencentmaas.com/v1", BaseURLEnvVar: "TOKENHUB_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "togetherai", Aliases: nil, ModelsDevID: "togetherai", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: nil, BaseURLOverride: "", BaseURLEnvVar: "", ImplementationStatus: ProviderRowBacked},
		{ID: "vercel", Aliases: []string{"ai-gateway", "aigateway", "vercel-ai-gateway"}, ModelsDevID: "vercel", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: true, EnvVars: []string{"AI_GATEWAY_API_KEY"}, BaseURLOverride: "https://ai-gateway.vercel.sh/v1", BaseURLEnvVar: "AI_GATEWAY_BASE_URL", ImplementationStatus: ProviderRowBacked},
		{ID: "xai", Aliases: []string{"grok", "x-ai", "x.ai"}, ModelsDevID: "xai", TransportFamily: "codex_responses", AuthType: "api_key", Aggregator: false, EnvVars: []string{"XAI_API_KEY"}, BaseURLOverride: "https://api.x.ai/v1", BaseURLEnvVar: "XAI_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "xiaomi", Aliases: []string{"mimo", "xiaomi-mimo"}, ModelsDevID: "xiaomi", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"XIAOMI_API_KEY"}, BaseURLOverride: "https://api.xiaomimimo.com/v1", BaseURLEnvVar: "XIAOMI_BASE_URL", ImplementationStatus: ProviderImplemented},
		{ID: "zai", Aliases: []string{"glm", "z-ai", "z.ai", "zhipu"}, ModelsDevID: "zai", TransportFamily: "openai_chat", AuthType: "api_key", Aggregator: false, EnvVars: []string{"GLM_API_KEY", "ZAI_API_KEY", "Z_AI_API_KEY"}, BaseURLOverride: "https://api.z.ai/api/paas/v4", BaseURLEnvVar: "GLM_BASE_URL", ImplementationStatus: ProviderImplemented},
	}
	return append([]ProviderManifestEntry(nil), entries...)
}

func ResolveProviderManifestEntry(provider string) (ProviderManifestEntry, bool) {
	want := normalizeProviderManifestID(provider)
	if want == "" {
		return ProviderManifestEntry{}, false
	}
	for _, entry := range HermesProviderRegistryManifest() {
		if entry.ID == want {
			return entry, true
		}
		for _, alias := range entry.Aliases {
			if alias == want {
				return entry, true
			}
		}
	}
	return ProviderManifestEntry{}, false
}

func normalizeProviderManifestID(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}
