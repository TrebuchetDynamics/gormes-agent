// Package tui renders the Gormes Dashboard. The TUI is a pure consumer of
// kernel.RenderFrame: it never assembles assistant text from raw provider
// events, never mutates kernel state directly. It sees the world only through
// the render channel; any user-originated events go back to the kernel via
// the Submit / Cancel callbacks provided by cmd/gormes/main.go.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/banner"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessionspage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/toolsview"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/usagepage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/voice"
)

// Submitter is the callback wired by main.go to enqueue a user turn on the
// kernel. Return value is intentionally omitted — kernel backpressure
// (ErrEventMailboxFull) is vanishingly rare with a 16-slot buffer and, when
// it does fire, the kernel itself surfaces the error on the next render
// frame. The TUI does NOT act on the return value; it just schedules.
type Submitter func(text string)

// Canceller is the callback wired by main.go to send PlatformEventCancel.
type Canceller func()

// Steerer is the callback wired by main.go to send PlatformEventSteer while a
// turn is active. Nil means steer-mode drafts fall back to the next-turn queue.
type Steerer func(text string)

// SetSessionModelFunc is the TUI-local bridge to the kernel's resident
// in-session model override. It applies to future turns without resetting the
// current transcript.
type SetSessionModelFunc func(provider, model string) error

// SessionResetFunc is the TUI-local bridge to the kernel's reset-session seam.
// It clears the active conversation only when the kernel is idle/failed.
type SessionResetFunc func() error

// GatewayLogTailFunc returns the most recent gateway log lines for /logs.
// The native TUI owns display and limit parsing; production callers own the
// concrete log source so internal/tui stays free of HTTP and config-path I/O.
type GatewayLogTailFunc func(limit int) (string, error)

// SessionTitleResult is the TUI-local response shape for /title. Title is the
// current or newly persisted title. Pending mirrors Hermes' session.title RPC
// response for adapters that defer the write while a session initializes.
type SessionTitleResult struct {
	Title   string
	Pending bool
}

// SessionTitleFunc gets or sets the active session title for /title. An empty
// title argument queries the current title; a non-empty title persists an
// operator-chosen title. Production callers own persistence so internal/tui
// remains free of session DB I/O.
type SessionTitleFunc func(sessionID, title string) (SessionTitleResult, error)

// SessionDirectoryEntry is the TUI read model for the local /sessions and
// /resume picker page. Production callers map their durable session directory
// into this small shape so internal/tui stays free of SQLite and config paths.
type SessionDirectoryEntry = sessionspage.Entry

// SessionDirectoryFunc returns recent sessions for the native TUI picker page.
// The limit is caller-supplied and already clamped by the slash handler.
type SessionDirectoryFunc func(limit int) ([]SessionDirectoryEntry, error)

// SessionResumeResult is the replay payload returned by the local /resume
// adapter. History is already ordered for direct render-frame replacement.
type SessionResumeResult struct {
	SessionID string
	History   []llm.Message
}

// SessionResumeFunc resolves an operator-supplied id/prefix and switches the
// resident runtime session. Production callers own persistence and kernel I/O
// so internal/tui stays free of SQLite/config paths and kernel mutation policy.
type SessionResumeFunc func(ctx context.Context, query string) (SessionResumeResult, error)

// AccountUsageFunc fetches provider account-usage evidence for /usage. It is
// injected so internal/tui can render provider limits/cost evidence without
// knowing config paths, credentials, HTTP clients, or provider-specific policy.
type AccountUsageFunc func(ctx context.Context) (llm.AccountUsageSnapshot, error)

// ToolsConfigureRequest is the TUI-local request shape for /tools enable|disable.
// Production callers own config persistence and MCP/server policy; internal/tui
// only parses the operator slash input and renders the adapter evidence.
type ToolsConfigureRequest struct {
	Action    string
	Names     []string
	SessionID string
}

// ToolsConfigureResult mirrors the Hermes ui-tui tools.configure response
// fields that affect visible TUI output.
type ToolsConfigureResult = toolsview.Result

// ToolsConfigureFunc updates the active TUI tool configuration for /tools.
// It is injected so internal/tui never writes config files directly.
type ToolsConfigureFunc func(ToolsConfigureRequest) (ToolsConfigureResult, error)

// SkillSlashReloadResult is the native TUI reload payload for /reload-skills.
// Commands replaces the dynamic /skill-name registry; Output is operator-facing
// evidence rendered in the status line.
type SkillSlashReloadResult struct {
	Commands []skills.SkillSlashCommand
	Output   string
}

// SkillSlashReloadFunc refreshes dynamic skill slash commands for the native
// TUI. Production callers own filesystem/config access; internal/tui only
// swaps the in-memory registry.
type SkillSlashReloadFunc func(context.Context) (SkillSlashReloadResult, error)

// VoiceToggleRequest is the TUI-local request shape for /voice status|on|off|tts.
// Production callers own runtime voice state, setup checks, and config access;
// internal/tui only parses slash input and renders adapter evidence.
type VoiceToggleRequest = voice.Request

// VoiceToggleResult mirrors the Hermes ui-tui voice.toggle response fields
// that affect visible output and frontend record-key state.
type VoiceToggleResult = voice.Result

// VoiceToggleFunc updates or reads local voice mode state for /voice. It is
// injected so internal/tui never starts live audio or writes config files.
type VoiceToggleFunc func(VoiceToggleRequest) (VoiceToggleResult, error)

type VoiceRecordRequest struct {
	SessionID string
}

type VoiceRecordEvidence struct {
	Code    string
	Message string
}

type VoiceAudioArtifact struct {
	ID       string
	MIMEType string
	Bytes    []byte
	Path     string
}

type VoiceTranscript struct {
	Text string
}

type VoiceRecorder interface {
	Start(context.Context, VoiceRecordRequest) (VoiceRecordEvidence, error)
	Stop(context.Context, VoiceRecordRequest) (VoiceAudioArtifact, VoiceRecordEvidence, error)
}

type VoiceTranscriber interface {
	Transcribe(context.Context, VoiceAudioArtifact) (VoiceTranscript, VoiceRecordEvidence, error)
}

type VoicePlayback interface {
	Speak(context.Context, string) (VoiceRecordEvidence, error)
}

type VoiceRuntime struct {
	Recorder    VoiceRecorder
	Transcriber VoiceTranscriber
	Playback    VoicePlayback
}

func (r VoiceRuntime) empty() bool {
	return r.Recorder == nil && r.Transcriber == nil && r.Playback == nil
}

// SkinConfigRequest is the TUI-local request shape for /skin. Name is empty
// for read-only status and non-empty for a requested skin switch.
type SkinConfigRequest = skin.Request

// SkinConfigResult mirrors the Hermes config.get/config.set skin response
// value that affects visible output and active native TUI skin state.
type SkinConfigResult = skin.Result

// SkinConfigFunc gets or sets the active display skin for /skin. Production
// callers own persistence so internal/tui never writes config files directly.
type SkinConfigFunc func(SkinConfigRequest) (SkinConfigResult, error)

// Options carries local TUI settings that do not belong to kernel state.
type Options struct {
	MouseTracking bool
	MouseModeCmd  func(enabled bool) tea.Cmd
	// VoiceRecordKey is the Hermes-compatible voice.record_key value used
	// by the pure key resolver. Empty falls back to Ctrl+B.
	VoiceRecordKey string
	// VoiceToggle is the injected status/toggle adapter invoked by /voice.
	// nil keeps /voice consumed with visible setup evidence; cmd/gormes wires
	// a no-live-audio adapter for local TUI startup.
	VoiceToggle VoiceToggleFunc
	// VoiceRuntime owns fakeable live-audio orchestration for the native TUI.
	// Production callers may wire real recorder/STT/TTS adapters; tests use
	// hermetic fakes so internal/tui never opens audio devices directly.
	VoiceRuntime VoiceRuntime
	// SkinName seeds the active native TUI skin. Empty or unknown values fall
	// back to default so invalid persisted config cannot break startup.
	SkinName string
	// SkinConfig is the injected get/set adapter invoked by /skin. nil keeps
	// the command consumed with visible degraded evidence; cmd/gormes wires a
	// config-backed adapter for local TUI startup.
	SkinConfig SkinConfigFunc
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
	// to delegate to internal/persistence/transcript.ExportMarkdown over the
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
	// SkillsCommand is the injected local command runner invoked by /skills.
	// nil falls back to gateway.HandleSkillsCommand so read-only skills commands
	// still work in tests and legacy callers; cmd/gormes wires URL install seams.
	SkillsCommand func(string) string
	// SkillSlashCommands are Hermes-compatible dynamic /skill-name invocations.
	// Built-in slash handlers keep precedence; prompt templates may not shadow
	// these commands.
	SkillSlashCommands []skills.SkillSlashCommand
	// SkillSlashReload refreshes SkillSlashCommands for /reload-skills.
	// nil keeps the command consumed with visible unavailable evidence.
	SkillSlashReload SkillSlashReloadFunc
	// PromptTemplates are local operator-authored Markdown snippets exposed as
	// slash expansions. They never override built-in slash handlers and expand
	// into editable composer text rather than submitting to the model directly.
	PromptTemplates prompttemplates.Catalog
	// GatewayLogTail is the injected gateway log-tail reader invoked by /logs.
	// nil keeps /logs consumed with `no gateway logs`; cmd/gormes wires a
	// bounded live-gateway/file-fallback adapter for local TUI startup.
	GatewayLogTail GatewayLogTailFunc
	// SessionTitle is the injected get/set adapter invoked by /title. nil keeps
	// /title consumed with visible degraded evidence; cmd/gormes wires a
	// metadata-backed adapter for local TUI startup.
	SessionTitle SessionTitleFunc
	// SessionDirectory is the injected recent-session lister invoked by
	// /sessions and /resume. nil keeps the commands consumed with visible
	// degraded evidence; cmd/gormes wires a Goncho memory-backed adapter for
	// local TUI startup.
	SessionDirectory SessionDirectoryFunc
	// AccountUsage is the injected provider account-usage fetcher invoked by
	// /usage after the local frame-telemetry page is opened. nil keeps /usage
	// local-only; cmd/gormes wires a provider adapter for local TUI startup.
	AccountUsage AccountUsageFunc
	// ToolsConfigure is the injected config adapter invoked by /tools
	// enable|disable. nil keeps the command consumed with visible degraded
	// evidence; cmd/gormes wires a platform_toolsets-backed adapter for local
	// TUI startup.
	ToolsConfigure ToolsConfigureFunc
	// SessionResume is the injected id/prefix resolver + resident-session
	// switcher invoked by `/resume <id-or-prefix>`. nil keeps argument-based
	// resume consumed with visible degraded evidence.
	SessionResume SessionResumeFunc
	// SessionTree is the injected lineage/label tree reader invoked by /tree.
	// Production callers own session metadata/transcript I/O; internal/tui only
	// renders the tree and dispatches typed label/restore requests.
	SessionTree        SessionTreeFunc
	SessionTreeLabel   SessionTreeLabelFunc
	SessionTreeRestore SessionTreeRestoreFunc
	// SetSessionModel is the injected kernel apply seam invoked by /model.
	// nil keeps /model consumed with visible degraded evidence.
	SetSessionModelFunc SetSessionModelFunc
	// ModelPickerCatalog provides the TUI-local provider/model choices for
	// /model. Local TUI startup wires DefaultModelPickerCatalog; remote TUI
	// intentionally leaves it nil.
	ModelPickerCatalog ModelPickerCatalogFunc
	// ModelProvider and ModelName seed the local picker with the active
	// provider/model before the first kernel frame arrives.
	ModelProvider string
	ModelName     string
	// SessionReset is the injected kernel reset helper invoked by /clear and
	// /new. nil keeps those commands consumed with visible degraded evidence.
	SessionReset SessionResetFunc
	// CompactTranscript starts the transcript in the Hermes compact view mode.
	// /compact mutates this at runtime; tiny terminals still auto-compact even
	// when this option is false.
	CompactTranscript bool
	// StatusBarMode controls where the Hermes-compatible status rule renders.
	// Empty defaults to top, matching Hermes ui-tui's display default.
	StatusBarMode StatusBarMode
	// DetailsState controls visibility of Hermes detail sections (thinking,
	// tools, subagents, activity) in the native transcript renderer.
	DetailsState DetailsState
	// IndicatorStyle controls the running-turn busy indicator glyph family.
	// Empty defaults to kaomoji, matching Hermes ui-tui's display default.
	IndicatorStyle IndicatorStyle
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
	// BusyInputMode controls Enter on plain text while a kernel turn is active:
	// interrupt (default), queue, or steer. Queue and steer keep drafts visible
	// in the bottom-pinned chrome until delivered or cleared.
	BusyInputMode HermesBusyInputMode
	// Steer is the injected active-turn guidance adapter used when BusyInputMode
	// is steer. Nil degrades to queue mode with visible evidence.
	Steer Steerer
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

	inputHistory    *HermesHistory
	slashCompletion slashCompletionState

	// frame is the latest RenderFrame received from the kernel. View() renders
	// this snapshot; Update() replaces it on every frameMsg.
	frame kernel.RenderFrame

	frames   <-chan kernel.RenderFrame
	submit   Submitter
	cancel   Canceller
	steer    Steerer
	inFlight bool // true between a user submit and the next terminal frame

	mouseTracking     bool
	mouseModeCmd      func(enabled bool) tea.Cmd
	voiceRecordKey    string
	activeSkinName    string
	activeSkin        HermesSkin
	statusMessage     string
	transientPage     *TransientPageState
	busyGuard         BusyInputEvaluator
	offlineSmoke      bool
	compactTranscript bool
	statusBarMode     StatusBarMode
	detailsState      DetailsState
	indicatorStyle    IndicatorStyle
	spinnerFrame      int
	busyInputMode     HermesBusyInputMode
	queuedMessages    QueuedMessages
	steeringMessages  QueuedMessages
	extensionUI       extensionUIState

	// sessionID, when non-empty, is the locally-tracked active session
	// owned by a successful /branch fork. SessionID() prefers it over
	// frame.SessionID so subsequent UI reads see the branch session even
	// before the kernel acks the switch on its next render frame.
	sessionID          string
	sessionBranch      SessionBranchFunc
	sessionExport      SessionExportFunc
	clipboardWrite     func(string) error
	kanbanSlash        KanbanSlashFunc
	skillsCommand      func(string) string
	skillSlashCommands []skills.SkillSlashCommand
	skillSlashReload   SkillSlashReloadFunc
	promptTemplates    prompttemplates.Catalog
	gatewayLogTail     GatewayLogTailFunc
	sessionTitle       SessionTitleFunc
	sessionDirectory   SessionDirectoryFunc
	sessionTree        SessionTreeFunc
	sessionTreeLabel   SessionTreeLabelFunc
	sessionTreeRestore SessionTreeRestoreFunc
	accountUsage       AccountUsageFunc
	toolsConfigure     ToolsConfigureFunc
	voiceToggle        VoiceToggleFunc
	voiceRuntime       VoiceRuntime
	voiceRecording     bool
	voiceProcessing    bool
	voiceLastSpokenSeq uint64
	skinConfig         SkinConfigFunc
	sessionResume      SessionResumeFunc
	todoReader         func(sessionID string) []TodoItem

	slashRegistry *SlashRegistry

	setSessionModel    SetSessionModelFunc
	modelPickerCatalog ModelPickerCatalogFunc
	modelProvider      string
	modelName          string
	sessionReset       SessionResetFunc
	modelPicker        *ModelPickerState
	modelPickerChoices []ModelPickerCatalogProvider

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
	banner.SetWelcomeContext(opts.WelcomeVersion, opts.WelcomeToolCount, opts.WelcomeToolsets...)

	skin := DefaultHermesSkin()
	if resolved, ok := ResolveBuiltinSkin(opts.SkinName); ok {
		skin = resolved
	}
	ta := textarea.New()
	ta.Placeholder = "Type a message and hit Enter…"
	ta.ShowLineNumbers = false
	// Match Hermes prompt_toolkit prompt symbol so the bottom-pinned chrome
	// shows the operator-recognisable skin prompt glyph at the start of every
	// input line instead of the textarea default cursor marker.
	normalPrompt, _ := skin.PromptSymbols("default")
	ta.Prompt = normalPrompt
	ApplyTextareaSkin(&ta, skin)
	ta.SetHeight(1)
	ta.Focus()
	return Model{
		editor:             ta,
		inputHistory:       NewHermesHistory(),
		frames:             frames,
		submit:             submit,
		cancel:             cancel,
		steer:              opts.Steer,
		mouseTracking:      opts.MouseTracking,
		mouseModeCmd:       opts.MouseModeCmd,
		voiceRecordKey:     opts.VoiceRecordKey,
		activeSkinName:     skin.Name,
		activeSkin:         skin,
		sessionBranch:      opts.SessionBranch,
		busyGuard:          opts.BusyGuard,
		statusMessage:      opts.StartupNotice,
		sessionExport:      opts.SessionExport,
		clipboardWrite:     opts.ClipboardWrite,
		kanbanSlash:        opts.KanbanSlash,
		skillsCommand:      opts.SkillsCommand,
		skillSlashCommands: opts.SkillSlashCommands,
		skillSlashReload:   opts.SkillSlashReload,
		promptTemplates:    opts.PromptTemplates,
		gatewayLogTail:     opts.GatewayLogTail,
		sessionTitle:       opts.SessionTitle,
		sessionDirectory:   opts.SessionDirectory,
		sessionTree:        opts.SessionTree,
		sessionTreeLabel:   opts.SessionTreeLabel,
		sessionTreeRestore: opts.SessionTreeRestore,
		accountUsage:       opts.AccountUsage,
		toolsConfigure:     opts.ToolsConfigure,
		voiceToggle:        opts.VoiceToggle,
		voiceRuntime:       opts.VoiceRuntime,
		skinConfig:         opts.SkinConfig,
		sessionResume:      opts.SessionResume,
		todoReader:         opts.TodoReader,
		setSessionModel:    opts.SetSessionModelFunc,
		modelPickerCatalog: opts.ModelPickerCatalog,
		modelProvider:      opts.ModelProvider,
		modelName:          opts.ModelName,
		sessionReset:       opts.SessionReset,
		compactTranscript:  opts.CompactTranscript,
		statusBarMode:      normalizeStatusBarMode(opts.StatusBarMode),
		busyInputMode:      normalizeHermesBusyInputMode(opts.BusyInputMode),
		detailsState:       NormalizeDetailsState(opts.DetailsState),
		indicatorStyle:     NormalizeIndicatorStyle(string(opts.IndicatorStyle)),
		offlineSmoke:       opts.OfflineSmoke,
	}.withRebuiltSlashRegistry()
}

func (m Model) withRebuiltSlashRegistry() Model {
	m.rebuildSlashRegistry()
	return m
}

func (m *Model) rebuildSlashRegistry() {
	r := NewDefaultSlashRegistry()
	r.RegisterSkillSlashCommands(m.skillSlashCommands)
	r.RegisterPromptTemplates(m.promptTemplates)
	m.slashRegistry = r
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

// ─── model picker types and methods (consolidated from slash_model.go) ─────

type modelSessionSetMsg struct {
	Provider string
	Model    string
	Err      error
}

func (m *Model) loadModelPickerCatalog() ([]ModelPickerCatalogProvider, error) {
	fn := m.modelPickerCatalog
	if fn == nil {
		return nil, fmt.Errorf("no model catalog configured")
	}
	catalog, err := fn()
	if err != nil {
		return nil, err
	}
	return normalizeModelPickerCatalog(catalog), nil
}

func (m *Model) updateModelPickerForKey(msg tea.KeyMsg) tea.Cmd {
	if m.modelPicker == nil {
		return nil
	}
	next, cmd := UpdateModelPicker(msg, *m.modelPicker)
	next.Width = m.width
	next.Height = m.height
	if next.SelectedProviderIndex >= 0 && next.SelectedProviderIndex < len(m.modelPickerChoices) {
		next.Models = modelsForProviderIndex(m.modelPickerChoices, next.SelectedProviderIndex)
		if next.SelectedModelIndex >= len(next.Models) {
			next.SelectedModelIndex = len(next.Models) - 1
		}
	}
	m.modelPicker = &next
	return cmd
}

func (m *Model) handleModelPickerConfirmed(result ModelPickerResult) tea.Cmd {
	m.modelPicker = nil
	if strings.TrimSpace(result.Provider) == "" {
		m.statusMessage = "model: unchanged"
		return nil
	}
	provider, model := m.normalizeConfirmedModelSelection(result.Provider, result.Model)
	if strings.TrimSpace(model) == "" {
		m.statusMessage = "model: no model selected"
		return nil
	}
	res := m.applyModelSelection(provider, model)
	if res.StatusMessage != "" {
		m.statusMessage = res.StatusMessage
	}
	return res.Cmd
}

func (m *Model) normalizeConfirmedModelSelection(provider, model string) (string, string) {
	return modelpicker.NormalizeConfirmedSelection(m.modelPickerChoices, provider, model)
}

func (m *Model) applyModelSelection(provider, model string) SlashResult {
	model = strings.TrimSpace(model)
	if model == "" {
		return SlashResult{Handled: true, StatusMessage: "model: no model selected"}
	}
	provider = strings.TrimSpace(provider)
	if m.setSessionModel == nil {
		return SlashResult{Handled: true, StatusMessage: "model: switch unavailable"}
	}
	return SlashResult{
		Handled: true,
		Cmd: func() tea.Msg {
			err := m.setSessionModel(provider, model)
			return modelSessionSetMsg{Provider: provider, Model: model, Err: err}
		},
	}
}

func (m *Model) handleModelSessionSet(msg modelSessionSetMsg) {
	if msg.Err != nil {
		m.statusMessage = "model: " + msg.Err.Error()
		return
	}
	provider := strings.TrimSpace(msg.Provider)
	model := strings.TrimSpace(msg.Model)
	if provider != "" {
		m.modelProvider = provider
	}
	if model != "" {
		m.modelName = model
		m.frame.Model = model
	}
	m.statusMessage = fmt.Sprintf("model -> %s", model)
}

func (m *Model) currentModelProvider() string {
	return strings.TrimSpace(m.modelProvider)
}

func (m *Model) currentModelName() string {
	if model := strings.TrimSpace(m.frame.Model); model != "" {
		return model
	}
	return strings.TrimSpace(m.modelName)
}

// ─── usage account types and methods (consolidated from slash_usage.go) ─────

const (
	usageAccountTimeout     = 30 * time.Second
	usageAccountLoadingLine = usagepage.AccountLoadingLine
)

type usageAccountMsg struct {
	Lines []string
	Err   error
}

func (m *Model) usageAccountCmd() tea.Cmd {
	fn := m.accountUsage
	if fn == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), usageAccountTimeout)
		defer cancel()
		snapshot, err := fn(ctx)
		if err != nil {
			return usageAccountMsg{Err: err}
		}
		return usageAccountMsg{Lines: llm.RenderAccountUsageLines(snapshot, llm.AccountUsageRenderOptions{})}
	}
}

func (m *Model) handleUsageAccount(msg usageAccountMsg) {
	lines := msg.Lines
	if msg.Err != nil {
		lines = []string{"Provider: unavailable", "Usage unavailable: " + msg.Err.Error()}
		m.statusMessage = "usage account unavailable: " + msg.Err.Error()
	} else {
		m.statusMessage = "usage account updated"
	}
	if m.transientPage == nil || m.transientPage.Title != "Usage" {
		return
	}
	m.transientPage.Body = replaceUsageAccountLoading(m.transientPage.Body, lines)
}

// ─── end of consolidated model methods ──────────────────────────────────────

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

// RunningPlaceholder returns the idle editor placeholder text, customized with
// any busy-available slash commands.
func (m Model) RunningPlaceholder() string {
	var busySlashes []string
	if m.slashRegistry != nil {
		busySlashes = m.slashRegistry.BusyAvailableSlashes()
	}
	return composer.RunningPlaceholder(m.inFlight, busySlashes)
}

func (m Model) emitMouseModeCmd(enabled bool) tea.Cmd {
	if m.mouseModeCmd != nil {
		return m.mouseModeCmd(enabled)
	}
	return defaultMouseModeCmd(enabled)
}
