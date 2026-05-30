package tui

import (
	"os"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/browser"
)

const defaultBrowserCDPURL = browser.DefaultCDPURL

func browserSlashHandler(input string, _ *Model) SlashResult {
	return SlashResult{Handled: true, StatusMessage: browser.HandleSlash(input, os.Getenv, os.Setenv)}
}

func browserStatusMessage() string {
	return browser.StatusMessage(browserCDPURLFromEnv())
}

func browserCDPURLFromEnv() string {
	return browser.CDPURLFromEnv(os.Getenv)
}

func validateBrowserCDPURL(endpoint string) error {
	return browser.ValidateCDPURL(endpoint)
}
