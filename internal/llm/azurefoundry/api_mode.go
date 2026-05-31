package azurefoundry

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/apimode"

func AzureFoundryAPIModeForModel(modelName string) (AzureTransport, bool) {
	return apimode.ForModel(modelName)
}
