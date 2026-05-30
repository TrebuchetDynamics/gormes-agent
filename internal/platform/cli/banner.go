package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display"

// BannerVersion is the pure input model for startup banner version text.
type BannerVersion = display.BannerVersion

// BannerGitState captures the already-resolved git facts shown in the banner.
type BannerGitState = display.BannerGitState

// FormatContextLength formats a model context window using Hermes' compact
// K/M display rules.
func FormatContextLength(tokens int) string { return display.FormatContextLength(tokens) }

// DisplayToolsetName normalizes internal and legacy toolset names for display.
func DisplayToolsetName(toolsetName string) string { return display.DisplayToolsetName(toolsetName) }

// FormatBannerVersionLabel returns the deterministic version label used in the
// CLI startup banner. Git state is injected by callers so this helper remains
// file, subprocess, and network inert.
func FormatBannerVersionLabel(version BannerVersion) string {
	return display.FormatBannerVersionLabel(version)
}
