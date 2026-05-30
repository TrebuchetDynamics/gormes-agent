package llm

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry"

const AzureTransportCodexResponses AzureTransport = azurefoundry.AzureTransportCodexResponses

func AzureFoundryAPIModeForModel(modelName string) (AzureTransport, bool) {
	return azurefoundry.AzureFoundryAPIModeForModel(modelName)
}
