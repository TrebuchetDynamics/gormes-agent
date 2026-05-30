package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"

const hermesMinimalChromeWidth = skin.MinimalChromeWidth

// HermesSkin is the Go-native subset of Hermes prompt_toolkit skin data that
// the Bubble Tea renderer needs before full YAML skin loading exists.
type HermesSkin = skin.HermesSkin

// HermesSkinColors mirrors the default tokens consumed by Hermes' TUI style
// overrides. Tokens stay as hex strings so later renderers can choose the
// appropriate terminal style backend without reparsing config.
type HermesSkinColors = skin.HermesSkinColors

// DefaultHermesSkin returns the built-in Hermes "default" skin tokens used by
// cli.py/hermes_cli/skin_engine.py, with the response label rebranded for the
// Gormes product identity.
func DefaultHermesSkin() HermesSkin { return skin.DefaultHermesSkin() }

// DefaultToolEmojis returns the per-tool emoji map matching Hermes' display.py
// get_tool_emoji mappings. Skin overrides can replace individual entries.
func DefaultToolEmojis() map[string]string { return skin.DefaultToolEmojis() }

// ResolveBuiltinSkin returns a built-in skin by name. Empty resolves to the
// default skin; names are case-insensitive and trimmed.
func ResolveBuiltinSkin(name string) (HermesSkin, bool) { return skin.ResolveBuiltinSkin(name) }

// BuiltinSkins returns all built-in skin definitions keyed by name.
func BuiltinSkins() map[string]HermesSkin { return skin.BuiltinSkins() }
