package skin

import "strings"

const MinimalChromeWidth = 64

// HermesSkin is the Go-native subset of Hermes prompt_toolkit skin data that
// the Bubble Tea renderer needs before full YAML skin loading exists.
type HermesSkin struct {
	Name          string
	ResponseLabel string
	PromptSymbol  string
	HelpHeader    string
	ToolPrefix    string
	Colors        HermesSkinColors
	ToolEmojis    map[string]string // per-tool emoji overrides
}

// HermesSkinColors mirrors the default tokens consumed by Hermes' TUI style
// overrides. Tokens stay as hex strings so later renderers can choose the
// appropriate terminal style backend without reparsing config.
type HermesSkinColors struct {
	// Banner colors (Python: banner_*)
	BannerBorder string
	BannerTitle  string
	BannerAccent string
	BannerDim    string
	BannerText   string

	// UI colors (Python: ui_*)
	UIAcent string
	UILabel string
	UIOk    string
	UIError string
	UIWarn  string

	// Input/Prompt
	Prompt         string
	Placeholder    string
	InputRule      string
	ResponseBorder string

	// Session
	SessionLabel  string
	SessionBorder string

	// Status bar
	StatusBarBackground string
	StatusBarText       string
	StatusBarStrong     string
	StatusBarDim        string
	StatusBarGood       string
	StatusBarWarn       string
	StatusBarBad        string
	StatusBarCritical   string
}

// DefaultHermesSkin returns the built-in Hermes "default" skin tokens used by
// cli.py/hermes_cli/skin_engine.py, with the response label rebranded for the
// Gormes product identity.
func DefaultHermesSkin() HermesSkin {
	return HermesSkin{
		Name:          "default",
		ResponseLabel: " ⚕ Gormes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "(^_^)? Available Commands",
		ToolPrefix:    "┊",
		Colors: HermesSkinColors{
			BannerBorder: "#CD7F32",
			BannerTitle:  "#FFD700",
			BannerAccent: "#FFBF00",
			BannerDim:    "#B8860B",
			BannerText:   "#FFF8DC",
			UIAcent:      "#FFBF00",
			UILabel:      "#DAA520",
			UIOk:         "#4caf50",
			UIError:      "#ef5350",
			UIWarn:       "#ffa726",

			Prompt:         "#FFF8DC",
			Placeholder:    "#B8860B",
			InputRule:      "#CD7F32",
			ResponseBorder: "#FFD700",

			SessionLabel:  "#DAA520",
			SessionBorder: "#8B8682",

			StatusBarBackground: "#1a1a2e",
			StatusBarText:       "#C0C0C0",
			StatusBarStrong:     "#FFD700",
			StatusBarDim:        "#8B8682",
			StatusBarGood:       "#8FBC8F",
			StatusBarWarn:       "#FFD700",
			StatusBarBad:        "#FF8C00",
			StatusBarCritical:   "#FF6B6B",
		},
		ToolEmojis: DefaultToolEmojis(),
	}
}

// DefaultResponseLabel returns the trimmed built-in response-region label for
// Hermes-compatible chrome that does not carry an explicit skin value.
func DefaultResponseLabel() string {
	return strings.TrimSpace(DefaultHermesSkin().ResponseLabel)
}

// DefaultToolEmojis returns the per-tool emoji map matching Hermes' display.py
// get_tool_emoji mappings. Skin overrides can replace individual entries.
func DefaultToolEmojis() map[string]string {
	return map[string]string{
		"web_search":         "🔍",
		"web_extract":        "📄",
		"web_crawl":          "🕸️",
		"terminal":           "💻",
		"process":            "⚙️",
		"read_file":          "📖",
		"write_file":         "✍️",
		"patch":              "🔧",
		"search_files":       "🔎",
		"browser_navigate":   "🌐",
		"browser_snapshot":   "📸",
		"browser_click":      "👆",
		"browser_type":       "⌨️",
		"browser_scroll":     "↕️",
		"browser_back":       "◀️",
		"browser_press":      "⌨️",
		"browser_get_images": "🖼️",
		"browser_vision":     "👁️",
		"todo":               "📋",
		"session_search":     "🔍",
		"memory":             "🧠",
		"skills_list":        "📚",
		"skill_view":         "📚",
		"image_generate":     "🎨",
		"text_to_speech":     "🔊",
		"vision_analyze":     "👁️",
		"mixture_of_agents":  "🧠",
		"send_message":       "📨",
		"cronjob":            "⏰",
		"execute_code":       "💻",
		"delegate_task":      "🔀",
	}
}

// ToolEmoji returns the display emoji for a tool, falling back to "⚡" if
// neither the skin nor the default map has an entry.
func (s HermesSkin) ToolEmoji(name string) string {
	if s.ToolEmojis != nil {
		if e, ok := s.ToolEmojis[name]; ok {
			return e
		}
	}
	if e, ok := DefaultToolEmojis()[name]; ok {
		return e
	}
	return "⚡"
}

// UseMinimalChrome reports whether Hermes hides low-value TUI chrome for the
// given terminal width. Hermes switches to minimal chrome below 64 columns.
func (s HermesSkin) UseMinimalChrome(width int) bool {
	return width < MinimalChromeWidth
}

// PromptSymbols returns the normal prompt and the suffix used by special prompt
// states such as approval, sudo, and active-agent prompts.
func (s HermesSkin) PromptSymbols(profileName string) (normalPrompt, stateSuffix string) {
	symbol := normalizePromptSymbol(s.PromptSymbol)
	profile := strings.TrimSpace(profileName)
	if profile != "" && profile != "default" && profile != "custom" {
		symbol = profile + " " + symbol
	}

	stripped := strings.TrimSpace(symbol)
	if stripped == "" {
		return "❯ ", "❯ "
	}
	parts := strings.Fields(stripped)
	candidate := ""
	if len(parts) > 0 {
		candidate = parts[len(parts)-1]
	}
	if strings.ContainsAny(candidate, "❯>$#›»→") {
		return symbol, strings.TrimRight(candidate, " \t\r\n") + " "
	}
	return symbol, symbol
}

func normalizePromptSymbol(symbol string) string {
	symbol = strings.TrimRight(symbol, " \t\r\n")
	if strings.TrimSpace(symbol) == "" {
		return "❯ "
	}
	return symbol + " "
}

// ResolveBuiltinSkin returns a built-in skin by name. Empty resolves to the
// default skin; names are case-insensitive and trimmed.
func ResolveBuiltinSkin(name string) (HermesSkin, bool) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		name = "default"
	}
	skin, ok := BuiltinSkins()[name]
	return skin, ok
}

// BuiltinSkins returns all built-in skin definitions keyed by name.
func BuiltinSkins() map[string]HermesSkin {
	return map[string]HermesSkin{
		"default":        DefaultHermesSkin(),
		"ares":           aresSkin(),
		"mono":           monoSkin(),
		"slate":          slateSkin(),
		"daylight":       daylightSkin(),
		"poseidon":       poseidonSkin(),
		"sisyphus":       sisyphusSkin(),
		"charizard":      charizardSkin(),
		"warm-lightmode": warmLightmodeSkin(),
	}
}

func aresSkin() HermesSkin {
	return HermesSkin{
		Name:          "ares",
		ResponseLabel: " ⚔ Gormes ",
		PromptSymbol:  "⚔ ",
		HelpHeader:    "(⚔) Available Commands",
		ToolPrefix:    "╎",
		Colors: HermesSkinColors{
			BannerBorder: "#9F1C1C", BannerTitle: "#C7A96B", BannerAccent: "#DD4A3A",
			BannerDim: "#6B1717", BannerText: "#F1E6CF",
			UIAcent: "#DD4A3A", UILabel: "#C7A96B", UIOk: "#4caf50", UIError: "#ef5350", UIWarn: "#ffa726",
			Prompt: "#F1E6CF", Placeholder: "#6B1717", InputRule: "#9F1C1C", ResponseBorder: "#C7A96B",
			SessionLabel: "#C7A96B", SessionBorder: "#6E584B",
			StatusBarBackground: "#2A1212", StatusBarText: "#F1E6CF", StatusBarStrong: "#C7A96B",
			StatusBarDim: "#6E584B", StatusBarGood: "#7BC96F", StatusBarWarn: "#C7A96B",
			StatusBarBad: "#DD4A3A", StatusBarCritical: "#EF5350",
		},
	}
}

func monoSkin() HermesSkin {
	return HermesSkin{
		Name:          "mono",
		ResponseLabel: " ⚕ Gormes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "[?] Available Commands",
		ToolPrefix:    "┊",
		Colors: HermesSkinColors{
			BannerBorder: "#555555", BannerTitle: "#e6edf3", BannerAccent: "#aaaaaa",
			BannerDim: "#444444", BannerText: "#c9d1d9",
			UIAcent: "#aaaaaa", UILabel: "#888888", UIOk: "#888888", UIError: "#cccccc", UIWarn: "#999999",
			Prompt: "#c9d1d9", Placeholder: "#444444", InputRule: "#444444", ResponseBorder: "#aaaaaa",
			SessionLabel: "#888888", SessionBorder: "#555555",
			StatusBarBackground: "#1F1F1F", StatusBarText: "#C9D1D9", StatusBarStrong: "#E6EDF3",
			StatusBarDim: "#777777", StatusBarGood: "#B5B5B5", StatusBarWarn: "#AAAAAA",
			StatusBarBad: "#D0D0D0", StatusBarCritical: "#F0F0F0",
		},
	}
}

func slateSkin() HermesSkin {
	return HermesSkin{
		Name:          "slate",
		ResponseLabel: " ⚕ Gormes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "(^_^)? Available Commands",
		ToolPrefix:    "┊",
		Colors: HermesSkinColors{
			BannerBorder: "#4169e1", BannerTitle: "#7eb8f6", BannerAccent: "#8EA8FF",
			BannerDim: "#4b5563", BannerText: "#c9d1d9",
			UIAcent: "#7eb8f6", UILabel: "#8EA8FF", UIOk: "#63D0A6", UIError: "#F7A072", UIWarn: "#e6a855",
			Prompt: "#c9d1d9", Placeholder: "#4b5563", InputRule: "#4169e1", ResponseBorder: "#7eb8f6",
			SessionLabel: "#7eb8f6", SessionBorder: "#4b5563",
			StatusBarBackground: "#151C2F", StatusBarText: "#C9D1D9", StatusBarStrong: "#7EB8F6",
			StatusBarDim: "#4B5563", StatusBarGood: "#63D0A6", StatusBarWarn: "#E6A855",
			StatusBarBad: "#F7A072", StatusBarCritical: "#FF7A7A",
		},
	}
}

func daylightSkin() HermesSkin {
	return HermesSkin{
		Name:          "daylight",
		ResponseLabel: " ⚕ Gormes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "[?] Available Commands",
		ToolPrefix:    "│",
		Colors: HermesSkinColors{
			BannerBorder: "#2563EB", BannerTitle: "#0F172A", BannerAccent: "#1D4ED8",
			BannerDim: "#475569", BannerText: "#111827",
			UIAcent: "#2563EB", UILabel: "#0F766E", UIOk: "#15803D", UIError: "#B91C1C", UIWarn: "#B45309",
			Prompt: "#111827", Placeholder: "#475569", InputRule: "#93C5FD", ResponseBorder: "#2563EB",
			SessionLabel: "#1D4ED8", SessionBorder: "#64748B",
			StatusBarBackground: "#E5EDF8", StatusBarText: "#111827", StatusBarStrong: "#0F172A",
			StatusBarDim: "#64748B", StatusBarGood: "#15803D", StatusBarWarn: "#B45309",
			StatusBarBad: "#D84315", StatusBarCritical: "#B91C1C",
		},
	}
}

func poseidonSkin() HermesSkin {
	return HermesSkin{
		Name:          "poseidon",
		ResponseLabel: " Ψ Gormes ",
		PromptSymbol:  "Ψ ",
		HelpHeader:    "(Ψ) Available Commands",
		ToolPrefix:    "│",
		Colors: HermesSkinColors{
			BannerBorder: "#2A6FB9", BannerTitle: "#A9DFFF", BannerAccent: "#5DB8F5",
			BannerDim: "#153C73", BannerText: "#EAF7FF",
			UIAcent: "#5DB8F5", UILabel: "#A9DFFF", UIOk: "#4caf50", UIError: "#ef5350", UIWarn: "#ffa726",
			Prompt: "#EAF7FF", Placeholder: "#153C73", InputRule: "#2A6FB9", ResponseBorder: "#5DB8F5",
			SessionLabel: "#A9DFFF", SessionBorder: "#496884",
			StatusBarBackground: "#0F2440", StatusBarText: "#EAF7FF", StatusBarStrong: "#A9DFFF",
			StatusBarDim: "#496884", StatusBarGood: "#6ED7B0", StatusBarWarn: "#5DB8F5",
			StatusBarBad: "#2A6FB9", StatusBarCritical: "#D94F4F",
		},
	}
}

func sisyphusSkin() HermesSkin {
	return HermesSkin{
		Name:          "sisyphus",
		ResponseLabel: " ◉ Gormes ",
		PromptSymbol:  "◉ ",
		HelpHeader:    "(◉) Available Commands",
		ToolPrefix:    "│",
		Colors: HermesSkinColors{
			BannerBorder: "#B7B7B7", BannerTitle: "#F5F5F5", BannerAccent: "#E7E7E7",
			BannerDim: "#4A4A4A", BannerText: "#D3D3D3",
			UIAcent: "#E7E7E7", UILabel: "#D3D3D3", UIOk: "#919191", UIError: "#E7E7E7", UIWarn: "#B7B7B7",
			Prompt: "#F5F5F5", Placeholder: "#4A4A4A", InputRule: "#656565", ResponseBorder: "#B7B7B7",
			SessionLabel: "#919191", SessionBorder: "#656565",
			StatusBarBackground: "#202020", StatusBarText: "#D3D3D3", StatusBarStrong: "#F5F5F5",
			StatusBarDim: "#656565", StatusBarGood: "#B7B7B7", StatusBarWarn: "#D3D3D3",
			StatusBarBad: "#E7E7E7", StatusBarCritical: "#F5F5F5",
		},
	}
}

func charizardSkin() HermesSkin {
	return HermesSkin{
		Name:          "charizard",
		ResponseLabel: " ✦ Gormes ",
		PromptSymbol:  "✦ ",
		HelpHeader:    "(✦) Available Commands",
		ToolPrefix:    "│",
		Colors: HermesSkinColors{
			BannerBorder: "#C75B1D", BannerTitle: "#FFD39A", BannerAccent: "#F29C38",
			BannerDim: "#7A3511", BannerText: "#FFF0D4",
			UIAcent: "#F29C38", UILabel: "#FFD39A", UIOk: "#4caf50", UIError: "#ef5350", UIWarn: "#ffa726",
			Prompt: "#FFF0D4", Placeholder: "#7A3511", InputRule: "#C75B1D", ResponseBorder: "#F29C38",
			SessionLabel: "#FFD39A", SessionBorder: "#6C4724",
			StatusBarBackground: "#2B160E", StatusBarText: "#FFF0D4", StatusBarStrong: "#FFD39A",
			StatusBarDim: "#6C4724", StatusBarGood: "#6BCB77", StatusBarWarn: "#F29C38",
			StatusBarBad: "#E2832B", StatusBarCritical: "#EF5350",
		},
	}
}

func warmLightmodeSkin() HermesSkin {
	return HermesSkin{
		Name:          "warm-lightmode",
		ResponseLabel: " ⚕ Gormes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "(^_^)? Available Commands",
		ToolPrefix:    "│",
		Colors: HermesSkinColors{
			BannerBorder: "#8B6914", BannerTitle: "#5C3D11", BannerAccent: "#8B4513",
			BannerDim: "#8B7355", BannerText: "#2C1810",
			UIAcent: "#8B4513", UILabel: "#5C3D11", UIOk: "#2E7D32", UIError: "#C62828", UIWarn: "#E65100",
			Prompt: "#2C1810", Placeholder: "#8B7355", InputRule: "#8B6914", ResponseBorder: "#8B6914",
			SessionLabel: "#5C3D11", SessionBorder: "#A0845C",
			StatusBarBackground: "#F5F0E8", StatusBarText: "#2C1810", StatusBarStrong: "#5C3D11",
			StatusBarDim: "#A0845C", StatusBarGood: "#2E7D32", StatusBarWarn: "#E65100",
			StatusBarBad: "#C62828", StatusBarCritical: "#B71C1C",
		},
	}
}
