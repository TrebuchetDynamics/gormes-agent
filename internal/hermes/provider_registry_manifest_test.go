package hermes

import "testing"

var upstreamOverlayProviders = []string{
	"alibaba",
	"alibaba-coding-plan",
	"anthropic",
	"arcee",
	"azure-foundry",
	"bedrock",
	"copilot-acp",
	"deepseek",
	"github-copilot",
	"gmi",
	"google-gemini-cli",
	"huggingface",
	"kilo",
	"kimi-for-coding",
	"lmstudio",
	"minimax",
	"minimax-cn",
	"nous",
	"nvidia",
	"ollama-cloud",
	"openai-codex",
	"opencode",
	"opencode-go",
	"openrouter",
	"qwen-oauth",
	"stepfun",
	"tencent-tokenhub",
	"vercel",
	"xai",
	"xiaomi",
	"zai",
}

var upstreamModelsDevProviders = []string{
	"ai-gateway",
	"alibaba",
	"anthropic",
	"cohere",
	"copilot",
	"deepseek",
	"fireworks",
	"gemini",
	"google",
	"groq",
	"huggingface",
	"kilocode",
	"kimi-coding",
	"kimi-coding-cn",
	"minimax",
	"minimax-cn",
	"mistral",
	"nvidia",
	"ollama-cloud",
	"openai",
	"openai-codex",
	"opencode-go",
	"opencode-zen",
	"openrouter",
	"perplexity",
	"qwen-oauth",
	"stepfun",
	"togetherai",
	"xai",
	"xiaomi",
	"zai",
}

var upstreamProviderPrefixes = []string{
	"ai-gateway",
	"alibaba",
	"aliyun",
	"anthropic",
	"arcee",
	"arcee-ai",
	"arceeai",
	"claude",
	"copilot",
	"copilot-acp",
	"custom",
	"dashscope",
	"deep-seek",
	"deepseek",
	"gemini",
	"github",
	"github-copilot",
	"github-models",
	"glm",
	"gmi",
	"gmi-cloud",
	"gmicloud",
	"go",
	"google",
	"google-ai-studio",
	"google-gemini",
	"grok",
	"kilo",
	"kilocode",
	"kimi",
	"kimi-cn",
	"kimi-coding",
	"kimi-coding-cn",
	"local",
	"mimo",
	"minimax",
	"minimax-cn",
	"moonshot",
	"moonshot-cn",
	"nemotron",
	"nim",
	"nous",
	"nvidia",
	"nvidia-nim",
	"ollama",
	"ollama-cloud",
	"openai-codex",
	"opencode",
	"opencode-go",
	"opencode-zen",
	"openrouter",
	"qwen",
	"qwen-oauth",
	"qwen-portal",
	"stepfun",
	"stepfun",
	"tencent",
	"tencent-cloud",
	"tencent-tokenhub",
	"tencentmaas",
	"tokenhub",
	"vercel",
	"x-ai",
	"x.ai",
	"xai",
	"xiaomi",
	"xiaomi-mimo",
	"z-ai",
	"z.ai",
	"zai",
	"zen",
	"zhipu",
}

var upstreamAuthProviders = []string{
	"ai-gateway",
	"alibaba",
	"alibaba-coding-plan",
	"anthropic",
	"arcee",
	"azure-foundry",
	"bedrock",
	"copilot",
	"copilot-acp",
	"deepseek",
	"gemini",
	"gmi",
	"google-gemini-cli",
	"huggingface",
	"kilocode",
	"kimi-coding",
	"kimi-coding-cn",
	"lmstudio",
	"minimax",
	"minimax-cn",
	"nous",
	"nvidia",
	"ollama-cloud",
	"openai-codex",
	"opencode-go",
	"opencode-zen",
	"qwen-oauth",
	"stepfun",
	"tencent-tokenhub",
	"xai",
	"xiaomi",
	"zai",
}

var upstreamProviderModelProviders = []string{
	"alibaba",
	"anthropic",
	"arcee",
	"azure-foundry",
	"bedrock",
	"copilot",
	"copilot-acp",
	"deepseek",
	"gemini",
	"gmi",
	"google-gemini-cli",
	"huggingface",
	"kilocode",
	"kimi-coding",
	"kimi-coding-cn",
	"minimax",
	"minimax-cn",
	"moonshot",
	"nous",
	"nvidia",
	"openai",
	"openai-codex",
	"opencode-go",
	"opencode-zen",
	"stepfun",
	"tencent-tokenhub",
	"xai",
	"xiaomi",
	"zai",
}

var upstreamProviderAliases = map[string]string{
	"ai-gateway":          "vercel",
	"aigateway":           "vercel",
	"alibaba-cloud":       "alibaba",
	"alibaba-coding":      "alibaba-coding-plan",
	"alibaba_coding":      "alibaba-coding-plan",
	"alibaba_coding_plan": "alibaba-coding-plan",
	"aliyun":              "alibaba",
	"amazon":              "bedrock",
	"amazon-bedrock":      "bedrock",
	"arcee-ai":            "arcee",
	"arceeai":             "arcee",
	"aws":                 "bedrock",
	"aws-bedrock":         "bedrock",
	"build-nvidia":        "nvidia",
	"claude":              "anthropic",
	"claude-code":         "anthropic",
	"copilot":             "github-copilot",
	"dashscope":           "alibaba",
	"deep-seek":           "deepseek",
	"gemini-cli":          "google-gemini-cli",
	"gemini-oauth":        "google-gemini-cli",
	"github":              "github-copilot",
	"github-copilot-acp":  "copilot-acp",
	"glm":                 "zai",
	"gmi-cloud":           "gmi",
	"gmicloud":            "gmi",
	"go":                  "opencode-go",
	"grok":                "xai",
	"hf":                  "huggingface",
	"hugging-face":        "huggingface",
	"huggingface-hub":     "huggingface",
	"kilo-code":           "kilo",
	"kilo-gateway":        "kilo",
	"kilocode":            "kilo",
	"kimi":                "kimi-for-coding",
	"kimi-coding":         "kimi-for-coding",
	"kimi-coding-cn":      "kimi-for-coding",
	"llama-cpp":           "local",
	"llama.cpp":           "local",
	"llamacpp":            "local",
	"lm-studio":           "lmstudio",
	"lm_studio":           "lmstudio",
	"lmstudio":            "lmstudio",
	"mimo":                "xiaomi",
	"minimax-china":       "minimax-cn",
	"minimax_cn":          "minimax-cn",
	"moonshot":            "kimi-for-coding",
	"nemotron":            "nvidia",
	"nim":                 "nvidia",
	"nvidia-nim":          "nvidia",
	"ollama":              "custom",
	"openai":              "openrouter",
	"opencode-go-sub":     "opencode-go",
	"opencode-zen":        "opencode",
	"qwen":                "alibaba",
	"step":                "stepfun",
	"stepfun-coding-plan": "stepfun",
	"tencent":             "tencent-tokenhub",
	"tencent-cloud":       "tencent-tokenhub",
	"tencentmaas":         "tencent-tokenhub",
	"tokenhub":            "tencent-tokenhub",
	"vercel-ai-gateway":   "vercel",
	"vllm":                "local",
	"x-ai":                "xai",
	"x.ai":                "xai",
	"xiaomi-mimo":         "xiaomi",
	"z-ai":                "zai",
	"z.ai":                "zai",
	"zen":                 "opencode",
	"zhipu":               "zai",
}

func TestHermesProviderRegistryManifestCoversUpstream(t *testing.T) {
	entries := HermesProviderRegistryManifest()
	if len(entries) == 0 {
		t.Fatal("provider manifest is empty")
	}
	seen := map[string]ProviderManifestEntry{}
	for _, entry := range entries {
		if entry.ID == "" {
			t.Fatal("provider manifest entry has empty ID")
		}
		if _, dup := seen[entry.ID]; dup {
			t.Fatalf("duplicate provider manifest ID %q", entry.ID)
		}
		seen[entry.ID] = entry
		if entry.TransportFamily == "" {
			t.Fatalf("%s TransportFamily is empty", entry.ID)
		}
		if entry.AuthType == "" {
			t.Fatalf("%s AuthType is empty", entry.ID)
		}
		switch entry.ImplementationStatus {
		case ProviderImplemented, ProviderOwned, ProviderRowBacked, ProviderExcluded:
		default:
			t.Fatalf("%s has invalid ImplementationStatus %q", entry.ID, entry.ImplementationStatus)
		}
	}
	for _, provider := range upstreamOverlayProviders {
		entry, ok := ResolveProviderManifestEntry(provider)
		if !ok {
			t.Fatalf("HERMES_OVERLAYS provider %q is not classified", provider)
		}
		if entry.ID == "" {
			t.Fatalf("HERMES_OVERLAYS provider %q resolved to empty entry", provider)
		}
	}
	for provider := range upstreamProviderAliases {
		if _, ok := ResolveProviderManifestEntry(provider); !ok {
			t.Fatalf("ALIASES key %q is not classified", provider)
		}
	}
	for provider := range upstreamProviderAliases {
		canonical := upstreamProviderAliases[provider]
		entry, ok := ResolveProviderManifestEntry(provider)
		if !ok {
			t.Fatalf("alias %q missing", provider)
		}
		if entry.ID != canonical {
			t.Fatalf("alias %q resolved to %q, want %q", provider, entry.ID, canonical)
		}
	}
	for _, provider := range upstreamModelsDevProviders {
		if _, ok := ResolveProviderManifestEntry(provider); !ok {
			t.Fatalf("PROVIDER_TO_MODELS_DEV provider %q is not classified", provider)
		}
	}
	for _, provider := range upstreamProviderPrefixes {
		if _, ok := ResolveProviderManifestEntry(provider); !ok {
			t.Fatalf("_PROVIDER_PREFIXES provider %q is not classified", provider)
		}
	}
	for _, provider := range upstreamAuthProviders {
		if _, ok := ResolveProviderManifestEntry(provider); !ok {
			t.Fatalf("PROVIDER_REGISTRY provider %q is not classified", provider)
		}
	}
	for _, provider := range upstreamProviderModelProviders {
		if _, ok := ResolveProviderManifestEntry(provider); !ok {
			t.Fatalf("_PROVIDER_MODELS provider %q is not classified", provider)
		}
	}
}

func TestHermesProviderRegistryManifestRecordsRequiredMetadata(t *testing.T) {
	tests := []struct {
		provider, transport, auth, status string
		aggregator                        bool
	}{
		{"openai-codex", "codex_responses", "oauth_external", "implemented", false},
		{"qwen-oauth", "openai_chat", "oauth_external", "owned", false},
		{"google-gemini-cli", "openai_chat", "oauth_external", "owned", false},
		{"copilot-acp", "codex_responses", "external_process", "row_backed", false},
		{"openrouter", "openai_chat", "api_key", "implemented", true},
		{"opencode-zen", "openai_chat", "api_key", "row_backed", true},
		{"kilocode", "openai_chat", "api_key", "row_backed", true},
		{"aws", "bedrock_converse", "aws_sdk", "implemented", false},
	}
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			entry, ok := ResolveProviderManifestEntry(tt.provider)
			if !ok {
				t.Fatalf("ResolveProviderManifestEntry(%q) missing", tt.provider)
			}
			if entry.TransportFamily != tt.transport {
				t.Fatalf("TransportFamily = %q, want %q", entry.TransportFamily, tt.transport)
			}
			if entry.AuthType != tt.auth {
				t.Fatalf("AuthType = %q, want %q", entry.AuthType, tt.auth)
			}
			if string(entry.ImplementationStatus) != tt.status {
				t.Fatalf("ImplementationStatus = %q, want %q", entry.ImplementationStatus, tt.status)
			}
			if entry.Aggregator != tt.aggregator {
				t.Fatalf("Aggregator = %v, want %v", entry.Aggregator, tt.aggregator)
			}
		})
	}
}
