package skin

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin/tokens"

const MinimalChromeWidth = tokens.MinimalChromeWidth

type HermesSkin = tokens.HermesSkin
type HermesSkinColors = tokens.HermesSkinColors

func DefaultHermesSkin() HermesSkin { return tokens.DefaultHermesSkin() }

func DefaultResponseLabel() string { return tokens.DefaultResponseLabel() }

func DefaultToolEmojis() map[string]string { return tokens.DefaultToolEmojis() }

func ResolveBuiltinSkin(name string) (HermesSkin, bool) { return tokens.ResolveBuiltinSkin(name) }

func BuiltinSkins() map[string]HermesSkin { return tokens.BuiltinSkins() }
