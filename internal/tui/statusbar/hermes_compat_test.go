package statusbar_test

import (
	"strings"
	"testing"

	tui "github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func forceLipglossTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func newWideStatusModel() tui.HermesStatusModel {
	return tui.HermesStatusModel{
		StatusLabel:      "ready",
		ModelName:        "anthropic/claude-sonnet-4-20250514",
		ContextTokens:    12_450,
		ContextLength:    200_000,
		SessionDuration:  14*60 + 32, // 14m32s
		PromptElapsed:    3,
		PromptLive:       true,
		HasPromptElapsed: true,
		CWDLabel:         "~/gormes-agent (development)",
	}
}

func TestHermesStatusBar_WideTerminal(t *testing.T) {
	got := tui.RenderHermesStatusBar(newWideStatusModel(), 120)

	// All five wide-tier components must appear in one line.
	wantContains := []string{
		"─ ready",
		"sonnet 4 20250514",
		"12.4K/200K",
		"[",
		"6%",
		"15m",
		// per-prompt elapsed is rendered with the live emoji
		"⏱",
		"─ ~/gormes-agent (development)",
	}
	for _, frag := range wantContains {
		if !strings.Contains(got, frag) {
			t.Fatalf("wide status bar missing %q in %q", frag, got)
		}
	}
	if strings.Contains(got, "\n") {
		t.Fatalf("status bar wraps to multiple lines: %q", got)
	}
	if !strings.Contains(got, " │ ") {
		t.Fatalf("wide status bar must use Hermes bar separator: %q", got)
	}
	if strings.Contains(got, "⚕") {
		t.Fatalf("current Hermes Ink status rule must not render the old product glyph prefix: %q", got)
	}
}

func TestHermesStatusBar_MidWidthCollapsesToModelPercentDuration(t *testing.T) {
	model := newWideStatusModel()

	got := tui.RenderHermesStatusBar(model, 60)

	if !strings.Contains(got, "─ ready") {
		t.Fatalf("mid-width status bar missing ready status rule: %q", got)
	}
	if !strings.Contains(got, "sonnet 4 20250514") {
		t.Fatalf("mid-width status bar missing model: %q", got)
	}
	if !strings.Contains(got, "6%") {
		t.Fatalf("mid-width status bar missing context percent: %q", got)
	}
	if !strings.Contains(got, "15m") {
		t.Fatalf("mid-width status bar missing duration: %q", got)
	}
	// Per-prompt elapsed is also hidden below 76 columns.
	if strings.Contains(got, "⏱") || strings.Contains(got, "⏲") {
		t.Fatalf("mid-width status bar leaks per-prompt timer: %q", got)
	}
	if strings.Contains(got, " · ") {
		t.Fatalf("mid-width status bar leaked old middot footer separator: %q", got)
	}
}

func TestHermesStatusBar_NarrowWidthCollapsesToModelDuration(t *testing.T) {
	got := tui.RenderHermesStatusBar(newWideStatusModel(), 50)

	if !strings.Contains(got, "─ ready") {
		t.Fatalf("narrow status bar missing ready status rule: %q", got)
	}
	if !strings.Contains(got, "sonnet") {
		// Narrow tier may trim the long model with an ellipsis.
		t.Fatalf("narrow status bar missing model name: %q", got)
	}
	if strings.Contains(got, "⏱") || strings.Contains(got, "⏲") {
		t.Fatalf("narrow status bar leaks per-prompt timer: %q", got)
	}
	if strings.Contains(got, "⚕") {
		t.Fatalf("narrow status bar leaked old product glyph prefix: %q", got)
	}
}

func TestHermesStatusBar_TrimsToWidth(t *testing.T) {
	model := tui.HermesStatusModel{
		ModelName:       "anthropic/" + strings.Repeat("ultra-long-model-name-", 20),
		ContextTokens:   100_000,
		ContextLength:   200_000,
		SessionDuration: 60 * 60, // 1h
	}

	for _, width := range []int{40, 52, 76, 80, 120, 200} {
		got := tui.RenderHermesStatusBar(model, width)
		if strings.Contains(got, "\n") {
			t.Fatalf("width=%d: status bar wraps: %q", width, got)
		}
		if w := lipgloss.Width(got); w > width {
			t.Fatalf("width=%d: status bar cell width %d exceeds announced width: %q", width, w, got)
		}
	}

	// Wide CJK glyphs in the model name must still respect width budget.
	cjk := tui.HermesStatusModel{
		ModelName:       strings.Repeat("你", 30),
		ContextTokens:   1000,
		ContextLength:   8000,
		SessionDuration: 30,
	}
	for _, width := range []int{20, 40, 80} {
		got := tui.RenderHermesStatusBar(cjk, width)
		if w := lipgloss.Width(got); w > width {
			t.Fatalf("cjk width=%d: status bar cell width %d > announced: %q", width, w, got)
		}
	}
}

func TestHermesStatusBarWithSkinUsesSharedStatusStyle(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := tui.BuiltinSkins()["poseidon"]
	shared := tui.SkinStylesFor(skin)

	got := tui.RenderHermesStatusBarWithSkin(newWideStatusModel(), 80, skin)
	plain := tui.StripANSIForTUI(got)
	if !strings.Contains(plain, "─ ready") || !strings.Contains(plain, "sonnet 4 20250514") {
		t.Fatalf("styled status bar lost status/model text:\n%s", got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("styled status bar should include ANSI style from active skin:\n%s", got)
	}
	if fg := shared.Status.GetForeground(); fg != lipgloss.Color(skin.Colors.StatusBarText) {
		t.Fatalf("shared status foreground = %v, want %v", fg, lipgloss.Color(skin.Colors.StatusBarText))
	}
	if bg := shared.Status.GetBackground(); bg != lipgloss.Color(skin.Colors.StatusBarBackground) {
		t.Fatalf("shared status background = %v, want %v", bg, lipgloss.Color(skin.Colors.StatusBarBackground))
	}
	if w := lipgloss.Width(got); w > 80 {
		t.Fatalf("styled status width = %d, want <= 80: %q", w, got)
	}
}

func TestHermesStatusBarWithSkinStylesContextSeverity(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := tui.BuiltinSkins()["poseidon"]
	model := newWideStatusModel()
	model.ContextTokens = 90
	model.ContextLength = 100

	got := tui.RenderHermesStatusBarWithSkin(model, 120, skin)
	segment := "[█████████░] 90%"
	if !strings.Contains(tui.StripANSIForTUI(got), segment) {
		t.Fatalf("styled status bar missing context segment %q:\n%s", segment, got)
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("styled status bar should include severity ANSI styling for context segment: %s", got)
	}
}

func TestHermesStatusBar_ContextThresholdStyles(t *testing.T) {
	for _, tc := range []struct {
		percent int
		want    tui.HermesStatusContextSeverity
	}{
		{percent: 0, want: tui.HermesStatusContextGood},
		{percent: 49, want: tui.HermesStatusContextGood},
		{percent: 50, want: tui.HermesStatusContextWarn},
		{percent: 80, want: tui.HermesStatusContextWarn},
		{percent: 81, want: tui.HermesStatusContextBad},
		{percent: 94, want: tui.HermesStatusContextBad},
		{percent: 95, want: tui.HermesStatusContextCritical},
		{percent: 100, want: tui.HermesStatusContextCritical},
	} {
		got := tui.HermesStatusBarContextSeverity(&tc.percent)
		if got != tc.want {
			t.Fatalf("severity for percent=%d = %v, want %v", tc.percent, got, tc.want)
		}
	}

	// nil percent (no context length known) must classify as dim.
	if got := tui.HermesStatusBarContextSeverity(nil); got != tui.HermesStatusContextDim {
		t.Fatalf("severity for nil percent = %v, want dim", got)
	}
}
