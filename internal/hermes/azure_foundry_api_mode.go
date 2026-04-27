package hermes

import "strings"

const AzureTransportCodexResponses AzureTransport = "codex_responses"

var azureFoundryResponsesPrefixes = []string{
	"codex",
	"gpt-5",
	"o1",
	"o3",
	"o4",
}

func AzureFoundryAPIModeForModel(modelName string) (AzureTransport, bool) {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if normalized == "" {
		return "", false
	}
	if slash := strings.LastIndex(normalized, "/"); slash >= 0 {
		normalized = strings.TrimSpace(normalized[slash+1:])
	}
	for _, prefix := range azureFoundryResponsesPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return AzureTransportCodexResponses, true
		}
	}
	return "", false
}
