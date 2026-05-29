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

func TestRenderComposerInputChromeAddsReusableAffordance(t *testing.T) {
	got := RenderComposerInputChrome(ComposerInputChrome{
		Width:   72,
		Prompt:  "❯ hello",
		Draft:   "hello",
		Skin:    DefaultHermesSkin(),
		Focused: true,
	})

	assertContainsInOrder(t, got, "Ask Gormes", "Enter send", "Shift+Enter newline", "❯ hello")
	for _, line := range strings.Split(got, "\n") {
		if width := lipgloss.Width(line); width > 72 {
			t.Fatalf("composer chrome line width %d exceeds 72:\n%q\n\n%s", width, line, got)
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

func TestRenderComposerInputChromeShowsContextualHints(t *testing.T) {
	tests := []struct {
		name      string
		draft     string
		multiline bool
		want      string
	}{
		{name: "empty", want: "/ commands"},
		{name: "plain", draft: "hello", want: "Shift+Enter newline"},
		{name: "slash", draft: "/he", want: "Tab complete"},
		{name: "multiline", draft: "line 1\nline 2", multiline: true, want: "Enter send"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderComposerInputChrome(ComposerInputChrome{
				Width:     72,
				Prompt:    "❯ " + tt.draft,
				Draft:     tt.draft,
				Skin:      DefaultHermesSkin(),
				Focused:   true,
				Multiline: tt.multiline,
			})
			if !strings.Contains(got, tt.want) {
				t.Fatalf("composer hint missing %q:\n%s", tt.want, got)
			}
		})
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

func TestRenderComposerInputChromeShowsPausedState(t *testing.T) {
	got := RenderComposerInputChrome(ComposerInputChrome{
		Width:   72,
		Prompt:  "❯ draft",
		Skin:    DefaultHermesSkin(),
		Focused: false,
	})

	assertContainsInOrder(t, got, "Composer paused", "❯ draft")
}

func TestComposerInputChromeViewportGatePreservesShortTranscriptSpace(t *testing.T) {
	if showComposerInputChrome(80, 14) {
		t.Fatal("composer affordance should stay hidden on short terminals")
	}
	if !showComposerInputChrome(80, 24) {
		t.Fatal("composer affordance should show on roomy terminals")
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
	assertContainsInOrder(t, roomy, "done", "Ask Gormes", "❯")

	m.height = 14
	compact := m.View()
	if strings.Contains(compact, "Ask Gormes") {
		t.Fatalf("short terminal View() rendered composer affordance and stole transcript space:\n%s", compact)
	}
}
