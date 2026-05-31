package apimode

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/transport"
)

var responsesPrefixes = []string{
	"codex",
	"gpt-5",
	"o1",
	"o3",
	"o4",
}

// ForModel returns an Azure Foundry transport override for model families
// that require a non-default API surface.
func ForModel(modelName string) (transport.Transport, bool) {
	normalized := strings.ToLower(strings.TrimSpace(modelName))
	if normalized == "" {
		return "", false
	}
	if slash := strings.LastIndex(normalized, "/"); slash >= 0 {
		normalized = strings.TrimSpace(normalized[slash+1:])
	}
	for _, prefix := range responsesPrefixes {
		if strings.HasPrefix(normalized, prefix) {
			return transport.CodexResponses, true
		}
	}
	return "", false
}
