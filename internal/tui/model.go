// Package tui renders the Gormes Dashboard. The TUI is a pure consumer of
// kernel.RenderFrame: it never assembles assistant text from raw provider
// events, never mutates kernel state directly. It sees the world only through
// the render channel; any user-originated events go back to the kernel via
// the Submit / Cancel callbacks provided by cmd/gormes/main.go.
package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// Submitter is the callback wired by main.go to enqueue a user turn on the
// kernel. Return value is intentionally omitted — kernel backpressure
// (ErrEventMailboxFull) is vanishingly rare with a 16-slot buffer and, when
// it does fire, the kernel itself surfaces the error on the next render
// frame. The TUI does NOT act on the return value; it just schedules.
type Submitter func(text string)

// Canceller is the callback wired by main.go to send PlatformEventCancel.
type Canceller func()

// Options carries local TUI settings that do not belong to kernel state.
type Options struct {
	MouseTracking bool
	MouseModeCmd  func(enabled bool) tea.Cmd
	// VoiceRecordKey is the Hermes-compatible voice.record_key value used
	// by the pure key resolver. Empty falls back to Ctrl+B.
	VoiceRecordKey string
	// SessionBranch is the injected fork helper invoked by the /branch
	// slash command. nil disables /branch (handler returns
	// `branch: store unavailable`); cmd/gormes wires the real
	// session.Fork-backed implementation in main.go.
	SessionBranch SessionBranchFunc
	// BusyGuard, when set, is consulted before every Enter-driven submit
	// or slash dispatch. While the underlying long-running CLI command
	// (e.g. /compress) is active, overlapping user input is rejected with
	// visible busy evidence so the kernel cannot receive a competing turn.
	// nil disables the busy check (legacy callers, kernel-only tests).
	BusyGuard BusyInputEvaluator
	// SessionExport is the injected canonical-transcript export helper
	// invoked by the /save slash command. The implementation is expected
	// to delegate to internal/transcript.ExportMarkdown over the
	// persisted session store; nil disables /save (handler returns
	// `save: store unavailable`). cmd/gormes wires the real
	// MemoryDBPath-backed implementation in main.go so the TUI never
	// opens a SQLite handle itself.
	SessionExport SessionExportFunc
	// ClipboardWrite is the injected clipboard writer invoked by /copy.
	// nil keeps copy unavailable; production callers may wire OSC52 or a
	// platform clipboard helper outside the TUI model.
	ClipboardWrite func(string) error
	// KanbanSlash is the injected local command runner invoked by /kanban.
	// nil keeps /kanban consumed with unavailable evidence; cmd/gormes wires
	// this to the same Cobra command tree as `gormes kanban`.
	KanbanSlash KanbanSlashFunc
	// TodoReader returns the current session's active todo items for the TUI
	// todo panel. nil disables the panel.
	TodoReader func(sessionID string) []TodoItem
	// OfflineSmoke keeps plain-text submits inside the TUI. It is used by
	// `gormes --offline` so the demo path proves the native UI without
	// contacting a provider or enqueueing a kernel turn.
	OfflineSmoke bool
	// StartupNotice is rendered in the normal TUI hint row after startup.
	// Runtime callers use it for recoverable degraded state that should not
	// scar terminal scrollback before Bubble Tea enters the alt screen.
	StartupNotice string
	// WelcomeVersion / WelcomeToolCount seed the session-aware welcome panel
	// with the operator-facing release version and agent tool count, which
	// are unreachable from internal/tui (main.Version is package main; the
	// tool count is absent from kernel.RenderFrame). Zero values keep the
	// R1 best-effort/omit behavior.
	WelcomeVersion   string
	WelcomeToolCount int
	WelcomeToolsets  []string
}

// BusyInputVerdict mirrors the cli.BusyInputVerdict shape for callers that
// only depend on internal/tui. The TUI does not import cli to keep the
// dependency one-way; instead it accepts any evaluator that produces this
// result. cli.BusyCommandGuard returns the equivalent struct via its
// EvaluateInput method.
type BusyInputVerdict struct {
	Rejected bool
	Evidence string
}

// BusyInputEvaluator decides whether editor input should be rejected because
// a long-running CLI command is currently executing. cli.BusyCommandGuard
// implements this interface; TUI tests can supply a fake. A nil evaluator
// disables the busy branch entirely so non-CLI surfaces (kernel-only tests)
// continue to work unchanged.
type BusyInputEvaluator interface {
	EvaluateInput(input string) BusyInputVerdict
}

// Model is the Bubble Tea state. The only external dependency is the
// read-side of the render channel (from kernel.Render()). Everything else
// is local UI state.
type Model struct {
	width, height int

	editor textarea.Model

	// frame is the latest RenderFrame received from the kernel. View() renders
	// this snapshot; Update() replaces it on every frameMsg.
	frame kernel.RenderFrame

	frames   <-chan kernel.RenderFrame
	submit   Submitter
	cancel   Canceller
	inFlight bool // true between a user submit and the next terminal frame

	mouseTracking  bool
	mouseModeCmd   func(enabled bool) tea.Cmd
	voiceRecordKey string
	statusMessage  string
	busyGuard      BusyInputEvaluator
	offlineSmoke   bool

	// sessionID, when non-empty, is the locally-tracked active session
	// owned by a successful /branch fork. SessionID() prefers it over
	// frame.SessionID so subsequent UI reads see the branch session even
	// before the kernel acks the switch on its next render frame.
	sessionID      string
	sessionBranch  SessionBranchFunc
	sessionExport  SessionExportFunc
	clipboardWrite func(string) error
	kanbanSlash    KanbanSlashFunc
	todoReader     func(sessionID string) []TodoItem

	slashRegistry *SlashRegistry

	// PanelState holds the active modal panel derived from the latest
	// RenderFrame. These fields are updated by Update() when a frameMsg
	// arrives so that View() can render the appropriate panel chrome.
	ApprovalState *kernel.KernelApprovalState
	ClarifyState  *kernel.KernelClarifyState
	SecretState   *kernel.KernelSecretState
}

// NewModel constructs the Bubble Tea model. frames is the kernel's Render()
// channel; submit/cancel are closures from main.go that forward to
// kernel.Submit with the appropriate PlatformEvent kind.
func NewModel(frames <-chan kernel.RenderFrame, submit Submitter, cancel Canceller) Model {
	return NewModelWithOptions(frames, submit, cancel, Options{MouseTracking: true})
}

// NewModelWithOptions constructs the Bubble Tea model with explicit local TUI
// options. cmd/gormes seeds these from config; tests can inject MouseModeCmd to
// assert terminal mode changes without a real terminal.
func NewModelWithOptions(frames <-chan kernel.RenderFrame, submit Submitter, cancel Canceller, opts Options) Model {
	// Seed the session-aware welcome panel from the caller (cmd/gormes wires
	// the real release version + agent tool count). Zero values are safe and
	// keep the R1 best-effort/omit behavior.
	SetWelcomeContext(opts.WelcomeVersion, opts.WelcomeToolCount, opts.WelcomeToolsets...)

	ta := textarea.New()
	ta.Placeholder = "Type a message and hit Enter…"
	ta.ShowLineNumbers = false
	// Match Hermes prompt_toolkit prompt symbol so the bottom-pinned chrome
	// shows the operator-recognisable `❯ ` glyph at the start of every input
	// line instead of the textarea default cursor marker.
	normalPrompt, _ := DefaultHermesSkin().PromptSymbols("default")
	ta.Prompt = normalPrompt
	ta.SetHeight(1)
	ta.Focus()
	return Model{
		editor:         ta,
		frames:         frames,
		submit:         submit,
		cancel:         cancel,
		mouseTracking:  opts.MouseTracking,
		mouseModeCmd:   opts.MouseModeCmd,
		voiceRecordKey: opts.VoiceRecordKey,
		sessionBranch:  opts.SessionBranch,
		busyGuard:      opts.BusyGuard,
		statusMessage:  opts.StartupNotice,
		sessionExport:  opts.SessionExport,
		clipboardWrite: opts.ClipboardWrite,
		kanbanSlash:    opts.KanbanSlash,
		todoReader:     opts.TodoReader,
		offlineSmoke:   opts.OfflineSmoke,
		slashRegistry:  NewDefaultSlashRegistry(),
	}
}

// SessionID returns the model's active session identifier. A locally-tracked
// branch session (set by the /branch slash handler) takes precedence over the
// kernel-supplied frame.SessionID so the TUI surfaces the fork target even
// before the next render frame arrives.
func (m *Model) SessionID() string {
	if m.sessionID != "" {
		return m.sessionID
	}
	return m.frame.SessionID
}

// frameMsg wraps an incoming kernel.RenderFrame as a Bubble Tea message so
// Update() can handle it via the normal msg switch.
type frameMsg kernel.RenderFrame

// waitFrame returns a tea.Cmd that blocks on the render channel once and
// converts the next frame into a frameMsg. Update() re-schedules it after
// handling each frameMsg so the pump never stops. If the channel closes
// (kernel exit), we return tea.Quit to unwind cleanly.
func (m Model) waitFrame() tea.Cmd {
	return func() tea.Msg {
		f, ok := <-m.frames
		if !ok {
			return tea.QuitMsg{}
		}
		return frameMsg(f)
	}
}

// Init is the Bubble Tea entry point. We start the cursor blink and the
// first render-frame wait in parallel.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{textarea.Blink, m.waitFrame()}
	if !m.mouseTracking {
		cmds = append(cmds, m.emitMouseModeCmd(false))
	}
	return tea.Batch(cmds...)
}
