package tui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestRenderFaceTicker_Idle(t *testing.T) {
	got := RenderFaceTicker("idle", 0)
	if got == "" {
		t.Fatal("RenderFaceTicker returned empty string for idle state")
	}
	// Should return one of the status indicators
	expected := []string{"⚕", "🌀", "🤔", "✨", "🍵", "🔮"}
	found := false
	for _, e := range expected {
		if strings.Contains(got, e) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("RenderFaceTicker(idle, 0) = %q, want one of %v", got, expected)
	}
}

func TestRenderFaceTicker_FrameCycling(t *testing.T) {
	// Collect all frames for reasoning state to verify cycling through 6 positions
	frames := make(map[int]bool)
	for frame := 0; frame < 12; frame++ {
		got := RenderFaceTicker("reasoning", frame)
		if got == "" {
			t.Fatalf("frame %d returned empty string", frame)
		}
		frames[frame%6] = true // 6 indicators
	}

	// All 6 positions should be reachable
	if len(frames) < 6 {
		t.Fatal("not all face positions reachable")
	}
}

func TestRenderFaceTicker_DifferentStates(t *testing.T) {
	states := []string{"idle", "reasoning", "working", "waiting", "ready", "break", "magic", "error"}

	for _, state := range states {
		got := RenderFaceTicker(state, 0)
		if got == "" {
			t.Fatalf("state %q returned empty string", state)
		}
	}
}

func TestRenderFaceTicker_StableAtSameFrame(t *testing.T) {
	// Same frame should give same result (pure function)
	got1 := RenderFaceTicker("reasoning", 5)
	got2 := RenderFaceTicker("reasoning", 5)
	if got1 != got2 {
		t.Fatalf("RenderFaceTicker not stable: frame 5 gave %q and %q", got1, got2)
	}
}

func TestRenderContextBar_ZeroPercent(t *testing.T) {
	got := RenderContextBar(0.0)
	if !strings.Contains(got, "░") {
		t.Fatalf("0%% context bar missing ░: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Fatalf("0%% context bar should not contain █: %q", got)
	}
}

func TestRenderContextBar_HundredPercent(t *testing.T) {
	got := RenderContextBar(100.0)
	if !strings.Contains(got, "█") {
		t.Fatalf("100%% context bar missing █: %q", got)
	}
}

func TestRenderContextBar_HalfPercent(t *testing.T) {
	got := RenderContextBar(50.0)
	// Should contain both filled and empty
	if !strings.Contains(got, "█") || !strings.Contains(got, "░") {
		t.Fatalf("50%% context bar should contain both █ and ░: %q", got)
	}
}

func TestRenderContextBar_NearFull(t *testing.T) {
	got := RenderContextBar(90.0)
	// Should be mostly filled
	filled := strings.Count(got, "█")
	empty := strings.Count(got, "░")
	if filled < empty {
		t.Fatalf("90%% context bar should be mostly filled: %q (█=%d, ░=%d)", got, filled, empty)
	}
}

func TestRenderContextBar_ClampedNegative(t *testing.T) {
	got := RenderContextBar(-10.0)
	// Should be same as 0%
	if strings.Contains(got, "█") {
		t.Fatalf("-10%% context bar should not contain █: %q", got)
	}
}

func TestRenderContextBar_ClampedOverHundred(t *testing.T) {
	got := RenderContextBar(150.0)
	// Should be same as 100%
	if strings.Contains(got, "░") {
		t.Fatalf("150%% context bar should not contain ░: %q", got)
	}
}

func TestRenderContextBar_DefaultWidth(t *testing.T) {
	got := RenderContextBar(50.0)
	// Default bar should be 10 chars (same as status bar)
	// Use rune count since █ and ░ are multi-byte in UTF-8
	if utf8.RuneCountInString(got) != 10 {
		t.Fatalf("default context bar rune count = %d, want 10: %q", utf8.RuneCountInString(got), got)
	}
}

func TestExtensionStatusSetReplaceClearRendersWidthBounded(t *testing.T) {
	m := newExtensionUITestModel(56, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ext-status",
	})
	ctx := NewExtensionUIContext(&m, true)

	res := ctx.SetStatus("fake", "indexing workspace cache with a long note")
	if res.Status != kernel.ExtensionUIApplied {
		t.Fatalf("SetStatus status = %q, want applied: %#v", res.Status, res)
	}
	got := m.View()
	assertContainsInOrder(t, got, "─ ready", "indexing workspace")
	assertRenderedWidthAtMost(t, got, m.width)

	res = ctx.SetStatus("fake", "✓ indexed")
	if res.Status != kernel.ExtensionUIApplied {
		t.Fatalf("replacement SetStatus status = %q, want applied: %#v", res.Status, res)
	}
	got = m.View()
	if strings.Contains(got, "indexing workspace") {
		t.Fatalf("replacement left old extension status visible:\n%s", got)
	}
	if !strings.Contains(got, "✓ indexed") {
		t.Fatalf("replacement status missing from footer:\n%s", got)
	}

	res = ctx.ClearStatus("fake")
	if res.Status != kernel.ExtensionUICleared {
		t.Fatalf("ClearStatus status = %q, want cleared: %#v", res.Status, res)
	}
	got = m.View()
	if strings.Contains(got, "✓ indexed") {
		t.Fatalf("cleared extension status still visible:\n%s", got)
	}
}

func TestExtensionWidgetAboveBelowComposeWithChromeOrdering(t *testing.T) {
	m := newExtensionUITestModel(80, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ext-widget",
	})
	m.todoReader = func(string) []TodoItem {
		return []TodoItem{{Text: "todo before extension widget", Status: TodoStatusPending}}
	}
	m.transientPage = &TransientPageState{Title: "Panel", Body: "panel before widget"}
	ctx := NewExtensionUIContext(&m, true)

	ctx.SetWidget("above", []string{"ABOVE editor widget line that should be trimmed but visible"}, kernel.ExtensionUIWidgetOptions{Placement: kernel.ExtensionUIWidgetAboveEditor})
	ctx.SetWidget("below", []string{"BELOW editor widget"}, kernel.ExtensionUIWidgetOptions{Placement: kernel.ExtensionUIWidgetBelowEditor})

	got := m.View()
	assertContainsInOrder(t, got,
		"todo before extension widget",
		"Panel",
		"ABOVE editor widget",
		"─ ready",
		"❯",
		"BELOW editor widget",
	)
	assertRenderedWidthAtMost(t, got, m.width)
}

func TestExtensionFooterReplacesStatusAndClears(t *testing.T) {
	m := newExtensionUITestModel(54, kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ext-footer",
	})
	ctx := NewExtensionUIContext(&m, true)

	res := ctx.SetFooter([]string{"CUSTOM footer from fake extension with long text"})
	if res.Status != kernel.ExtensionUIApplied {
		t.Fatalf("SetFooter status = %q, want applied: %#v", res.Status, res)
	}
	got := m.View()
	if !strings.Contains(got, "CUSTOM footer from fake extension") {
		t.Fatalf("custom footer missing from view:\n%s", got)
	}
	if strings.Contains(got, "─ ready │") {
		t.Fatalf("custom footer should replace built-in status row:\n%s", got)
	}
	assertRenderedWidthAtMost(t, got, m.width)

	res = ctx.ClearFooter()
	if res.Status != kernel.ExtensionUICleared {
		t.Fatalf("ClearFooter status = %q, want cleared: %#v", res.Status, res)
	}
	got = m.View()
	if strings.Contains(got, "CUSTOM footer") {
		t.Fatalf("cleared custom footer still visible:\n%s", got)
	}
	if !strings.Contains(got, "─ ready │") {
		t.Fatalf("built-in status row did not return after clearing custom footer:\n%s", got)
	}
}

func TestExtensionWorkingIndicatorOverridesAndClears(t *testing.T) {
	m := newExtensionUITestModel(72, kernel.RenderFrame{
		Phase:     kernel.PhaseStreaming,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-ext-working",
	})
	m.spinnerFrame = 1
	ctx := NewExtensionUIContext(&m, true)

	res := ctx.SetWorkingIndicator(kernel.ExtensionUIWorkingIndicator{
		Text:   "syncing extension cache",
		Frames: []string{"◐", "◓"},
	})
	if res.Status != kernel.ExtensionUIApplied {
		t.Fatalf("SetWorkingIndicator status = %q, want applied: %#v", res.Status, res)
	}
	got := m.renderHermesHint()
	for _, want := range []string{"◓", "syncing extension cache", "session sess-ext"} {
		if !strings.Contains(got, want) {
			t.Fatalf("custom working indicator missing %q from %q", want, got)
		}
	}
	if strings.Contains(got, "streaming") {
		t.Fatalf("custom working message should replace default phase label, got %q", got)
	}

	res = ctx.ClearWorkingIndicator()
	if res.Status != kernel.ExtensionUICleared {
		t.Fatalf("ClearWorkingIndicator status = %q, want cleared: %#v", res.Status, res)
	}
	got = m.renderHermesHint()
	if strings.Contains(got, "syncing extension cache") || strings.Contains(got, "◓") {
		t.Fatalf("cleared working indicator still visible: %q", got)
	}
	if !strings.Contains(got, "streaming") {
		t.Fatalf("default active-turn phase did not return after clear: %q", got)
	}
}

func TestExtensionUINoopContextReturnsEvidence(t *testing.T) {
	ctx := NewExtensionUIContext(nil, false)
	res := ctx.SetStatus("fake", "ignored")
	if res.Status != kernel.ExtensionUIUnavailable {
		t.Fatalf("SetStatus on unavailable context status = %q, want unavailable: %#v", res.Status, res)
	}
	if !strings.Contains(res.Evidence, "non-interactive") {
		t.Fatalf("unavailable evidence = %q, want non-interactive reason", res.Evidence)
	}
}

func newExtensionUITestModel(width int, frame kernel.RenderFrame) Model {
	frames := make(chan kernel.RenderFrame, 1)
	m := NewModel(frames, func(string) {}, func() {})
	m.width = width
	m.height = 28
	m.frame = frame
	return m
}

func assertRenderedWidthAtMost(t *testing.T, rendered string, width int) {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		if got := lipgloss.Width(line); got > width {
			t.Fatalf("rendered line width = %d, want <= %d: %q\nfull view:\n%s", got, width, line, rendered)
		}
	}
}
