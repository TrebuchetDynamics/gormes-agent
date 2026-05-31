package azurefoundry

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/transport"

// AzureTransport names the request shape an Azure Foundry endpoint expects.
//
// Mirrors the api_mode classifications in
// hermes_cli/azure_detect.py: "chat_completions" / "anthropic_messages" /
// unknown.
type AzureTransport = transport.Transport

const (
	AzureTransportUnknown        AzureTransport = transport.Unknown
	AzureTransportOpenAI         AzureTransport = transport.OpenAI
	AzureTransportAnthropic      AzureTransport = transport.Anthropic
	AzureTransportCodexResponses AzureTransport = transport.CodexResponses
)
