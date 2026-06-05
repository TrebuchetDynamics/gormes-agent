package azurefoundry

import "github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/pathsniff"

// ClassifyAzurePath inspects rawURL and returns AzureTransportAnthropic when
// the URL's path equals "/anthropic", ends with "/anthropic", or contains a
// "/anthropic/" segment (case-insensitive). It returns AzureTransportUnknown
// for empty paths, parse errors, and every other shape — including OpenAI
// deployment paths.
//
// Mirrors hermes_cli/azure_detect.py:_looks_like_anthropic_path. Pure URL
// inspection: never opens HTTP, reads env or config, writes files, or starts
// goroutines.
func ClassifyAzurePath(rawURL string) AzureTransport {
	return pathsniff.Classify(rawURL)
}
