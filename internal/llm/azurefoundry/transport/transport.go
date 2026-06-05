package transport

// Transport names the request shape an Azure Foundry endpoint expects.
//
// Mirrors the api_mode classifications in
// hermes_cli/azure_detect.py: "chat_completions" / "anthropic_messages" /
// unknown.
type Transport string

const (
	Unknown        Transport = "unknown"
	OpenAI         Transport = "openai_chat_completions"
	Anthropic      Transport = "anthropic_messages"
	CodexResponses Transport = "codex_responses"
)
