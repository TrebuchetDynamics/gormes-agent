package attachments

import (
	"net/url"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
)

// TrustedHost reports whether rawURL is an HTTPS Discord CDN/media URL.
func TrustedHost(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" {
		return false
	}
	switch strings.ToLower(parsed.Hostname()) {
	case "cdn.discordapp.com", "media.discordapp.net":
		return true
	default:
		return false
	}
}

// CleanMediaType strips parameters and normalizes a media type.
func CleanMediaType(mediaType string) string { return channelutil.CleanMediaType(mediaType) }

// SafeFileName returns a basename safe for local cache paths and evidence text.
func SafeFileName(fileName string) string { return channelutil.SafeFileName(fileName) }

// SafeToken returns a bounded token safe for a local cache directory name.
func SafeToken(s string) string { return channelutil.SafeTokenDefault(s, "discord") }
