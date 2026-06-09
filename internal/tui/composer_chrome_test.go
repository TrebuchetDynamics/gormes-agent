package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestRenderTextInputChromeAddsReusableLabelAndHint(t *testing.T) {
	got := RenderTextInputChrome(TextInputChrome{
		Width: 72,
		Label: "Profile field",
		Hint:  "Enter save · Esc back",
		Value: "Name: main",
		Skin:  DefaultHermesSkin(),
	})

	assertContainsInOrder(t, got, "Profile field", "Enter save", "Name: main")
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 72 {
			t.Fatalf("text input chrome line width %d exceeds 72:\n%q\n\n%s", width, line, got)
		}
	}
}

func TestRenderComposerInputChromeStaysBareLikeHermesPrompt(t *testing.T) {
	got := RenderComposerInputChrome(ComposerInputChrome{
		Width:   72,
		Prompt:  "❯ hello",
		Draft:   "hello",
		Skin:    DefaultHermesSkin(),
		Focused: true,
	})

	if got != "❯ hello" {
		t.Fatalf("composer chrome = %q, want bare Hermes prompt", got)
	}
	for _, noisy := range []string{"Ask Gormes", "Enter send", "Shift+Enter newline"} {
		if strings.Contains(got, noisy) {
			t.Fatalf("composer chrome leaked helper text %q: %q", noisy, got)
		}
	}
}

func TestRenderComposerInputChromeReusesCompactPromptOnNarrowTerminals(t *testing.T) {
	got := RenderComposerInputChrome(ComposerInputChrome{
		Width:   32,
		Prompt:  "❯ compact",
		Skin:    DefaultHermesSkin(),
		Focused: true,
	})

	if got != "❯ compact" {
		t.Fatalf("narrow composer chrome = %q, want bare prompt", got)
	}
}

func TestComposerInputChromeProvidesModeAwareKeyHelp(t *testing.T) {
	plain := ComposerInputChrome{Draft: "hello"}.KeyHelp()
	if len(plain) != 2 || plain[0].Keys[0] != "Enter" || plain[1].Keys[0] != "Shift+Enter" {
		t.Fatalf("plain composer key help = %+v", plain)
	}
	slash := ComposerInputChrome{Draft: "/he"}.KeyHelp()
	if len(slash) != 2 || slash[0].Keys[0] != "Tab" || slash[1].Keys[0] != "↑" {
		t.Fatalf("slash composer key help = %+v", slash)
	}
}

func TestRenderComposerInputChromeDoesNotAddPausedHeader(t *testing.T) {
	got := RenderComposerInputChrome(ComposerInputChrome{
		Width:   72,
		Prompt:  "❯ draft",
		Skin:    DefaultHermesSkin(),
		Focused: false,
	})

	if got != "❯ draft" {
		t.Fatalf("paused composer chrome = %q, want bare prompt", got)
	}
}

func TestComposerInputChromeViewportGateKeepsBarePrompt(t *testing.T) {
	for _, size := range [][2]int{{80, 14}, {80, 24}} {
		if showComposerInputChrome(size[0], size[1]) {
			t.Fatalf("composer affordance should stay hidden for Hermes-like bare prompt at %dx%d", size[0], size[1])
		}
	}
}

func TestViewShowsComposerAffordanceOnlyWhenRoomy(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		Model: "test/model",
		History: []llm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "done"},
		},
	}
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.width = 80
	m.height = 24
	m.frame = frame

	roomy := m.View()
	assertContainsInOrder(t, roomy, "done", "❯")
	if strings.Contains(roomy, "Ask Gormes") {
		t.Fatalf("roomy terminal View() rendered noisy composer affordance:\n%s", roomy)
	}

	m.height = 14
	compact := m.View()
	if strings.Contains(compact, "Ask Gormes") {
		t.Fatalf("short terminal View() rendered composer affordance and stole transcript space:\n%s", compact)
	}
}
