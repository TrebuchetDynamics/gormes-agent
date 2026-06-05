package pathsniff

import (
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/azurefoundry/transport"
)

// Classify inspects rawURL and returns Anthropic when the URL's path equals
// "/anthropic", ends with "/anthropic", or contains a "/anthropic/" segment
// (case-insensitive). It returns Unknown for empty paths, parse errors, and
// every other shape — including OpenAI deployment paths.
func Classify(rawURL string) transport.Transport {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed == nil || parsed.Path == "" {
		return transport.Unknown
	}
	path := strings.TrimRight(strings.ToLower(parsed.Path), "/")
	if path == "/anthropic" || strings.HasSuffix(path, "/anthropic") || strings.Contains(path, "/anthropic/") {
		return transport.Anthropic
	}
	return transport.Unknown
}
