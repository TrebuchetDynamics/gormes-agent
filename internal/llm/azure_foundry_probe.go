package llm

import (
	"context"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry"
)

// DetectAzureFoundry classifies an Azure Foundry endpoint into a transport
// read model from deterministic inputs only.
func DetectAzureFoundry(ctx context.Context, client *http.Client, base, apiKey string) (AzureProbeResult, error) {
	return azurefoundry.DetectAzureFoundry(ctx, client, base, apiKey)
}
