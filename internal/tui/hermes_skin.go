package tui

import "strings"

const hermesMinimalChromeWidth = 64

// HermesSkin is the Go-native subset of Hermes prompt_toolkit skin data that
// the Bubble Tea renderer needs before full YAML skin loading exists.
type HermesSkin struct {
	Name          string
	ResponseLabel string
	PromptSymbol  string
	HelpHeader    string
	ToolPrefix    string
	Colors        HermesSkinColors
}

// HermesSkinColors mirrors the default tokens consumed by Hermes' TUI style
// overrides. Tokens stay as hex strings so later renderers can choose the
// appropriate terminal style backend without reparsing config.
type HermesSkinColors struct {
	Prompt              string
	Placeholder         string
	InputRule           string
	ResponseBorder      string
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
// cli.py/hermes_cli/skin_engine.py for the prompt_toolkit TUI.
func DefaultHermesSkin() HermesSkin {
	return HermesSkin{
		Name:          "default",
		ResponseLabel: " ⚕ Hermes ",
		PromptSymbol:  "❯ ",
		HelpHeader:    "(^_^)? Available Commands",
		ToolPrefix:    "┊",
		Colors: HermesSkinColors{
			Prompt:              "#FFF8DC",
			Placeholder:         "#B8860B",
			InputRule:           "#CD7F32",
			ResponseBorder:      "#FFD700",
			StatusBarBackground: "#1a1a2e",
			StatusBarText:       "#FFF8DC",
			StatusBarStrong:     "#FFD700",
			StatusBarDim:        "#B8860B",
			StatusBarGood:       "#4caf50",
			StatusBarWarn:       "#ffa726",
			StatusBarBad:        "#FFBF00",
			StatusBarCritical:   "#ef5350",
		},
	}
}

// UseMinimalChrome reports whether Hermes hides low-value TUI chrome for the
// given terminal width. Hermes switches to minimal chrome below 64 columns.
func (s HermesSkin) UseMinimalChrome(width int) bool {
	return width < hermesMinimalChromeWidth
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
