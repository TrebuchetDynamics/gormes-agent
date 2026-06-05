package browser

import (
	"net/url"
	"strings"
)

const DefaultCDPURL = "http://127.0.0.1:9222"

type EnvGetter func(string) string

type EnvSetter func(string, string) error

// HandleSlash resolves /browser locally without coupling the command parsing
// and CDP URL validation rules to the root Bubble Tea model. Environment IO is
// injected so package tui can preserve its process-wide behavior while this
// package stays deterministic under test.
func HandleSlash(input string, getenv EnvGetter, setenv EnvSetter) string {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 1 || strings.EqualFold(fields[1], "status") {
		return StatusMessage(CDPURLFromEnv(getenv))
	}
	switch strings.ToLower(fields[1]) {
	case "connect":
		if len(fields) < 3 {
			return "browser: usage /browser connect <cdp-url>"
		}
		endpoint := strings.TrimSpace(fields[2])
		if err := ValidateCDPURL(endpoint); err != nil {
			return "browser: invalid CDP URL: " + err.Error()
		}
		if setenv != nil {
			_ = setenv("BROWSER_CDP_URL", endpoint)
			_ = setenv("CHROME_REMOTE_DEBUGGING_URL", endpoint)
		}
		return "browser: connected " + endpoint
	default:
		return "browser: usage /browser status | /browser connect <cdp-url>"
	}
}

func CDPURLFromEnv(getenv EnvGetter) string {
	if getenv == nil {
		return ""
	}
	if endpoint := strings.TrimSpace(getenv("BROWSER_CDP_URL")); endpoint != "" {
		return endpoint
	}
	return strings.TrimSpace(getenv("CHROME_REMOTE_DEBUGGING_URL"))
}

func StatusMessage(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "browser: not connected; start Chrome with --remote-debugging-port=9222 and run /browser connect " + DefaultCDPURL
	}
	return "browser: connected " + endpoint
}

func ValidateCDPURL(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	if parsed.Host == "" {
		return url.InvalidHostError("missing host")
	}
	switch parsed.Scheme {
	case "http", "https", "ws", "wss":
		return nil
	default:
		return url.InvalidHostError("scheme must be http, https, ws, or wss")
	}
}
