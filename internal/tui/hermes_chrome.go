package tui

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/chrome"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
)

// HermesChromeInput is the pure input to RenderHermesChrome. The TUI renderer
// pre-builds each section (conversation tail, optional activity/hint row,
// status rule, prompt+input area, optional voice/image/completion rows) and
// asks this helper to assemble the bottom-pinned ordering Hermes' Ink TUI
// renders in ui-tui/src/components/appLayout.tsx.
type HermesChromeInput = chrome.Input

// RenderHermesChrome assembles the bottom-pinned chrome stack used by
// Hermes' Ink frontend. Layout order matches ComposerPane in
// ui-tui/src/components/appLayout.tsx. All sections are caller-rendered
// strings; this helper only picks order and drops empty optional rows.
func RenderHermesChrome(in HermesChromeInput) string {
	return chrome.Render(in)
}

// HermesChromeUseAltScreen reports whether the bottom-pinned Hermes chrome
// should be rendered in the terminal alt-screen. Upstream Hermes' current Ink
// TUI uses an alternate screen by default and only skips it for explicit inline
// mode. Gormes' Bubble Tea surface is likewise a full-screen renderer; normal
// scrollback leaves stale frame fragments visible after render ticks.
func trimTrailingLineWhitespace(s string) string {
	return chrome.TrimTrailingLineWhitespace(s)
}

func HermesChromeUseAltScreen() bool {
	return chrome.UseAltScreen()
}

// HermesChromeAssistantLabel returns the response-region label rendered above
// assistant output. It keeps Hermes' response-box shape while using Gormes'
// product label.
func HermesChromeAssistantLabel() string {
	return skin.DefaultResponseLabel()
}
