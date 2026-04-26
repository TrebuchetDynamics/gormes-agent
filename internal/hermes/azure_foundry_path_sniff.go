package hermes

import (
	"net/url"
	"strings"
)

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
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil || parsed.Path == "" {
		return AzureTransportUnknown
	}
	path := strings.TrimRight(strings.ToLower(parsed.Path), "/")
	if path == "/anthropic" || strings.HasSuffix(path, "/anthropic") || strings.Contains(path, "/anthropic/") {
		return AzureTransportAnthropic
	}
	return AzureTransportUnknown
}
