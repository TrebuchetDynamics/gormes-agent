package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestChatPalette_SkinDerivedAndRoleDistinct(t *testing.T) {
	def := BuiltinSkins()["default"]
	p := chatPaletteFor(def)

	// Every semantic color must be sourced from the active skin's tokens,
	// never hardcoded.
	cases := map[string][2]string{
		"user":       {p.user, def.Colors.UILabel},
		"toolName":   {p.toolName, def.Colors.UIAcent},
		"toolOutput": {p.toolOutput, def.Colors.BannerDim},
		"error":      {p.errorc, def.Colors.UIError},
		"prompt":     {p.prompt, def.Colors.Prompt},
		"separator":  {p.separator, def.Colors.SessionBorder},
	}
	for role, pair := range cases {
		if pair[0] == "" || pair[0] != pair[1] {
			t.Fatalf("role %s color %q not sourced from skin token %q", role, pair[0], pair[1])
		}
	}

	// Roles must be visually distinguishable from one another.
	if p.user == p.toolName || p.toolName == p.errorc || p.user == p.errorc {
		t.Fatalf("semantic roles not distinct: user=%q tool=%q error=%q", p.user, p.toolName, p.errorc)
	}
}

func TestChatPalette_ReThemesPerSkin(t *testing.T) {
	def := chatPaletteFor(BuiltinSkins()["default"])
	pos := chatPaletteFor(BuiltinSkins()["poseidon"])

	if def.user == pos.user && def.errorc == pos.errorc && def.toolName == pos.toolName {
		t.Fatalf("switching skin did not re-theme any chat role: default=%+v poseidon=%+v", def, pos)
	}

	// Styles resolve without panicking for every built-in skin.
	for name, sk := range BuiltinSkins() {
		cs := chatStylesFor(sk)
		if cs.User.Render("x") == "" || cs.Error.Render("y") == "" {
			t.Fatalf("skin %s produced empty role render", name)
		}
	}
}

func TestSkinStylesSharedAcrossBubbleInputs(t *testing.T) {
	skin := BuiltinSkins()["poseidon"]
	shared := SkinStylesFor(skin)
	if got, want := shared.Selected.GetForeground(), lipgloss.Color(skin.Colors.UIAcent); got != want {
		t.Fatalf("shared selected foreground = %v, want %v", got, want)
	}
	if got, want := shared.Status.GetBackground(), lipgloss.Color(skin.Colors.StatusBarBackground); got != want {
		t.Fatalf("shared status background = %v, want %v", got, want)
	}
	if got, want := shared.FocusLine.GetBackground(), lipgloss.Color(skin.Colors.StatusBarBackground); got != want {
		t.Fatalf("shared focus line background = %v, want %v", got, want)
	}

	input := textinput.New()
	ApplyTextInputSkin(&input, skin)
	if got, want := input.PromptStyle.GetForeground(), shared.Prompt.GetForeground(); got != want {
		t.Fatalf("textinput prompt foreground = %v, want shared %v", got, want)
	}
	if got, want := input.Cursor.Style.GetForeground(), shared.Cursor.GetForeground(); got != want {
		t.Fatalf("textinput cursor foreground = %v, want shared %v", got, want)
	}
}

func TestComposerTextareaStylesFollowActiveSkin(t *testing.T) {
	skin := BuiltinSkins()["poseidon"]
	m := NewModelWithOptions(nil, nil, nil, Options{SkinName: skin.Name})

	if got, want := m.editor.FocusedStyle.Prompt.GetForeground(), lipgloss.Color(skin.Colors.Prompt); got != want {
		t.Fatalf("focused prompt foreground = %v, want %v", got, want)
	}
	if got, want := m.editor.FocusedStyle.Placeholder.GetForeground(), lipgloss.Color(skin.Colors.Placeholder); got != want {
		t.Fatalf("focused placeholder foreground = %v, want %v", got, want)
	}
	if got, want := m.editor.FocusedStyle.Text.GetForeground(), lipgloss.Color(skin.Colors.BannerText); got != want {
		t.Fatalf("focused text foreground = %v, want %v", got, want)
	}
	if got, want := m.editor.Cursor.Style.GetForeground(), lipgloss.Color(skin.Colors.UIAcent); got != want {
		t.Fatalf("cursor foreground = %v, want %v", got, want)
	}
	if got, want := m.editor.FocusedStyle.CursorLine.GetBackground(), lipgloss.Color(skin.Colors.StatusBarBackground); got != want {
		t.Fatalf("focused cursor line background = %v, want %v", got, want)
	}
	if got, want := m.editor.BlurredStyle.CursorLine.GetBackground(), lipgloss.Color(skin.Colors.StatusBarBackground); got == want {
		t.Fatalf("blurred cursor line background = %v, should not keep focused background %v", got, want)
	}

	accepted, err := m.applySkinName("ares")
	if err != nil || accepted != "ares" {
		t.Fatalf("applySkinName(ares) = %q, %v", accepted, err)
	}
	ares := BuiltinSkins()["ares"]
	if got, want := m.editor.FocusedStyle.Prompt.GetForeground(), lipgloss.Color(ares.Colors.Prompt); got != want {
		t.Fatalf("hot-swapped prompt foreground = %v, want %v", got, want)
	}
}

func TestComposerFocusDimsWhenOverlayOwnsFocus(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := BuiltinSkins()["poseidon"]
	m := NewModelWithOptions(nil, nil, nil, Options{SkinName: skin.Name})

	if !m.composerChatFocused() {
		t.Fatal("new chat model should focus composer when no overlay is active")
	}
	promptSymbol, _ := skin.PromptSymbols("default")
	promptSymbol = strings.TrimSpace(promptSymbol)
	focused := renderComposerPromptWithFocus(m.editor, skin, true)
	if !strings.Contains(StripANSIForTUI(focused), promptSymbol+" Type a message") {
		t.Fatalf("focused composer lost prompt/placeholder:\n%s", focused)
	}
	if !strings.Contains(focused, "48;2") {
		t.Fatalf("focused composer should render active focus-line background:\n%s", focused)
	}

	m.transientPage = &TransientPageState{Title: "Help", Body: "overlay owns focus"}
	if m.composerChatFocused() {
		t.Fatal("transient page should move visual focus away from composer")
	}
	blurred := m.renderComposerPrompt(m.editor)
	if !strings.Contains(StripANSIForTUI(blurred), promptSymbol+" Type a message") {
		t.Fatalf("blurred composer should preserve Hermes prompt text:\n%s", blurred)
	}
	if strings.Contains(blurred, "48;2") {
		t.Fatalf("overlay-blurred composer should not keep active focus-line background:\n%s", blurred)
	}
}

func TestConversationViewportTail_UsesActiveSkinGlyphs(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "invoke ares"},
			{Role: "assistant", Content: "ares response"},
		},
	}
	got := StripANSIForTUI(conversationViewportTailWithSkinAndDetails(frame, 100, 24, false, DefaultDetailsState(), BuiltinSkins()["ares"]))

	for _, want := range []string{"⚔ invoke ares", "╎ ares response"} {
		if !strings.Contains(got, want) {
			t.Fatalf("active skin transcript missing %q:\n%s", want, got)
		}
	}
	for _, stale := range []string{"❯ invoke ares", "┊ ares response"} {
		if strings.Contains(got, stale) {
			t.Fatalf("active skin transcript kept default glyph %q:\n%s", stale, got)
		}
	}
}

func TestView_RoleDifferentiationSurvivesStyleRefactor(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []hermes.Message{
			{Role: "user", Content: "first ask"},
			{Role: "assistant", Content: "the answer"},
			{Role: "tool", Name: "search_files", Content: "hit one\nhit two"},
		},
		LastError: "boom failure",
	}
	got := renderConv(frame, 100, 24)

	// Each role keeps a distinct, scannable marker after routing through the
	// semantic style system.
	for role, marker := range map[string]string{
		"user":      "❯",
		"assistant": "┊",
		"tool":      "⚡ search_files",
		"error":     "err:",
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("role %s lost its differentiator %q after style refactor:\n%s", role, marker, got)
		}
	}
}
