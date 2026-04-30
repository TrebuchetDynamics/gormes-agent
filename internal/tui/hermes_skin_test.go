package tui

import (
	"strings"
	"testing"
)

func TestHermesSkin_DefaultTokens(t *testing.T) {
	skin := DefaultHermesSkin()

	if skin.PromptSymbol != "❯ " {
		t.Fatalf("PromptSymbol = %q, want Hermes default prompt", skin.PromptSymbol)
	}
	if skin.ResponseLabel == "" || !strings.Contains(skin.ResponseLabel, "Gormes") {
		t.Fatalf("ResponseLabel = %q, want Gormes response label in Hermes-style chrome", skin.ResponseLabel)
	}
	if strings.Contains(skin.ResponseLabel, "Hermes") {
		t.Fatalf("ResponseLabel = %q, must not expose upstream Hermes product label", skin.ResponseLabel)
	}
	if skin.Colors.InputRule != "#CD7F32" {
		t.Fatalf("InputRule color = %q, want Hermes bronze", skin.Colors.InputRule)
	}

	wantStatus := map[string]string{
		"background": "#1a1a2e",
		"text":       "#C0C0C0",
		"strong":     "#FFD700",
		"dim":        "#8B8682",
	}
	gotStatus := map[string]string{
		"background": skin.Colors.StatusBarBackground,
		"text":       skin.Colors.StatusBarText,
		"strong":     skin.Colors.StatusBarStrong,
		"dim":        skin.Colors.StatusBarDim,
	}
	for name, want := range wantStatus {
		if got := gotStatus[name]; got != want {
			t.Fatalf("status %s color = %q, want %q", name, got, want)
		}
	}
}

func TestHermesSkin_MinimalChromeThreshold(t *testing.T) {
	skin := DefaultHermesSkin()

	for _, tc := range []struct {
		width int
		want  bool
	}{
		{width: 0, want: true},
		{width: 63, want: true},
		{width: 64, want: false},
		{width: 120, want: false},
	} {
		if got := skin.UseMinimalChrome(tc.width); got != tc.want {
			t.Fatalf("UseMinimalChrome(%d) = %v, want %v", tc.width, got, tc.want)
		}
	}
}

func TestHermesSkin_ProfilePromptPrefix(t *testing.T) {
	skin := DefaultHermesSkin()

	prompt, suffix := skin.PromptSymbols("research")
	if prompt != "research ❯ " {
		t.Fatalf("profile prompt = %q, want profile-prefixed Hermes prompt", prompt)
	}
	if suffix != "❯ " {
		t.Fatalf("state suffix = %q, want unchanged Hermes arrow suffix", suffix)
	}

	defaultPrompt, defaultSuffix := skin.PromptSymbols("default")
	if defaultPrompt != "❯ " || defaultSuffix != "❯ " {
		t.Fatalf("default profile prompt/suffix = %q/%q, want unprefixed Hermes arrow", defaultPrompt, defaultSuffix)
	}
}

func TestHermesSkin_BannerColors(t *testing.T) {
	skin := DefaultHermesSkin()

	wantBanner := map[string]string{
		"border": "#CD7F32",
		"title":  "#FFD700",
		"accent": "#FFBF00",
		"dim":    "#B8860B",
		"text":   "#FFF8DC",
	}
	gotBanner := map[string]string{
		"border": skin.Colors.BannerBorder,
		"title":  skin.Colors.BannerTitle,
		"accent": skin.Colors.BannerAccent,
		"dim":    skin.Colors.BannerDim,
		"text":   skin.Colors.BannerText,
	}
	for name, want := range wantBanner {
		if got := gotBanner[name]; got != want {
			t.Fatalf("banner %s color = %q, want %q", name, got, want)
		}
	}
}

func TestHermesSkin_UIColors(t *testing.T) {
	skin := DefaultHermesSkin()

	if skin.Colors.UIOk != "#4caf50" {
		t.Fatalf("UI ok color = %q, want #4caf50", skin.Colors.UIOk)
	}
	if skin.Colors.UIError != "#ef5350" {
		t.Fatalf("UI error color = %q, want #ef5350", skin.Colors.UIError)
	}
	if skin.Colors.UIWarn != "#ffa726" {
		t.Fatalf("UI warn color = %q, want #ffa726", skin.Colors.UIWarn)
	}
}

func TestHermesSkin_ToolEmojis(t *testing.T) {
	skin := DefaultHermesSkin()

	tests := []struct {
		tool  string
		emoji string
	}{
		{"web_search", "🔍"},
		{"terminal", "💻"},
		{"read_file", "📖"},
		{"write_file", "✍️"},
		{"memory", "🧠"},
		{"execute_code", "🐍"},
		{"delegate_task", "🔀"},
		{"unknown_tool", "⚡"},
	}
	for _, tc := range tests {
		if got := skin.ToolEmoji(tc.tool); got != tc.emoji {
			t.Fatalf("ToolEmoji(%q) = %q, want %q", tc.tool, got, tc.emoji)
		}
	}
}
