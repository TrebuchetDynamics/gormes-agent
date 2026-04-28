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
	if skin.ResponseLabel == "" || !strings.Contains(skin.ResponseLabel, "Hermes") {
		t.Fatalf("ResponseLabel = %q, want Hermes response label", skin.ResponseLabel)
	}
	if skin.Colors.InputRule != "#CD7F32" {
		t.Fatalf("InputRule color = %q, want Hermes bronze", skin.Colors.InputRule)
	}

	wantStatus := map[string]string{
		"background": "#1a1a2e",
		"text":       "#FFF8DC",
		"strong":     "#FFD700",
		"dim":        "#B8860B",
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
