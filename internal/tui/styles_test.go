package tui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/testenv"
)

func TestChatStyles_ReThemesPerSkin(t *testing.T) {
	defCS := chatStylesFor(BuiltinSkins()["default"])
	posCS := chatStylesFor(BuiltinSkins()["poseidon"])

	// Compare foreground colors directly (not rendered output) to avoid
	// truecolor profile dependency.
	if defCS.User.GetForeground() == posCS.User.GetForeground() {
		t.Fatalf("switching skin did not re-theme user style: default=%v poseidon=%v",
			defCS.User.GetForeground(), posCS.User.GetForeground())
	}

	// Styles resolve without panicking for every built-in skin.
	for name, sk := range BuiltinSkins() {
		cs := chatStylesFor(sk)
		if cs.User.GetForeground() == nil {
			t.Fatalf("skin %s produced empty user color", name)
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
	if got, want := shared.ActivePill.GetForeground(), lipgloss.Color(skin.Colors.StatusBarBackground); got != want {
		t.Fatalf("shared active pill foreground = %v, want %v", got, want)
	}
	if got, want := shared.ActivePill.GetBackground(), lipgloss.Color(skin.Colors.UIAcent); got != want {
		t.Fatalf("shared active pill background = %v, want %v", got, want)
	}
	if got, want := shared.BannerBorder.GetForeground(), lipgloss.Color(skin.Colors.BannerBorder); got != want {
		t.Fatalf("shared banner border foreground = %v, want %v", got, want)
	}
	if got, want := shared.BannerAccent.GetForeground(), lipgloss.Color(skin.Colors.BannerAccent); got != want {
		t.Fatalf("shared banner accent foreground = %v, want %v", got, want)
	}
	if got, want := shared.BannerDim.GetForeground(), lipgloss.Color(skin.Colors.BannerDim); got != want {
		t.Fatalf("shared banner dim foreground = %v, want %v", got, want)
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
	testenv.TrueColor(t)
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

func TestModelViewGlobalChromeSmokeAcrossBuiltInSkins(t *testing.T) {
	testenv.TrueColor(t)
	frame := kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-global-skin-smoke",
		History: []llm.Message{
			{Role: "user", Content: "please improve the global TUI styling"},
			{Role: "assistant", Content: "# Styling pass\n\nUse shared skin tokens for `gormes` chrome."},
			{Role: "tool", Name: "bash", Content: "go test ./internal/tui -count=1"},
		},
		ContextStatus: &llm.ContextStatus{ContextLength: 200_000, LastTotalTokens: 123_456},
	}
	skins := BuiltinSkins()
	names := make([]string, 0, len(skins))
	for name := range skins {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		skin := skins[name]
		t.Run(name, func(t *testing.T) {
			frames := make(chan kernel.RenderFrame, 1)
			m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{MouseTracking: true, SkinName: name})
			m.width = 96
			m.height = 28
			m.frame = frame
			m.editor.SetValue("/skin")
			m.todoReader = func(string) []TodoItem {
				return []TodoItem{
					{Text: "route chrome through shared styles", Status: TodoStatusDone},
					{Text: "smoke active skin focus and completions", Status: TodoStatusPending},
				}
			}

			got := m.View()
			plain := StripANSIForTUI(got)
			promptSymbol, _ := skin.PromptSymbols("default")
			for _, want := range []string{
				"─ running",
				"Search /skin",
				strings.TrimSpace(promptSymbol),
				"route chrome through shared styles",
				"please improve the global TUI styling",
				"Styling pass",
			} {
				if !strings.Contains(plain, want) {
					t.Fatalf("global skin smoke missing %q for skin %s:\n%s", want, name, plain)
				}
			}
			if !strings.Contains(got, "\x1b[") {
				t.Fatalf("global skin smoke for skin %s rendered no ANSI styling:\n%s", name, got)
			}
			assertRenderedWidthAtMost(t, got, m.width)
		})
	}
}

func TestTUISurfaceColorsRouteThroughSharedStyles(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		switch filepath.Base(path) {
		case "hermes_facade.go", "hermes_skin.go":
			return nil
		}
		// Allow subpackage skin/styles.go since it owns the canonical token
		// definitions and skin.SkinStylesFor is the blessed access pattern.
		if strings.HasPrefix(path, "skin/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "lipgloss.Color(") {
			offenders = append(offenders, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan TUI style files: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("TUI component files must route colors through SkinStyles/hermes_facade.go, found direct lipgloss.Color use in: %s", strings.Join(offenders, ", "))
	}
}

func TestConversationViewportTail_UsesActiveSkinGlyphs(t *testing.T) {
	frame := kernel.RenderFrame{
		Phase: kernel.PhaseIdle,
		History: []llm.Message{
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
		History: []llm.Message{
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
