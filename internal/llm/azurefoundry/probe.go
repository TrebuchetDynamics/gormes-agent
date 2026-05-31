package azurefoundry

import (
	"context"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/detection"
)

// DetectAzureFoundry classifies an Azure Foundry endpoint into a transport
// read model from deterministic inputs only: the base URL, an injected
// HTTP client, and an API key string. It never reads or writes
// AZURE_FOUNDRY_BASE_URL, AZURE_FOUNDRY_API_KEY, deployment config, or
// model context metadata.
//
// Decision order, mirroring hermes_cli/azure_detect.py:detect:
//
//  1. Path sniff. If ClassifyAzurePath reports AzureTransportAnthropic,
//     return Transport=anthropic_messages immediately without HTTP.
//  2. HTTP probes. Otherwise delegate to ProbeAzureFoundry, which probes
//     <base>/models for an OpenAI-shaped catalog and falls through to
//     <base>/v1/messages for an Anthropic Messages-shaped error.
//  3. Manual fallback. When neither classification fires, return
//     Transport=unknown with Reason="manual_required".
//
// Models, when present, are advisory only - the helper does not persist
// them. The only non-nil error returned is the caller's context error
// when ctx is cancelled or deadlined mid-probe.
func DetectAzureFoundry(ctx context.Context, client *http.Client, base, apiKey string) (AzureProbeResult, error) {
	return detection.Detect(ctx, client, base, apiKey)
}
