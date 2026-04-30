package tui

import (
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func newHermesChromeFrame() kernel.RenderFrame {
	return kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "anthropic/claude-sonnet-4-20250514",
		SessionID: "sess-hermes-chrome",
		History: []hermes.Message{
			{Role: "user", Content: "ping from operator"},
			{Role: "assistant", Content: "pong from hermes"},
		},
	}
}

func TestHermesChrome_NoSidebar(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	frames <- newHermesChromeFrame()
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32

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
		if !strings.Contains(line, "Hermes") {
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
	// The status bar carries the model name, which is unique to that row.
	statusIdx := strings.Index(got, "claude-sonnet-4-20250514")
	promptIdx := strings.Index(got, "❯")

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

func TestHermesChrome_ResponseBoxLabel(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := newHermesChromeFrame()
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 120
	m.height = 32
	m.frame = f

	got := m.View()

	if strings.Contains(got, "gormes:") {
		t.Fatalf("View leaked dashboard `gormes:` assistant tag instead of Hermes response label:\n%s", got)
	}
	if !strings.Contains(got, "Hermes") {
		t.Fatalf("View missing Hermes response label:\n%s", got)
	}
	if !strings.Contains(got, "pong from hermes") {
		t.Fatalf("View missing assistant content under Hermes label:\n%s", got)
	}
	// User previews must remain visible — only the assistant tag is rebranded.
	if !strings.Contains(got, "you:") {
		t.Fatalf("View dropped user preview tag:\n%s", got)
	}
	if !strings.Contains(got, "ping from operator") {
		t.Fatalf("View missing user preview content:\n%s", got)
	}
}

func TestHermesChrome_MinimalWidthDropsBottomRule(t *testing.T) {
	in := HermesChromeInput{
		Width:        50,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "<<PROMPT>>",
	}

	got := RenderHermesChrome(in)

	if !strings.Contains(got, "<<STATUS>>") {
		t.Fatalf("narrow chrome dropped status bar:\n%s", got)
	}
	if !strings.Contains(got, "<<PROMPT>>") {
		t.Fatalf("narrow chrome dropped prompt:\n%s", got)
	}

	// Hermes hides the bottom input rule below 64 columns. The top rule
	// stays so the prompt is still visually separated from the status bar.
	bottomRule := strings.Repeat("─", 8)
	statusIdx := strings.Index(got, "<<STATUS>>")
	promptIdx := strings.Index(got, "<<PROMPT>>")
	tail := got[promptIdx+len("<<PROMPT>>"):]
	if strings.Contains(tail, bottomRule) {
		t.Fatalf("narrow chrome rendered a bottom rule below the prompt:\n%s", got)
	}

	// At 120 cols, the bottom rule re-appears beneath the prompt. This pins
	// the asymmetric Hermes _tui_input_rule_height("top"=1, "bottom"=0|1) rule.
	wide := RenderHermesChrome(HermesChromeInput{
		Width:        120,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "<<PROMPT>>",
	})
	wTail := wide[strings.Index(wide, "<<PROMPT>>")+len("<<PROMPT>>"):]
	if !strings.Contains(wTail, bottomRule) {
		t.Fatalf("wide chrome should render a bottom rule beneath the prompt:\n%s", wide)
	}

	// Sanity: the narrow chrome still inserts a top rule between status and prompt.
	between := got[statusIdx+len("<<STATUS>>") : promptIdx]
	if !strings.Contains(between, bottomRule) {
		t.Fatalf("narrow chrome must keep top rule between status and prompt:\n%q", between)
	}
}

func TestHermesChrome_UsesAltScreen(t *testing.T) {
	if !HermesChromeUseAltScreen() {
		t.Fatal("HermesChromeUseAltScreen() = false; full-screen Hermes chrome must use alt-screen to avoid stale render frames")
	}
}
