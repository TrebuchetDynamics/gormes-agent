package display

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/display/banner"

// BannerVersion is the pure input model for startup banner version text.
type BannerVersion = banner.Version

// BannerGitState captures the already-resolved git facts shown in the banner.
type BannerGitState = banner.GitState

// FormatContextLength formats a model context window using Hermes' compact
// K/M display rules.
func FormatContextLength(tokens int) string { return banner.FormatContextLength(tokens) }

// DisplayToolsetName normalizes internal and legacy toolset names for display.
func DisplayToolsetName(toolsetName string) string { return banner.DisplayToolsetName(toolsetName) }

// FormatBannerVersionLabel returns the deterministic version label used in the
// CLI startup banner. Git state is injected by callers so this helper remains
// file, subprocess, and network inert.
func FormatBannerVersionLabel(version BannerVersion) string {
	return banner.FormatVersionLabel(version)
}
