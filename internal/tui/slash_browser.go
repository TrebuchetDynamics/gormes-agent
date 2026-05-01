package tui

import (
	"net/url"
	"os"
	"strings"
)

const defaultBrowserCDPURL = "http://127.0.0.1:9222"

func browserSlashHandler(input string, _ *Model) SlashResult {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 1 || strings.EqualFold(fields[1], "status") {
		return SlashResult{Handled: true, StatusMessage: browserStatusMessage()}
	}
	switch strings.ToLower(fields[1]) {
	case "connect":
		if len(fields) < 3 {
			return SlashResult{Handled: true, StatusMessage: "browser: usage /browser connect <cdp-url>"}
		}
		endpoint := strings.TrimSpace(fields[2])
		if err := validateBrowserCDPURL(endpoint); err != nil {
			return SlashResult{Handled: true, StatusMessage: "browser: invalid CDP URL: " + err.Error()}
		}
		_ = os.Setenv("BROWSER_CDP_URL", endpoint)
		_ = os.Setenv("CHROME_REMOTE_DEBUGGING_URL", endpoint)
		return SlashResult{Handled: true, StatusMessage: "browser: connected " + endpoint}
	default:
		return SlashResult{Handled: true, StatusMessage: "browser: usage /browser status | /browser connect <cdp-url>"}
	}
}

func browserStatusMessage() string {
	endpoint := browserCDPURLFromEnv()
	if endpoint == "" {
		return "browser: not connected; start Chrome with --remote-debugging-port=9222 and run /browser connect " + defaultBrowserCDPURL
	}
	return "browser: connected " + endpoint
}

func browserCDPURLFromEnv() string {
	if endpoint := strings.TrimSpace(os.Getenv("BROWSER_CDP_URL")); endpoint != "" {
		return endpoint
	}
	return strings.TrimSpace(os.Getenv("CHROME_REMOTE_DEBUGGING_URL"))
}

func validateBrowserCDPURL(endpoint string) error {
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
