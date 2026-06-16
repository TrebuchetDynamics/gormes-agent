package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

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

func hermesScreenshotToolsets() []string {
	return []string{
		"browser", "browser-cdp", "clarify", "code_execution", "computer_use", "cronjob", "delegation", "discord",
		"email", "file", "homeassistant", "image_gen", "kanban", "memory", "messaging", "session_search",
		"skills", "terminal", "todo", "tts", "vision", "mcp", "notes", "web_search",
		"github", "calendar", "database", "maps", "shell", "workspace",
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

func TestHermesChrome_StatusTickReschedulesIdleDurationRefresh(t *testing.T) {
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	updated, cmd := m.Update(statusTickMsg{})
	if cmd == nil {
		t.Fatal("status tick returned nil cmd, want rescheduled idle duration refresh")
	}
	if updated.(Model).spinnerFrame != 0 {
		t.Fatalf("status tick should not advance active spinner frame")
	}
}

func TestHermesChrome_StatusBarUsesLiveSessionDuration(t *testing.T) {
	f := newHermesChromeFrame()
	m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
	m.width = 120
	m.height = 28
	m.frame = f
	m.sessionStartedAt = time.Unix(1_000, 0)
	m.statusNow = func() time.Time { return time.Unix(1_001, 0) }

	got := RenderHermesStatusBarWithSkin(m.hermesStatusModelFromFrame(), m.width, m.currentSkin())
	if !strings.Contains(got, " │ 1s │ ⏲ 0s") {
		t.Fatalf("status bar should show live session duration before prompt timer, got %q", got)
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
	// The status rule carries the Hermes target glyph + model label.
	statusIdx := strings.Index(got, "⚕ sonnet 4 20250514")
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

func TestHermesChrome_ActiveWaitingHintIsQuiet(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{
		Phase:     kernel.PhaseConnecting,
		Model:     "gpt-5.5",
		SessionID: "sess-verbose-animation",
		History:   []llm.Message{{Role: "user", Content: "hello"}},
	}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{StartupNotice: "session: temporary (sessions.db busy)"})
	m.width = 120
	m.height = 28
	m.frame = f

	got := m.View()
	if !strings.Contains(got, "⠋ connecting") {
		t.Fatalf("active waiting hint missing quiet unicode spinner:\n%s", got)
	}
	for _, noisy := range []string{"Reasoning", "🤔", "(≧◡≦)", "session state: in-memory", "gateway status/stop", "session sess-"} {
		if strings.Contains(got, noisy) {
			t.Fatalf("active waiting view leaked noisy marker %q:\n%s", noisy, got)
		}
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

	if strings.Contains(got, "probe-user-08") {
		t.Fatalf("precondition drift: test frame is too tall to prove idle hint-row reclaim with input rules:\n%s", got)
	}
	if !strings.Contains(got, "probe-user-09") {
		t.Fatalf("idle View() reserved an empty hint/progress row beyond the prompt rule pair:\n%s", got)
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
	if !strings.Contains(got, "  ❯ ping from operator") {
		t.Fatalf("View missing Hermes-style two-space user prompt indent:\n%s", got)
	}
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
		"browser: browser_back, browser_click, ...",
		"browser-cdp: browser_cdp, browser_dialog",
		"sess-her",
		"26 tools",
		"Welcome to Gormes",
		"Type your message or /help for commands.",
		"✦ Tip: /voice tts toggles TTS-only mode",
		"⚕ sonnet 4 20250514",
		"[░░░░░░░░░░] --",
		"⏲ 0s",
		"❯",
	)
	if strings.Contains(got, "start typing below to begin") {
		t.Fatalf("Bubble Tea chat view leaked old empty placeholder:\n%s", got)
	}
}

func TestHermesChrome_ScreenshotStartupWelcomeAndInputStack(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{
		Phase:     kernel.PhaseIdle,
		Model:     "claude-opus-4-6",
		SessionID: "20260611_170004_284a49",
		ProviderStatus: llm.ProviderStatus{
			Provider: "nous-research",
		},
	}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{
		WelcomeVersion:          "0.2.24",
		WelcomeVersionDateAlias: "v2026.6.5",
		WelcomeGitCommit:        "d221e369abcdef",
		WelcomeToolCount:        28,
		WelcomeSkillCount:       70,
		WelcomeToolsets:         hermesScreenshotToolsets(),
		ProfileBaseHome:         "/home/xel/.gormes",
	})
	m.width = 140
	m.height = 52
	m.frame = f
	m.sessionStartedAt = time.Unix(1_000, 0)
	m.statusNow = func() time.Time { return time.Unix(1_001, 0) }

	got := m.View()
	assertContainsInOrder(t, got,
		"Gormes v0.2.24 (2026.6.5) · upstream d221e369",
		"Available Tools",
		"browser: browser_back, browser_click, ...",
		"(and 22 more toolsets…)",
		"Available Skills",
		"autonomous-ai-agents: claude-code, codex, hermes-agent, opencode",
		"claude-opus-4.6 · Nous Research",
		"/home/xel/.gormes",
		"Session: 20260611_170004_284a49",
		"28 tools · 70 skills · /help for commands",
		"Welcome to Gormes! Type your message or /help for commands.",
		"✦ Tip: /voice tts toggles TTS-only mode",
		"⚕ opus 4.6",
		"ctx --",
		"[░░░░░░░░░░] --",
		"⏲ 0s",
		"❯",
	)
	assertPromptRulePair(t, got, m.width, "❯")
	if !strings.Contains(got, "✦ Tip: /voice tts toggles TTS-only mode — agent replies out loud but you still type your prompts.\n\n ⚕ opus 4.6") {
		t.Fatalf("screenshot startup view should keep Hermes blank row between welcome tip and status rule:\n%s", got)
	}
	for _, stale := range []string{"Profile:", "CWD:", "Gormes Agent", "profile main", "main ❯", "Type a message and hit Enter…", "─ ready"} {
		if strings.Contains(got, stale) {
			t.Fatalf("screenshot startup view leaked old Gormes chrome %q:\n%s", stale, got)
		}
	}
}

func TestHermesChrome_WelcomePanelTitleCarriesBuildProvenance(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "claude-opus-4-6", SessionID: "sess-build"}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{
		WelcomeVersion:          "0.2.24",
		WelcomeVersionDateAlias: "v2026.6.5",
		WelcomeGitCommit:        "d221e369abcdef",
	})
	m.width = 120
	m.height = 28
	m.frame = f

	got := m.View()
	if !strings.Contains(got, "Gormes v0.2.24 (2026.6.5) · upstream d221e369") {
		t.Fatalf("welcome panel title missing Hermes-style build provenance:\n%s", got)
	}
}

func TestHermesChrome_WelcomePanelShowsGormesHomeInsteadOfWorkspaceCWD(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "gpt-5.5", SessionID: "sess-home"}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{ProfileBaseHome: "/home/example/.gormes"})
	m.width = 100
	m.height = 28
	m.frame = f

	got := m.View()
	if !strings.Contains(got, "/home/example/.gormes") {
		t.Fatalf("welcome panel should show Gormes home like Hermes shows its agent home:\n%s", got)
	}
}

func TestHermesChrome_ProfileDoesNotLeakIntoHermesChatChrome(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "gpt-5.5", SessionID: "sess-profile"}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{ProfileName: "mineru"})
	m.width = 120
	m.height = 30
	m.frame = f

	got := m.View()
	if !strings.Contains(got, "❯") {
		t.Fatalf("View missing bare Hermes prompt:\n%s", got)
	}
	for _, stale := range []string{"Profile:", "mineru ❯", "profile mineru", " · ~/"} {
		if strings.Contains(got, stale) {
			t.Fatalf("profile/cwd marker %q leaked into Hermes chat chrome:\n%s", stale, got)
		}
	}
}

func TestHermesChrome_CompactTranscriptDoesNotLeakProfileLabel(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "gpt-5.5", SessionID: "sess-compact-profile"}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{ProfileName: "mineru", CompactTranscript: true})
	m.width = 80
	m.height = 20
	m.frame = f

	got := m.View()
	if strings.Contains(got, "profile mineru") || strings.Contains(got, "profile: mineru") || strings.Contains(got, "mineru ❯") {
		t.Fatalf("compact transcript leaked profile label into Hermes chat chrome:\n%s", got)
	}
	if !strings.Contains(got, "⚕ Gormes · /help for commands") {
		t.Fatalf("compact transcript missing neutral Hermes-style intro:\n%s", got)
	}
}

func TestHermesChrome_DoesNotDuplicateProfileComposerPrompt(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "gpt-5.5", SessionID: "sess-profile"}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{
		ProfileName:   "mineru",
		StartupNotice: "session: temporary (sessions.db busy)",
	})
	m.height = 30
	m.frame = f

	for _, width := range []int{40, 45, 50, 55, 60, 65, 72, 90} {
		m.width = width
		got := m.View()
		if strings.Contains(got, "mineru ❯") {
			t.Fatalf("width=%d: View leaked profile label into Hermes composer prompt:\n%s", width, got)
		}
		if strings.Contains(got, "profile mineru") || strings.Contains(got, "profile: mineru") {
			t.Fatalf("width=%d: View leaked profile label into compact Hermes chat chrome:\n%s", width, got)
		}
		if !strings.Contains(got, "❯") {
			t.Fatalf("width=%d: View missing bare Hermes composer prompt:\n%s", width, got)
		}
	}
}

func TestHermesChrome_ScreenshotTranscriptTurnSpacing(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-transcript", History: []llm.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "Hi! How can I help?"},
		{Role: "user", Content: "whats ur name"},
		{Role: "assistant", Content: "I’m Gorm."},
	}}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 100
	m.height = 30
	m.frame = f

	got := m.View()
	wantTranscript := "  ❯ hi\n\n┊ Hi! How can I help?\n\n───\n\n  ❯ whats ur name\n\n┊ I’m Gorm."
	if !strings.Contains(got, wantTranscript) {
		t.Fatalf("View transcript spacing does not match Hermes screenshot block %q:\n%s", wantTranscript, got)
	}
	if !strings.Contains(got, "┊ I’m Gorm.\n\n ⚕ gpt 5.5") {
		t.Fatalf("Hermes transcript should leave one blank spacer row before the status/prompt stack:\n%s", got)
	}
	assertContainsInOrder(t, got,
		"  ❯ hi",
		"┊ Hi! How can I help?",
		"───",
		"  ❯ whats ur name",
		"┊ I’m Gorm.",
	)
	for _, stale := range []string{"you:", "assistant:", "⚕ Gormes", "gormes:"} {
		if strings.Contains(got, stale) {
			t.Fatalf("View leaked old transcript label %q instead of Hermes Ink gutter:\n%s", stale, got)
		}
	}
}

func TestHermesChrome_IdleStatusMessageUsesStatusRuleNotSeparateHintRow(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-notice"}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 100
	m.height = 24
	m.frame = f
	m.statusMessage = "profile main"

	got := m.View()
	if !strings.Contains(got, "profile main │ gpt 5.5") {
		t.Fatalf("idle status notice should render inside Hermes status rule instead of a separate hint row:\n%s", got)
	}
	if strings.Contains(got, "\nprofile main\n") {
		t.Fatalf("idle status notice rendered as separate old Gormes hint row:\n%s", got)
	}
}

func TestHermesChrome_ScreenshotBottomStackMatchesHermesPromptChrome(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-bottom", History: []llm.Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "Hi! How can I help?"},
	}}
	frames <- f
	m := NewModelWithOptions(frames, func(string) {}, func() {}, Options{ProfileName: "main"})
	m.width = 100
	m.height = 24
	m.frame = f
	m.sessionStartedAt = time.Unix(1_000, 0)
	m.statusNow = func() time.Time { return time.Unix(1_001, 0) }

	got := m.View()
	lines := strings.Split(got, "\n")
	statusIdx, promptIdx := -1, -1
	for i, line := range lines {
		if strings.Contains(line, "⚕ gpt 5.5") {
			statusIdx = i
		}
		if strings.TrimSpace(line) == "❯" {
			promptIdx = i
		}
	}
	if statusIdx < 0 || promptIdx < 0 {
		t.Fatalf("View missing Hermes status/prompt stack:\n%s", got)
	}
	if lines[promptIdx] != "❯ " {
		t.Fatalf("Hermes composer prompt line should reserve one cursor gap after the glyph, got %q:\n%s", lines[promptIdx], got)
	}
	for _, want := range []string{"ctx --", "[░░░░░░░░░░] --", "│ 1s │ ⏲ 0s"} {
		if !strings.Contains(lines[statusIdx], want) {
			t.Fatalf("Hermes status line missing screenshot fragment %q in %q:\n%s", want, lines[statusIdx], got)
		}
	}
	if promptIdx != statusIdx+2 {
		t.Fatalf("Hermes prompt should be directly below status + one rule row, got status=%d prompt=%d:\n%s", statusIdx, promptIdx, got)
	}
	for _, idx := range []int{statusIdx + 1, promptIdx + 1} {
		if idx >= len(lines) || strings.Trim(lines[idx], "─") != "" || lipgloss.Width(strings.TrimSpace(lines[idx])) != m.width {
			t.Fatalf("prompt rule line %d should be continuous full-width rule around prompt, got %q:\n%s", idx, lines[idx], got)
		}
	}
	for _, stale := range []string{"─ ready", "main ❯ Type a message", "profile main ·", "profile: main"} {
		if strings.Contains(got, stale) {
			t.Fatalf("View leaked old Gormes bottom chrome %q:\n%s", stale, got)
		}
	}
}

func TestHermesChrome_LongMultilineDraftExpandsLikeHermesInputHeight(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-long-multiline"}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 80
	m.height = 28
	m.frame = f
	m.editor.SetValue("one\ntwo\nthree\nfour\nfive")

	got := m.View()
	if !strings.Contains(got, "  five") {
		t.Fatalf("Hermes inputVisualHeight expands beyond four draft rows; Gormes clipped line five:\n%s", got)
	}
	if strings.Contains(got, "\n❯ five") {
		t.Fatalf("long multiline draft repeated prompt glyph on continuation row:\n%s", got)
	}
}

func TestHermesChrome_WrappedDraftComposerExpandsInsideDoubleRules(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-wrap-draft"}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 32
	m.height = 20
	m.frame = f
	m.editor.SetValue("this is a long wrapped composer draft")

	got := m.View()
	if !strings.Contains(got, "composer draft") {
		t.Fatalf("wrapped composer draft should expand vertically inside Hermes double-rule stack instead of clipping tail:\n%s", got)
	}
	if strings.Contains(got, "\n❯ composer") || strings.Contains(got, "\n❯ draft") {
		t.Fatalf("wrapped composer continuation repeated prompt glyph instead of using Hermes prompt blank:\n%s", got)
	}
}

func TestHermesChrome_MultilineDraftContinuationUsesPromptBlankInsideDoubleRules(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-multiline-draft"}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 80
	m.height = 24
	m.frame = f
	m.editor.SetValue("hello\nworld")

	got := m.View()
	want := strings.Repeat("─", m.width) + "\n❯ hello\n  world\n" + strings.Repeat("─", m.width)
	if !strings.Contains(got, want) {
		t.Fatalf("multiline composer should use Hermes prompt-blank continuation inside double-rule stack; want block %q:\n%s", want, got)
	}
	if strings.Contains(got, "\n❯ world\n") {
		t.Fatalf("multiline composer repeated prompt glyph on continuation row instead of Hermes prompt blank:\n%s", got)
	}
}

func TestHermesChrome_DraftComposerKeepsDoubleRuleStack(t *testing.T) {
	frames := make(chan kernel.RenderFrame, 1)
	f := kernel.RenderFrame{Phase: kernel.PhaseIdle, Model: "openai/gpt-5-5", SessionID: "sess-draft"}
	frames <- f
	m := NewModel(frames, func(string) {}, func() {})
	m.width = 88
	m.height = 24
	m.frame = f
	m.editor.SetValue("whats ur name")

	got := m.View()
	assertPromptRulePair(t, got, m.width, "❯ whats ur name")
	for _, stale := range []string{"main ❯", "Type a message and hit Enter…", "─ ready"} {
		if strings.Contains(got, stale) {
			t.Fatalf("draft composer leaked old Gormes chrome %q:\n%s", stale, got)
		}
	}
}

func TestHermesChrome_StyledEmptyPromptKeepsHermesCursorGap(t *testing.T) {
	got := RenderHermesChrome(HermesChromeInput{
		Width:        24,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "\x1b[36m❯\x1b[0m      ",
	})
	plain := StripANSIForTUI(got)
	wantStack := strings.Repeat("─", 24) + "\n❯ \n" + strings.Repeat("─", 24)
	if !strings.Contains(plain, wantStack) {
		t.Fatalf("styled empty prompt should preserve Hermes one-cell cursor gap in double-rule stack:\nraw=%q\nplain=%q", got, plain)
	}
}

func TestHermesChrome_InputPromptHasContinuousRulePair(t *testing.T) {
	got := RenderHermesChrome(HermesChromeInput{
		Width:        24,
		Conversation: "<<CONV>>",
		StatusBar:    "<<STATUS>>",
		Prompt:       "❯ Type a message",
	})

	lines := strings.Split(got, "\n")
	promptIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "❯ Type a message") {
			promptIdx = i
			break
		}
	}
	if promptIdx <= 0 || promptIdx+1 >= len(lines) {
		t.Fatalf("prompt must be wrapped by continuous rule lines:\n%s", got)
	}
	for _, idx := range []int{promptIdx - 1, promptIdx + 1} {
		trimmed := strings.TrimSpace(lines[idx])
		if trimmed != strings.Repeat("─", 24) {
			t.Fatalf("rule line %d = %q, want 24 continuous dashes around prompt:\n%s", idx, trimmed, got)
		}
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
	if strings.Contains(got, "❯ Type a message") {
		t.Fatalf("View leaked inline placeholder into bare Hermes composer prompt:\n%s", got)
	}
	if strings.Contains(got, "phase:") {
		t.Fatalf("View rendered debug phase chrome in idle state; Hermes keeps idle composer chrome quiet:\n%s", got)
	}
	if strings.Contains(got, "mouse: disabled") {
		t.Fatalf("View rendered persistent mouse-disabled noise in idle state:\n%s", got)
	}
	assertPromptRulePair(t, got, m.width, "❯")
}

func TestHermesChrome_InjectsStandaloneInputRulesAroundPrompt(t *testing.T) {
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
		assertPromptRulePair(t, got, width, "<<PROMPT>>")
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

func assertPromptRulePair(t *testing.T, got string, width int, promptNeedle string) {
	t.Helper()
	lines := strings.Split(got, "\n")
	promptIdx := -1
	for i, line := range lines {
		if !strings.Contains(line, promptNeedle) || i <= 0 || i+1 >= len(lines) {
			continue
		}
		if isStandaloneInputRuleLine(lines[i-1]) && isStandaloneInputRuleLine(lines[i+1]) {
			promptIdx = i
			break
		}
	}
	if promptIdx <= 0 || promptIdx+1 >= len(lines) {
		t.Fatalf("prompt %q must be wrapped by continuous rule lines:\n%s", promptNeedle, got)
	}
	rule := strings.Repeat("─", width)
	for _, idx := range []int{promptIdx - 1, promptIdx + 1} {
		if trimmed := strings.TrimSpace(lines[idx]); trimmed != rule {
			t.Fatalf("rule line %d = %q, want %q around prompt %q:\n%s", idx, trimmed, rule, promptNeedle, got)
		}
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
