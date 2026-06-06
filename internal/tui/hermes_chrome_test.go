package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func newHermesChromeFrame() kernel.RenderFrame {
	return kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-hermes-chrome",
		History: []llm.Message{
			{Role: "user", Content: "ping from operator"},
			{Role: "assistant", Content: "pong from hermes"},
		},
	}
}

func TestHermesChrome_NoSidebar(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := newHermesChromeFrame()
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32
	m.frame = f

	got := m.View()

	for _, banned := range []string{"Telemetry", "Soul Monitor"} {
		if strings.Contains(got, banned) {
			t.Fatalf("View at 120x32 leaked sidebar header %q:\n%s", banned, got)
		}
	}

	// The legacy sidebar pane was rendered with a rounded border that paired
	// a left-edge vertical pipe with a right-edge one on the conversation
	// row. Bottom-pinned chrome must not draw a sidebar pipe pair on the
	// conversation row carrying the assistant text.
	for _, line := range strings.Split(got, "\n") {
		if !strings.Contains(line, "Gormes") {
			continue
		}
		if strings.Count(line, "│") >= 2 {
			t.Fatalf("View at 120x32 still draws a side-by-side border on the response row: %q", line)
		}
	}
}

func TestHermesChrome_BottomPinnedOrder(t *testing.T) {
	in := HermesChromeInput{
		Width:        120,
		Conversation: "<<CONV>>",
		Spinner:      "<<SPIN>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "<<PROMPT>>",
		VoiceStatus:  "<<VOICE>>",
		ImageBar:     "<<IMAGE>>",
		Completions:  "<<COMPLETE>>",
	}

	got := RenderHermesChrome(in)

	wantOrder := []string{
		"<<CONV>>",
		"<<SPIN>>",
		"<<STATUS>>",
		"<<PROMPT>>",
		"<<VOICE>>",
		"<<IMAGE>>",
		"<<COMPLETE>>",
	}

	prev := -1
	for _, marker := range wantOrder {
		idx := strings.Index(got, marker)
		if idx < 0 {
			t.Fatalf("RenderHermesChrome output missing %q:\n%s", marker, got)
		}
		if idx <= prev {
			t.Fatalf("RenderHermesChrome ordering wrong: %q must appear after previous marker, got idx=%d prev=%d:\n%s", marker, idx, prev, got)
		}
		prev = idx
	}

	// Optional rows must drop out cleanly when absent — bottom-pinned chrome
	// must not introduce ghost blank lines between conversation and prompt.
	minimal := HermesChromeInput{
		Width:        120,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "<<PROMPT>>",
	}
	gotMin := RenderHermesChrome(minimal)
	for _, banned := range []string{"<<SPIN>>", "<<VOICE>>", "<<IMAGE>>", "<<COMPLETE>>"} {
		if strings.Contains(gotMin, banned) {
			t.Fatalf("minimal RenderHermesChrome leaked optional row %q:\n%s", banned, gotMin)
		}
	}
	if strings.Index(gotMin, "<<CONV>>") >= strings.Index(gotMin, "<<STATUS>>") {
		t.Fatalf("minimal chrome must keep conversation above status bar:\n%s", gotMin)
	}
	if strings.Index(gotMin, "<<STATUS>>") >= strings.Index(gotMin, "<<PROMPT>>") {
		t.Fatalf("minimal chrome must keep status bar above prompt:\n%s", gotMin)
	}
}

func TestHermesChrome_BottomPinnedOrder_View(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := newHermesChromeFrame()
	f.LastError = ""
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32
	m.frame = f

	got := m.View()

	convIdx := strings.Index(got, "pong from hermes")
	// The status rule carries the current Hermes Ink status + model label.
	statusIdx := strings.Index(got, "─ ready │ sonnet 4 20250514")
	promptIdx := strings.LastIndex(got, "❯")

	if convIdx < 0 {
		t.Fatalf("View missing conversation content:\n%s", got)
	}
	if statusIdx < 0 {
		t.Fatalf("View missing Hermes status bar:\n%s", got)
	}
	if promptIdx < 0 {
		t.Fatalf("View missing Hermes prompt symbol:\n%s", got)
	}
	if convIdx >= statusIdx {
		t.Fatalf("conversation must precede status bar:\n%s", got)
	}
	if statusIdx >= promptIdx {
		t.Fatalf("status bar must precede prompt:\n%s", got)
	}
}

func TestHermesChrome_IdleViewDoesNotReserveEmptyHintRow(t *testing.T) {
	history := make([]llm.Message, 0, 10)
	for i := 1; i <= 10; i++ {
		history = append(history, llm.Message{Role: "user", Content: fmt.Sprintf("probe-user-%02d", i)})
	}
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "test/model", History: history}
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.width = 80
	m.height = 14
	m.frame = f

	got := m.View()

	if strings.Contains(got, "probe-user-07") {
		t.Fatalf("precondition drift: test frame is too tall to prove a one-row reclaim:\n%s", got)
	}
	if !strings.Contains(got, "probe-user-08") {
		t.Fatalf("idle View() reserved an empty hint/progress row instead of giving it back to transcript:\n%s", got)
	}
	if strings.Contains(got, "streaming") || strings.Contains(got, "connecting") {
		t.Fatalf("idle View() rendered active progress text unexpectedly:\n%s", got)
	}
}

func TestHermesChrome_InkStyleTranscriptGutter(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := newHermesChromeFrame()
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32
	m.frame = f

	got := m.View()

	if strings.Contains(got, "gormes:") {
		t.Fatalf("View leaked dashboard `gormes:` assistant tag instead of Gormes response label:\n%s", got)
	}
	if strings.Contains(got, "⚕ Hermes") {
		t.Fatalf("View leaked upstream Hermes product label instead of Gormes label:\n%s", got)
	}
	if strings.Contains(got, "⚕ Gormes") {
		t.Fatalf("View leaked old label-heavy assistant tag instead of Hermes Ink message gutter:\n%s", got)
	}
	if strings.Contains(got, "you:") {
		t.Fatalf("View leaked old label-heavy user tag instead of Hermes Ink prompt glyph:\n%s", got)
	}
	assertContainsInOrder(t, got, "❯", "ping from operator", "┊", "pong from hermes")
}

func TestHermesChrome_EmptyChatIntroUsesBubbleTeaView(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-hermes-intro",
	}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{
		WelcomeToolCount: 26,
		WelcomeToolsets:  []string{"browser", "browser-cdp", "clarify", "code_execution", "computer_use", "cronjob", "delegation", "discord", "email"},
	})
	m.width = 100
	m.height = 28
	m.frame = f

	got := m.View()

	assertContainsInOrder(t, got,
		"Gormes",
		"sess-her",
		"browser, browser-cdp",
		"26 tools",
		"Welcome to Gormes",
		"Type your message or /help for commands.",
		"─ ready │ sonnet 4 20250514",
		"❯ Type a message",
	)
	if strings.Contains(got, "start typing below to begin") {
		t.Fatalf("Bubble Tea chat view leaked old empty placeholder:\n%s", got)
	}
}

func TestHermesChrome_InputPromptIsUnboxedSingleLineByDefault(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := newHermesChromeFrame()
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32
	m.frame = f

	got := m.View()

	for _, banned := range []string{"╭", "╮", "╰", "╯"} {
		if strings.Contains(got, banned) {
			t.Fatalf("View rendered boxed input chrome %q; Hermes prompt should be unboxed:\n%s", banned, got)
		}
	}
	if count := strings.Count(got, "❯ Type a message"); count != 1 {
		t.Fatalf("View rendered %d composer prompts, want one single-line idle prompt:\n%s", count, got)
	}
	if strings.Contains(got, "phase:") {
		t.Fatalf("View rendered debug phase chrome in idle state; Hermes keeps idle composer chrome quiet:\n%s", got)
	}
	if strings.Contains(got, "mouse: disabled") {
		t.Fatalf("View rendered persistent mouse-disabled noise in idle state:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if isStandaloneInputRuleLine(line) {
			t.Fatalf("View rendered standalone full-width input rule %q; current Hermes Ink uses the status rule as composer separation:\n%s", line, got)
		}
	}
}

func TestHermesChrome_DoesNotInjectStandaloneInputRules(t *testing.T) {
	for _, width := range []int{50, 120} {
		got := RenderHermesChrome(HermesChromeInput{
			Width:        width,
			Conversation: "<<CONV>>",
			StatusBar:    "<<STATUS>>",
			Prompt:       "<<PROMPT>>",
		})

		if !strings.Contains(got, "<<STATUS>>") {
			t.Fatalf("width=%d: chrome dropped status bar:\n%s", width, got)
		}
		if !strings.Contains(got, "<<PROMPT>>") {
			t.Fatalf("width=%d: chrome dropped prompt:\n%s", width, got)
		}
		for _, line := range strings.Split(got, "\n") {
			if isStandaloneInputRuleLine(line) {
				t.Fatalf("width=%d: chrome injected standalone input rule %q:\n%s", width, line, got)
			}
		}
	}
}

func TestHermesChrome_OptionalRowsRemainBelowPrompt(t *testing.T) {
	got := RenderHermesChrome(HermesChromeInput{
		Width:        120,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "<<PROMPT>>",
		VoiceStatus:  "<<VOICE>>",
	})

	if strings.Index(got, "<<PROMPT>>") >= strings.Index(got, "<<VOICE>>") {
		t.Fatalf("voice row must remain below prompt:\n%s", got)
	}
}

func isStandaloneInputRuleLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= 8 && strings.Trim(trimmed, "─") == ""
}

func TestHermesChrome_UsesAltScreen(t *testing.T) {
	if !HermesChromeUseAltScreen() {
		t.Fatal("HermesChromeUseAltScreen() = false; full-screen Hermes chrome must use alt-screen to avoid stale render frames")
	}
}
