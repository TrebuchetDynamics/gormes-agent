package detection

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/pathsniff"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/probehttp"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/transport"
)

// Detect classifies an Azure Foundry endpoint into a transport read model from
// deterministic inputs only: the base URL, an injected HTTP client, and an API
// key string.
func Detect(ctx context.Context, client *http.Client, base, apiKey string) (probehttp.Result, error) {
	if sniffed := pathsniff.Classify(base); sniffed == transport.Anthropic {
		return probehttp.Result{
			Transport: transport.Anthropic,
			Reason:    "URL path sniff matched /anthropic - Anthropic Messages API",
			Evidence:  []string{pathSniffEvidence(base)},
		}, nil
	}

	res, err := probehttp.Probe(ctx, client, base, apiKey)
	if err != nil {
		return probehttp.Result{}, err
	}
	return res, nil
}

func pathSniffEvidence(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil {
		return "azure_path_sniff: anthropic"
	}
	scheme := parsed.Scheme
	host := parsed.Host
	path := strings.TrimRight(parsed.Path, "/")
	return fmt.Sprintf("azure_path_sniff: scheme=%s host=%s path=%s -> anthropic_messages", scheme, host, path)
}
