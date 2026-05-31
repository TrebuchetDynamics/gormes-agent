package azurefoundry

import (
	"context"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/probehttp"
)

// AzureProbeResult is what ProbeAzureFoundry returns to the caller.
//
// Transport is the classification verdict. Models is populated only when
// the /models probe returned an OpenAI-shaped catalog (possibly empty).
// Reason is a short human summary suitable for display in the wizard.
// Evidence captures the URLs probed, status codes, and any model IDs
// surfaced. The wizard renders these to the operator so they can audit
// the auto-detection result.
type AzureProbeResult = probehttp.Result

// ProbeAzureFoundry classifies an Azure Foundry endpoint by issuing fixed
// OpenAI models and Anthropic messages probes. The helper never writes to
// disk, never mutates configuration, and never retries beyond these probes.
// The only non-nil error returned is the caller's context error when ctx is
// cancelled or deadlined mid-probe.
func ProbeAzureFoundry(ctx context.Context, client *http.Client, base, apiKey string) (AzureProbeResult, error) {
	return probehttp.Probe(ctx, client, base, apiKey)
}
