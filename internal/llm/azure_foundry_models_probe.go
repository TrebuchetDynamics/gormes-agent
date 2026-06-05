package llm

import (
	"context"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry"
)

// AzureTransport names the request shape an Azure Foundry endpoint expects.
type AzureTransport = azurefoundry.AzureTransport

const (
	AzureTransportUnknown   = azurefoundry.AzureTransportUnknown
	AzureTransportOpenAI    = azurefoundry.AzureTransportOpenAI
	AzureTransportAnthropic = azurefoundry.AzureTransportAnthropic
)

// AzureProbeResult is what ProbeAzureFoundry returns to the caller.
type AzureProbeResult = azurefoundry.AzureProbeResult

// ProbeAzureFoundry classifies an Azure Foundry endpoint by issuing fixed probes.
func ProbeAzureFoundry(ctx context.Context, client *http.Client, base, apiKey string) (AzureProbeResult, error) {
	return azurefoundry.ProbeAzureFoundry(ctx, client, base, apiKey)
}
