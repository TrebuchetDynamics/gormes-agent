package tui

import (
	"context"
	"fmt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/branch"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/browser"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/compact"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/composer"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/details"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/help"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/historypage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/indicator"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/kanban"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/logs"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/modelpicker"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/queue"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/reset"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/save"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessionspage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/sessiontree"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skillsslash"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/skin"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/slashcompletion"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statusbar"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/statuspage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/terminal"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/title"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/toolsview"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/usagepage"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/voice"
	"os"
	"sort"
	"strings"
	tea "github.com/charmbracelet/bubbletea"
	"time"
	uislash "github.com/TrebuchetDynamics/gormes-agent/internal/tui/slash"
)



// SlashResult is the typed return value of a SlashHandler. It tells Update
// whether the input was consumed (Handled), what status line to show
// (StatusMessage), and any tea.Cmd to schedule alongside the editor reset.
//
// When Handled is false, Update MUST forward the input to kernel.Submit so a
// buggy handler can never silently drop a turn.
type SlashResult struct {
	Handled       bool
	StatusMessage string
	Cmd           tea.Cmd
	// EditorText, when non-empty, seeds the editor after the slash command text
	// has been consumed. /tree restore uses this to make a prior user turn
	// editable without submitting it or mutating the active session.
	EditorText string
}

// SlashHandler implements one slash command. It receives the full editor
// input (including the leading slash) so it can parse subcommands without
// the registry having to split arguments. It receives *Model so it can
// read local TUI state and, for handlers like /mouse, mutate it directly
// before returning the resulting Cmd.
type SlashHandler func(input string, model *Model) SlashResult

// SlashRegistry routes slash commands typed in the editor to handlers.
// Construction is the only writer; Dispatch is read-only over the map, so
// no synchronization is required as long as Register is called before the
// Bubble Tea program starts.
type SlashRegistry struct {
	handlers         map[string]slashEntry
	consumeFallbacks bool
}

// slashEntry pairs a SlashHandler with its registration metadata. Today the
// only metadata is BusyAvailable, consumed by Model.RunningPlaceholder when
// the running-agent placeholder enumerates discoverable busy-time actions.
type slashEntry struct {
	handler       SlashHandler
	busyAvailable bool
}

// RegisterOpt mutates a slashEntry at registration time. Used as variadic
// arguments to Register so most callers can keep their two-argument form
// while busy-aware commands opt in via WithBusyAvailable().
type RegisterOpt func(*slashEntry)

// WithBusyAvailable marks a slash command as safe to invoke while a kernel
// turn is in flight. Model.RunningPlaceholder enumerates these in the
// in-flight placeholder so operators can discover them without docs.
func WithBusyAvailable() RegisterOpt {
	return func(e *slashEntry) {
		e.busyAvailable = true
	}
}

// NewSlashRegistry returns an empty registry. Most callers want
// NewDefaultSlashRegistry, which pre-registers /mouse, /scroll, and the
// /save stub.
func NewSlashRegistry() *SlashRegistry {
	return &SlashRegistry{handlers: make(map[string]slashEntry)}
}

// Register binds a handler to a slash command name. Names are stored
// case-insensitively and without the leading "/" so callers may pass either
// "save" or "/save". Optional RegisterOpts (e.g. WithBusyAvailable) opt the
// command into the busy-time placeholder list.
func (r *SlashRegistry) Register(name string, handler SlashHandler, opts ...RegisterOpt) {
	entry := slashEntry{handler: handler}
	for _, opt := range opts {
		opt(&entry)
	}
	r.handlers[normalizeSlashName(name)] = entry
}

// RegisterIfAbsent binds a handler only when no existing command owns the name.
func (r *SlashRegistry) RegisterIfAbsent(name string, handler SlashHandler, opts ...RegisterOpt) bool {
	key := normalizeSlashName(name)
	if _, exists := r.handlers[key]; exists {
		return false
	}
	r.Register(key, handler, opts...)
	return true
}

// CommandNames returns registered slash command names without leading slashes.
func (r *SlashRegistry) CommandNames() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// BusyAvailableSlashes returns the names (without the leading "/") of every
// slash command registered with WithBusyAvailable, in alphabetical order so
// the running-agent placeholder is stable across map iteration ordering.
func (r *SlashRegistry) BusyAvailableSlashes() []string {
	names := make([]string, 0, len(r.handlers))
	for name, entry := range r.handlers {
		if entry.busyAvailable {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Dispatch parses the first whitespace-separated token of input. If it starts
// with "/" and matches a registered handler, the handler runs with the full
// original input string. Default registries also consume unresolved slash-like
// input with visible guidance so command text cannot fall through to the model.
// Bare registries created by NewSlashRegistry keep the old opt-in behavior:
// unknown slash commands return Handled=false for focused handler tests.
func (r *SlashRegistry) Dispatch(input string, model *Model) SlashResult {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return SlashResult{}
	}
	first := fields[0]
	if !strings.HasPrefix(first, "/") {
		return SlashResult{}
	}
	name := normalizeSlashName(first)
	entry, ok := r.handlers[name]
	if !ok {
		if r.consumeFallbacks {
			return slashFallbackResult(input)
		}
		return SlashResult{}
	}
	return entry.handler(input, model)
}

func normalizeSlashName(name string) string {
	return uislash.NormalizeName(name)
}

// NewDefaultSlashRegistry returns a registry pre-populated with the local
// slash commands the Gormes TUI ships today: /help, /clear and /new over the
// reset seam, /compact over the transcript view mode, /mouse and its /scroll
// alias, /save over the canonical transcript export seam, /branch over the
// session fork seam, plus the model, browser, kanban, copy, and quit helpers.
func NewDefaultSlashRegistry() *SlashRegistry {
	r := NewSlashRegistry()
	r.consumeFallbacks = true
	r.Register("help", helpSlashHandler, WithBusyAvailable())
	r.Register("clear", sessionResetSlashHandler)
	r.Register("new", sessionResetSlashHandler)
	r.Register("compact", compactSlashHandler, WithBusyAvailable())
	r.Register("details", detailsSlashHandler, WithBusyAvailable())
	r.Register("indicator", indicatorSlashHandler, WithBusyAvailable())
	r.Register("history", historySlashHandler, WithBusyAvailable())
	r.Register("queue", queueSlashHandler, WithBusyAvailable())
	r.Register("q", queueSlashHandler, WithBusyAvailable())
	r.Register("logs", logsSlashHandler, WithBusyAvailable())
	r.Register("title", titleSlashHandler, WithBusyAvailable())
	r.Register("sessions", sessionsSlashHandler)
	r.Register("resume", sessionsSlashHandler)
	r.Register("tree", treeSlashHandler)
	r.Register("usage", usageSlashHandler, WithBusyAvailable())
	r.Register("mouse", mouseSlashHandler)
	r.Register("scroll", mouseSlashHandler)
	r.Register("save", saveSlashHandler)
	r.Register("branch", branchSlashHandler)
	r.Register("copy", copySlashHandler)
	r.Register("status", statusSlashHandler, WithBusyAvailable())
	r.Register("statusbar", statusbarSlashHandler, WithBusyAvailable())
	r.Register("sb", statusbarSlashHandler, WithBusyAvailable())
	r.Register("browser", browserSlashHandler, WithBusyAvailable())
	r.Register("kanban", kanbanSlashHandler, WithBusyAvailable())
	r.Register("skills", skillsSlashHandler, WithBusyAvailable())
	r.Register("reload-skills", reloadSkillsSlashHandler, WithBusyAvailable())
	r.Register("reload_skills", reloadSkillsSlashHandler, WithBusyAvailable())
	r.Register("tools", toolsSlashHandler, WithBusyAvailable())
	r.Register("voice", voiceSlashHandler, WithBusyAvailable())
	r.Register("skin", skinSlashHandler, WithBusyAvailable())
	r.Register("model", modelSlashHandler)
	r.Register("m", modelSlashHandler)
	r.Register("quit", quitSlashHandler)
	r.Register("exit", quitSlashHandler)
	r.Register("redraw", redrawSlashHandler, WithBusyAvailable())
	return r
}

// DefaultSlashCommandNames returns the built-in command names reserved before
// prompt-template registration so templates cannot shadow local/Gateway slash UX.
func DefaultSlashCommandNames() []string {
	return NewDefaultSlashRegistry().CommandNames()
}

// RegisterSkillSlashCommands exposes enabled skills as Hermes-compatible local
// /skill-name invocations. Built-in handlers win if names collide; callers must
// register skills before prompt templates so templates cannot shadow skills.
func (r *SlashRegistry) RegisterSkillSlashCommands(commands []skills.SkillSlashCommand) {
	for _, command := range commands {
		command := command
		name := strings.TrimPrefix(command.Command, "/")
		if strings.TrimSpace(name) == "" {
			continue
		}
		names := []string{name}
		if underscoreAlias := strings.ReplaceAll(name, "-", "_"); underscoreAlias != name {
			names = append(names, underscoreAlias)
		}
		for _, registerName := range names {
			r.RegisterIfAbsent(registerName, func(input string, model *Model) SlashResult {
				message := skills.BuildSkillSlashCommandMessage(command, slashInvocationArgs(input), skills.SlashMessageOptions{RuntimeNote: "native-tui"})
				return dispatchSkillSlashMessage(model, command, message)
			})
		}
	}
}

// RegisterPromptTemplates exposes operator-authored prompt templates as local
// slash expansions. Built-in handlers and skill invocations win if names collide.
func (r *SlashRegistry) RegisterPromptTemplates(catalog prompttemplates.Catalog) {
	for _, tmpl := range catalog.Templates {
		tmpl := tmpl
		r.RegisterIfAbsent(tmpl.Name, func(input string, _ *Model) SlashResult {
			_, args, ok := prompttemplates.ParseInvocation(input)
			if !ok {
				return SlashResult{}
			}
			expanded := prompttemplates.Expand(tmpl, args)
			return SlashResult{
				Handled:       true,
				StatusMessage: "prompt_template_expanded: " + tmpl.Name,
				EditorText:    expanded,
			}
		}, WithBusyAvailable())
	}
}

func dispatchSkillSlashMessage(model *Model, command skills.SkillSlashCommand, message string) SlashResult {
	message = strings.TrimSpace(message)
	if message == "" {
		return SlashResult{Handled: true, StatusMessage: "skill_invocation_empty: " + command.Name}
	}
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "skill_invoked: " + command.Name}
	}
	if model.turnActive() {
		decision := ResolveHermesKey(HermesKeyEvent{Kind: HermesKeyEnter}, HermesInputState{
			Text:          message,
			Phase:         HermesPhaseRunning,
			BusyInputMode: model.busyInputMode,
		})
		switch decision.Action {
		case HermesActionQueueForNextTurn:
			model.queueFollowUpDraft(decision.SubmitText)
			return SlashResult{Handled: true}
		case HermesActionSteer:
			return SlashResult{Handled: true, Cmd: model.queueSteeringDraft(decision.SubmitText)}
		case HermesActionInterrupt:
			model.queueInterruptDraft(decision.SubmitText)
			return SlashResult{Handled: true, Cmd: model.cancelCmd()}
		}
	}
	if model.offlineSmoke {
		model.applyOfflineSmokeTurn(message)
		return SlashResult{Handled: true, StatusMessage: "skill_invoked_offline: " + command.Name}
	}
	model.inFlight = true
	return SlashResult{Handled: true, StatusMessage: "skill_invoked: " + command.Name, Cmd: model.submitCmd(message)}
}

func slashInvocationArgs(input string) string {
	return uislash.InvocationArgs(input)
}

func slashFallbackResult(input string) SlashResult {
	fallback := uislash.FallbackForInput(input)
	if !fallback.Handled {
		return SlashResult{}
	}
	return SlashResult{Handled: true, StatusMessage: fallback.Status}
}

func slashKnownUnhandledStatus(typed string, policy cli.CommandPolicy) string {
	return uislash.KnownUnhandledStatus(typed, policy)
}

func slashAmbiguousNameStatus(matches []string) string {
	return uislash.AmbiguousNameStatus(matches)
}

// mouseSlashHandler adapts the existing parseMouseTrackingSlash result into
// the SlashResult shape, mutating the model's mouseTracking field and
// emitting the terminal-mode Cmd only when the requested state differs from
// the current one, preserving the dedup behavior asserted by
// TestMouseSlashUpdatesRuntimeWithoutSubmitting.
func mouseSlashHandler(input string, model *Model) SlashResult {
	decision := terminal.HandleMouseSlash(input, model.mouseTracking)
	if !decision.Handled {
		return SlashResult{}
	}
	var cmd tea.Cmd
	if decision.Apply {
		model.mouseTracking = decision.Next
		cmd = model.emitMouseModeCmd(decision.Next)
	}
	return SlashResult{Handled: true, StatusMessage: decision.Status, Cmd: cmd}
}

func copySlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "copy: clipboard unavailable"}
	}
	result := composer.HandleCopySlash(input, model.frame.History, model.clipboardWrite != nil)
	if !result.WriteClipboard {
		return SlashResult{Handled: result.Handled, StatusMessage: result.Status}
	}
	if err := model.clipboardWrite(result.Text); err != nil {
		return SlashResult{Handled: true, StatusMessage: "copy: clipboard failed: " + err.Error()}
	}
	return SlashResult{Handled: true, StatusMessage: result.Status}
}

func skillsSlashHandler(input string, model *Model) SlashResult {
	handler := gateway.HandleSkillsCommand
	if model != nil && model.skillsCommand != nil {
		handler = model.skillsCommand
	}
	return SlashResult{Handled: true, StatusMessage: strings.TrimSpace(handler(input))}
}

func queueSlashHandler(input string, model *Model) SlashResult {
	currentLen := 0
	if model != nil {
		currentLen = model.queuedMessages.Len()
	}
	result := queue.HandleSlash(input, currentLen)
	if !result.Enqueue || model == nil {
		return SlashResult{Handled: true, StatusMessage: result.Status}
	}
	model.queueFollowUpDraft(result.Text)
	return SlashResult{Handled: true}
}

func reloadSkillsSlashHandler(_ string, model *Model) SlashResult {
	var reload skillsslash.ReloadFunc
	if model != nil && model.skillSlashReload != nil {
		reload = func(ctx context.Context) (skillsslash.ReloadResult, error) {
			result, err := model.skillSlashReload(ctx)
			return skillsslash.ReloadResult{Commands: result.Commands, Output: result.Output}, err
		}
	}
	decision := skillsslash.HandleReload(context.Background(), reload)
	if decision.Rebuild && model != nil {
		model.skillSlashCommands = append([]skills.SkillSlashCommand(nil), decision.Commands...)
		model.rebuildSlashRegistry()
	}
	return SlashResult{Handled: decision.Handled, StatusMessage: decision.Status}
}

func quitSlashHandler(_ string, _ *Model) SlashResult {
	return SlashResult{Handled: true, Cmd: tea.Quit}
}

func redrawSlashHandler(_ string, model *Model) SlashResult {
	model.forceLocalRedraw()
	return SlashResult{Handled: true, StatusMessage: "ui redrawn"}
}

// Package tui — Small slash command handlers.
//
// Each handler follows the SlashHandler func(input string, model *Model) SlashResult
// shape, delegates the core logic to a sibling subpackage, and mutates model
// state directly. Consolidating them here reduces root file count.


// ─── /help ──────────────────────────────────────────────────────────────────

func helpSlashHandler(_ string, _ *Model) SlashResult {
	return SlashResult{Handled: true, StatusMessage: nativeTUIHelpStatus()}
}

func nativeTUIHelpStatus() string {
	return help.NativeStatus()
}

// ─── /compact ───────────────────────────────────────────────────────────────

func compactSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "compact: TUI unavailable"}
	}
	result := compact.HandleSlash(input, model.compactTranscript)
	if result.OK {
		model.compactTranscript = result.Next
	}
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
}

func compactSlashNext(input string, current bool) (bool, bool) {
	return compact.Next(input, current)
}

// ─── /details ───────────────────────────────────────────────────────────────

const detailsSlashUsage = details.SlashUsage
const detailsSectionSlashUsage = details.SectionSlashUsage

func detailsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "details: TUI unavailable"}
	}
	next, status := details.ApplySlash(input, model.detailsState)
	model.detailsState = next
	return SlashResult{Handled: true, StatusMessage: status}
}

// ─── /indicator ─────────────────────────────────────────────────────────────

const indicatorUsage = indicator.SlashUsage

func indicatorSlashHandler(input string, model *Model) SlashResult {
	result := indicator.ParseSlash(input, indicator.Style(model.indicatorStyle))
	if result.Apply {
		model.indicatorStyle = IndicatorStyle(result.Style)
	}
	return SlashResult{Handled: true, StatusMessage: result.Status}
}

// ─── /browser ───────────────────────────────────────────────────────────────

const defaultBrowserCDPURL = browser.DefaultCDPURL

func browserSlashHandler(input string, _ *Model) SlashResult {
	return SlashResult{Handled: true, StatusMessage: browser.HandleSlash(input, os.Getenv, os.Setenv)}
}

func browserStatusMessage() string {
	return browser.StatusMessage(browserCDPURLFromEnv())
}

func browserCDPURLFromEnv() string {
	return browser.CDPURLFromEnv(os.Getenv)
}

func validateBrowserCDPURL(endpoint string) error {
	return browser.ValidateCDPURL(endpoint)
}

// ─── /kanban ────────────────────────────────────────────────────────────────

// KanbanSlashFunc runs a full /kanban editor command and returns bounded
// operator-facing output.
type KanbanSlashFunc func(input string) (string, error)

const maxKanbanSlashStatusRunes = kanban.MaxStatusRunes

func kanbanSlashHandler(input string, model *Model) SlashResult {
	var run kanban.Runner
	if model != nil && model.kanbanSlash != nil {
		run = kanban.Runner(model.kanbanSlash)
	}
	result := kanban.HandleSlash(input, run)
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
}

func boundKanbanSlashStatus(status string) string {
	return kanban.BoundStatus(status)
}

// ─── /logs ──────────────────────────────────────────────────────────────────

func logsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "logs: TUI unavailable"}
	}
	res := logs.HandleSlash(input, func(limit int) (string, error) {
		if model.gatewayLogTail == nil {
			return "", nil
		}
		return model.gatewayLogTail(limit)
	})
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}
	page := TransientPageState{Title: res.Title, Body: res.Body}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func logsTailLimit(input string) int {
	return logs.TailLimit(input)
}

// ─── /status ────────────────────────────────────────────────────────────────

func statusSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "status: TUI unavailable"}
	}
	res := statuspage.HandleSlash(model.frame, model.SessionID())
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
	}
	model.transientPage = &res.Page
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func BuildStatusPage(frame kernel.RenderFrame, sessionID string) TransientPageState {
	return statuspage.Build(frame, sessionID)
}

// ─── /title ─────────────────────────────────────────────────────────────────

func titleSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "title: TUI unavailable"}
	}
	res := title.HandleSlash(title.SlashRequest{
		Input:     input,
		SessionID: model.SessionID(),
		TitleFunc: titleSessionFunc(model.sessionTitle),
	})
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func titleSessionFunc(fn SessionTitleFunc) title.SessionTitleFunc {
	if fn == nil {
		return nil
	}
	return func(sessionID, nextTitle string) (title.SessionTitleResult, error) {
		res, err := fn(sessionID, nextTitle)
		return title.SessionTitleResult{Title: res.Title, Pending: res.Pending}, err
	}
}

func titleSlashArg(input string) (string, bool) {
	return title.SlashArg(input)
}

// ─── /tools ─────────────────────────────────────────────────────────────────

func toolsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "tools: TUI unavailable"}
	}
	var configure toolsview.ConfigureFunc
	if model.toolsConfigure != nil {
		configure = func(action string, names []string) (toolsview.Result, error) {
			return model.toolsConfigure(ToolsConfigureRequest{
				Action:    action,
				Names:     names,
				SessionID: model.SessionID(),
			})
		}
	}
	res := toolsview.HandleSlash(input, configure)
	if res.Fallback {
		return slashFallbackResult(input)
	}
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}
	model.transientPage = &TransientPageState{Title: res.Title, Body: res.Body}
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func toolsSlashUsage(action string) string {
	return toolsview.Usage(action)
}

func renderToolsConfigureLines(action string, result ToolsConfigureResult) []string {
	return toolsview.Lines(action, result)
}

// ─── /reset ─────────────────────────────────────────────────────────────────

func sessionResetSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: sessionResetSlashKind(input) + ": TUI unavailable"}
	}
	var resetFn reset.Func
	if model.sessionReset != nil {
		resetFn = func() error { return model.sessionReset() }
	}
	res := reset.HandleSlash(input, model.inFlight || turnIsActive(model.frame.Phase), resetFn)
	if !res.Reset {
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}

	model.frame.History = nil
	model.frame.DraftText = ""
	model.frame.LastError = ""
	model.frame.SessionID = ""
	model.sessionID = ""
	model.inFlight = false
	model.ApprovalState = nil
	model.ClarifyState = nil
	model.SecretState = nil
	model.modelPicker = nil

	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func sessionResetSlashKind(input string) string {
	return reset.Kind(input)
}

// ─── /save ──────────────────────────────────────────────────────────────────

// saveExportTimeout caps how long /save waits on the injected helper.
const saveExportTimeout = save.ExportTimeout

// SessionExportFunc is the injection point for the TUI /save command.
type SessionExportFunc = save.ExportFunc

func saveSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "save: store unavailable"}
	}
	return SlashResult{Handled: true, StatusMessage: save.HandleSlash(
		len(model.frame.History) > 0,
		model.frame.SessionID,
		model.sessionExport,
		os.Remove,
	)}
}

// ─── /skin ──────────────────────────────────────────────────────────────────

func skinSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "skin: TUI unavailable"}
	}
	var configure skin.ConfigFunc
	if model.skinConfig != nil {
		configure = skin.ConfigFunc(model.skinConfig)
	}
	result := skin.HandleSlash(input, model.SessionID(), configure)
	if result.Err != nil || !result.Apply {
		if result.Body != "" {
			model.transientPage = &TransientPageState{Title: "Skin", Body: result.Body}
		} else {
			model.transientPage = nil
		}
		return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
	}
	accepted, err := model.applySkinName(result.AcceptedName)
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "skin: " + err.Error()}
	}
	line := fmt.Sprintf("skin \u2192 %s", accepted)
	model.transientPage = &TransientPageState{Title: "Skin", Body: line}
	return SlashResult{Handled: true, StatusMessage: line}
}

func parseSkinSlashName(input string) string {
	return skin.SlashName(input)
}

func skinDisplayName(name string) string {
	return skin.DisplayName(name)
}

func (m *Model) applySkinName(name string) (string, error) {
	skin, ok := ResolveBuiltinSkin(name)
	if !ok {
		return "", fmt.Errorf("unknown skin %q", strings.TrimSpace(name))
	}
	prompt, _ := skin.PromptSymbols("default")
	m.activeSkinName = skin.Name
	m.activeSkin = skin
	m.editor.Prompt = prompt
	ApplyTextareaSkin(&m.editor, skin)
	return skin.Name, nil
}

func (m Model) currentSkin() HermesSkin {
	if strings.TrimSpace(m.activeSkin.Name) != "" {
		return m.activeSkin
	}
	return DefaultHermesSkin()
}

// ─── /history ───────────────────────────────────────────────────────────────

func historySlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "history: TUI unavailable"}
	}
	res := historypage.HandleSlash(model.frame.History, input)
	if !res.Open {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
	}
	model.transientPage = &res.Page
	return SlashResult{Handled: true, StatusMessage: res.StatusMessage}
}

func historyPreviewLimit(input string) int {
	return historypage.PreviewLimit(input)
}

func BuildHistoryPage(history []llm.Message, preview int) (TransientPageState, bool) {
	return historypage.Build(history, preview)
}

func historyMessageText(msg llm.Message) string {
	return historypage.MessageText(msg)
}

// ─── /voice ─────────────────────────────────────────────────────────────────

func voiceSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "voice: TUI unavailable"}
	}
	var toggle voice.ToggleFunc
	if model.voiceToggle != nil {
		toggle = voice.ToggleFunc(model.voiceToggle)
	}
	result := voice.HandleSlash(input, model.SessionID(), toggle)
	if result.Err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
	}
	if result.UpdateRecordKey {
		binding := tools.ResolveVoiceRecordKey(result.RecordKey, tools.VoiceRecordKeyOptions{})
		model.voiceRecordKey = binding.Raw
	}
	body := strings.Join(result.Lines, "\n")
	model.transientPage = &TransientPageState{Title: "Voice", Body: body}
	return SlashResult{Handled: true, StatusMessage: result.StatusMessage}
}

func parseVoiceSlashAction(input string) string {
	return voice.Action(input)
}

func renderVoiceToggleLines(action string, result VoiceToggleResult) []string {
	return voice.Lines(action, result)
}

func onOff(value bool) string {
	return voice.OnOff(value)
}

// ─── /branch (consolidated from slash_branch.go) ────────────────────────────

const branchForkTimeout = branch.ForkTimeout

// BranchRequest is the input to SessionBranchFunc.
type BranchRequest = branch.Request

// BranchResult is the helper's response.
type BranchResult = branch.Result

// SessionBranchFunc is the injection point for the TUI /branch command.
type SessionBranchFunc = branch.Func

func branchSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "branch: store unavailable"}
	}
	var fork branch.Func
	if model.sessionBranch != nil {
		fork = func(ctx context.Context, req branch.Request) (branch.Result, error) {
			return model.sessionBranch(ctx, req)
		}
	}
	res := branch.HandleSlash(input, len(model.frame.History) > 0, model.SessionID(), len(model.frame.History), cloneResumeHistory(model.frame.History), fork)
	if !res.Switch {
		return SlashResult{Handled: true, StatusMessage: res.Status}
	}

	model.sessionID = res.Branch.SessionID
	model.frame.SessionID = res.Branch.SessionID
	model.inFlight = false
	model.frame.DraftText = ""
	return SlashResult{Handled: true, StatusMessage: res.Status}
}

func branchTitleFromInput(input string) string {
	return branch.TitleFromInput(input)
}

func branchSuccessStatus(res BranchResult) string {
	return branch.SuccessStatus(res)
}

// ─── /sessions (consolidated from slash_sessions.go) ────────────────────────

const sessionResumeTimeout = 5 * time.Second

func sessionsSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "sessions: TUI unavailable"}
	}
	if sessionSlashName(input) == "resume" {
		if arg := sessionSlashArg(input); arg != "" {
			return resumeSlashWithArg(arg, model)
		}
	}
	if model.sessionDirectory == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "sessions: directory unavailable"}
	}
	entries, err := model.sessionDirectory(sessionsSlashLimit(input))
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "sessions: " + err.Error()}
	}
	page, ok := BuildSessionsPage(entries)
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "no sessions found"}
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "sessions opened"}
}

func resumeSlashWithArg(query string, model *Model) SlashResult {
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "interrupt the current turn before trying to switch sessions"}
	}
	if model.sessionResume == nil {
		return SlashResult{Handled: true, StatusMessage: "resume: session switch unavailable"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), sessionResumeTimeout)
	defer cancel()
	result, err := model.sessionResume(ctx, query)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "resume: " + err.Error()}
	}
	sessionID := strings.TrimSpace(result.SessionID)
	if sessionID == "" {
		return SlashResult{Handled: true, StatusMessage: "resume: no session selected"}
	}
	model.sessionID = sessionID
	model.frame.SessionID = sessionID
	model.frame.History = cloneResumeHistory(result.History)
	model.frame.DraftText = ""
	model.frame.LastError = ""
	model.inFlight = false
	model.transientPage = nil
	return SlashResult{Handled: true, StatusMessage: resumeSuccessStatus(sessionID, len(model.frame.History))}
}

func sessionSlashName(input string) string {
	return sessionspage.SlashName(input)
}

func sessionSlashArg(input string) string {
	return sessionspage.SlashArg(input)
}

func resumeSuccessStatus(sessionID string, messages int) string {
	return sessionspage.ResumeSuccessStatus(sessionID, messages)
}

func cloneResumeHistory(in []llm.Message) []llm.Message {
	return sessionspage.CloneResumeHistory(in)
}

func sessionsSlashLimit(input string) int {
	return sessionspage.Limit(input)
}

func BuildSessionsPage(entries []SessionDirectoryEntry) (TransientPageState, bool) {
	return sessionspage.Build(entries)
}

func messageCountLabel(count int) string {
	return sessionspage.MessageCountLabel(count)
}

func sessionDirectoryTimeLabel(ts int64) string {
	return sessionspage.TimeLabel(ts)
}

// ─── /tree (consolidated from slash_tree.go) ────────────────────────────────

const sessionTreeTimeout = 5 * time.Second

// SessionTreeFunc is the injection point for the TUI /tree command.
type SessionTreeFunc = sessiontree.QueryFunc

// SessionTreeLabelRequest is the input for /tree label/unlabel.
type SessionTreeLabelRequest = sessiontree.LabelRequest

// SessionTreeLabelResult is the response for /tree label/unlabel.
type SessionTreeLabelResult = sessiontree.LabelResult

// SessionTreeLabelFunc is the injection point for /tree label/unlabel.
type SessionTreeLabelFunc = sessiontree.LabelFunc

// SessionTreeRestoreRequest is the input for /tree restore.
type SessionTreeRestoreRequest = sessiontree.RestoreRequest

// SessionTreeRestoreResult is the response for /tree restore.
type SessionTreeRestoreResult = sessiontree.RestoreResult

// SessionTreeRestoreFunc is the injection point for /tree restore.
type SessionTreeRestoreFunc = sessiontree.RestoreFunc

func treeSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: TUI unavailable"}
	}
	args := slashArgs(input)
	if len(args) == 0 || treeSlashIsFilter(args[0]) {
		return openTreeSlash(args, model)
	}
	switch strings.ToLower(args[0]) {
	case "label":
		return treeLabelSlash(args[1:], model, "set")
	case "unlabel", "clear-label", "clear":
		return treeLabelSlash(args[1:], model, "clear")
	case "restore", "edit":
		return treeRestoreSlash(args[1:], model)
	default:
		return SlashResult{Handled: true, StatusMessage: "tree: usage /tree [--filter MODE] | /tree label <session> <label> | /tree unlabel <session> [label] | /tree restore <session> <turn_id>"}
	}
}

func openTreeSlash(args []string, model *Model) SlashResult {
	if model.sessionTree == nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: session tree unavailable"}
	}
	filter := parseTreeSlashFilter(args)
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTree(ctx, SessionTreeRequest{Filter: filter, ActiveSessionID: model.SessionID()})
	if err != nil {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: " + err.Error()}
	}
	if result.Filter == "" {
		result.Filter = filter
	}
	if result.ActiveSessionID == "" {
		result.ActiveSessionID = model.SessionID()
	}
	page, ok := BuildSessionTreePage(result)
	if !ok {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: "tree: no sessions found"}
	}
	model.transientPage = &page
	return SlashResult{Handled: true, StatusMessage: "tree opened"}
}

func treeLabelSlash(args []string, model *Model, action string) SlashResult {
	if model.sessionTreeLabel == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: labels unavailable"}
	}
	parsed, status, ok := sessiontree.ParseLabelRequest(args, action)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	req := SessionTreeLabelRequest{SessionID: parsed.SessionID, Action: parsed.Action, Label: parsed.Label}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeLabel(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: labels: " + err.Error()}
	}
	return SlashResult{Handled: true, StatusMessage: sessiontree.FormatLabelStatus(result.SessionID, req.SessionID, result.Labels)}
}

func treeRestoreSlash(args []string, model *Model) SlashResult {
	if model.inFlight || turnIsActive(model.frame.Phase) {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable while turn is active"}
	}
	if model.sessionTreeRestore == nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore unavailable"}
	}
	parsed, status, ok := sessiontree.ParseRestoreRequest(args)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	req := SessionTreeRestoreRequest{SessionID: parsed.SessionID, MessageID: parsed.MessageID}
	ctx, cancel := context.WithTimeout(context.Background(), sessionTreeTimeout)
	defer cancel()
	result, err := model.sessionTreeRestore(ctx, req)
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "tree: restore: " + err.Error()}
	}
	status, editable := sessiontree.FormatRestoreStatus(req, result)
	if !editable {
		return SlashResult{Handled: true, StatusMessage: status}
	}
	return SlashResult{Handled: true, StatusMessage: status, EditorText: result.Text}
}

func slashArgs(input string) []string {
	return sessiontree.SlashArgs(input)
}

func treeSlashIsFilter(arg string) bool {
	return sessiontree.SlashIsFilter(arg)
}

func parseTreeSlashFilter(args []string) string {
	return sessiontree.ParseSlashFilter(args)
}

// ─── /usage (consolidated from slash_usage.go) ──────────────────────────────

func usageSlashHandler(_ string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "usage: TUI unavailable"}
	}
	result := usagepage.HandleSlash(model.frame, model.SessionID(), model.accountUsage != nil)
	if !result.OpenPage {
		model.transientPage = nil
		return SlashResult{Handled: true, StatusMessage: result.Status}
	}
	model.transientPage = &result.Page
	var cmd tea.Cmd
	if result.FetchAccount {
		cmd = model.usageAccountCmd()
	}
	return SlashResult{Handled: true, StatusMessage: result.Status, Cmd: cmd}
}

func BuildUsagePage(frame kernel.RenderFrame, sessionID string) (TransientPageState, bool) {
	return usagepage.Build(frame, sessionID)
}

func appendUsageAccountLines(body string, lines []string) string {
	return usagepage.AppendAccountLines(body, lines)
}

func replaceUsageAccountLoading(body string, lines []string) string {
	return usagepage.ReplaceAccountLoading(body, lines)
}

// ─── /model (consolidated from slash_model.go) ──────────────────────────────

// ModelPickerCatalogProvider is the TUI-local provider/model catalog shape
// consumed by the /model overlay.
type ModelPickerCatalogProvider = modelpicker.CatalogProvider

// ModelPickerCatalogFunc returns a fresh provider/model catalog for the TUI
// /model picker.
type ModelPickerCatalogFunc func() ([]ModelPickerCatalogProvider, error)

// DefaultModelPickerCatalog adapts the shared Hermes picker provider list and
// curated model suggestions into the pure renderer entries used by
// ModelPickerState.
func DefaultModelPickerCatalog() ([]ModelPickerCatalogProvider, error) {
	return modelpicker.DefaultCatalog()
}

func modelSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "model: TUI unavailable"}
	}
	if model.inFlight {
		return SlashResult{Handled: true, StatusMessage: "model: cannot switch models while a turn is running"}
	}
	if arg := modelSlashArgument(input); arg != "" {
		return model.applyModelSelection(model.currentModelProvider(), arg)
	}
	catalog, err := model.loadModelPickerCatalog()
	if err != nil {
		return SlashResult{Handled: true, StatusMessage: "model: catalog unavailable: " + err.Error()}
	}
	if len(catalog) == 0 {
		return SlashResult{Handled: true, StatusMessage: "model: catalog unavailable"}
	}
	state := newModelPickerState(catalog, model.currentModelProvider(), model.currentModelName(), model.width, model.height)
	model.modelPicker = &state
	model.modelPickerChoices = catalog
	return SlashResult{Handled: true, StatusMessage: "model: select provider/model"}
}

func modelSlashArgument(input string) string {
	return modelpicker.SlashArgument(input)
}

func normalizeModelPickerCatalog(catalog []ModelPickerCatalogProvider) []ModelPickerCatalogProvider {
	return modelpicker.NormalizeCatalog(catalog)
}

func newModelPickerState(catalog []ModelPickerCatalogProvider, currentProvider, currentModel string, width, height int) ModelPickerState {
	return modelpicker.NewState(catalog, currentProvider, currentModel, width, height)
}

func modelsForProviderIndex(catalog []ModelPickerCatalogProvider, idx int) []ModelEntry {
	return modelpicker.ModelsForProviderIndex(catalog, idx)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// ─── Status bar slash handlers ──────────────────────────────────────────────

const statusBarSlashUsage = statusbar.SlashUsage

func statusbarSlashHandler(input string, model *Model) SlashResult {
	if model == nil {
		return SlashResult{Handled: true, StatusMessage: "statusbar: TUI unavailable"}
	}
	next, ok := statusBarSlashNext(input, model.statusBarMode)
	if !ok {
		return SlashResult{Handled: true, StatusMessage: statusBarSlashUsage}
	}
	model.statusBarMode = next
	return SlashResult{Handled: true, StatusMessage: "status bar " + string(next)}
}

func statusBarSlashNext(input string, current StatusBarMode) (StatusBarMode, bool) {
	return statusbar.SlashNext(input, current)
}



// SlashCompletion is one entry in a slash-completion menu. It carries enough
// metadata for a downstream Bubble Tea menu binding to render canonical
// commands, their aliases, recognized-but-unavailable commands, and static
// subcommands without re-querying the registry. Helpers in this file are pure:
// no goroutines, no terminal IO, no provider/config/plugin filesystem
// dependency, and no shared mutable state across calls.
type SlashCompletion = slashcompletion.Completion

type slashCompletionState struct {
	key          string
	index        int
	dismissedFor string
}

type slashCompletionMenu struct {
	request     TUICompletionRequest
	completions []SlashCompletion
	total       int
	key         string
}

type slashCompletionAcceptTrigger int

const (
	slashCompletionAcceptEnter slashCompletionAcceptTrigger = iota
	slashCompletionAcceptTab
)

// HermesSlashCommandCompletions returns the slash-command completions a Hermes
// prompt_toolkit completer would surface for the given editor buffer text. The
// helper is pure, deterministic, and case-insensitive on the typed prefix.
//
// Behavior, mirroring hermes_cli/commands.py:SlashCommandCompleter:
//   - input that does not start with "/" returns nil (no completions).
//   - the leading "/" is stripped and the remaining text is matched as a
//     case-insensitive prefix against every canonical command name and alias
//     in cli.CommandRegistry.
//   - exact matches are still returned so the menu can stay open while the
//     user keeps editing.
//   - results are sorted alphabetically and de-duplicated, so two callers see
//     the same list every time and a freshly returned slice does not share a
//     backing array with any cached state.
//   - recognized-but-unavailable commands appear in the list with
//     Available=false; only outright unknown prefixes return nil.
func HermesSlashCommandCompletions(input string) []SlashCompletion {
	return slashcompletion.CommandCompletions(input)
}

// PromptTemplateSlashCompletions returns prompt-template command completions.
func PromptTemplateSlashCompletions(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.PromptTemplateCompletions(input, catalog)
}

// SkillSlashCompletions returns enabled dynamic skill slash completions.
func SkillSlashCompletions(input string, commands []skills.SkillSlashCommand) []SlashCompletion {
	return slashcompletion.SkillCompletions(input, commands)
}

// SlashCompletionsWithPromptTemplates merges built-in Hermes/Gormes slash
// completions with non-shadowing prompt-template completions.
func SlashCompletionsWithPromptTemplates(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.WithPromptTemplates(input, catalog)
}

// SlashCompletionsWithDynamic merges built-in Hermes/Gormes slash completions,
// dynamic skill invocations, and prompt-template completions in precedence
// order. Later sources cannot shadow earlier ones.
func SlashCompletionsWithDynamic(input string, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.WithDynamic(input, commands, catalog)
}

// HermesSlashSubcommandCompletions returns static subcommand completions for
// inputs of the form "/cmd <prefix>" where the resolved command declares a
// non-empty Subcommands inventory in cli.CommandRegistry. Dynamic per-runtime
// menus (/model, /skin, /personality) are intentionally not surfaced here —
// they remain dependent rows that bind live config sources.
func HermesSlashSubcommandCompletions(input string) []SlashCompletion {
	return slashcompletion.SubcommandCompletions(input)
}

func (m Model) renderActiveSlashCompletionMenu(input string) string {
	if m.slashCompletion.dismissedFor == input {
		return ""
	}
	menu, ok := slashCompletionMenuForInput(input, m.width, m.skillSlashCommands, m.promptTemplates)
	if !ok {
		return ""
	}
	selected := 0
	if m.slashCompletion.key == menu.key {
		selected = clampSlashCompletionIndex(m.slashCompletion.index, len(menu.completions))
	}
	return renderSlashCompletionMenuWithDynamicSelected(input, m.width, m.currentSkin(), m.skillSlashCommands, m.promptTemplates, selected)
}

func (m *Model) activeSlashCompletionMenu() (slashCompletionMenu, bool) {
	input := m.editor.Value()
	if m.slashCompletion.dismissedFor == input {
		return slashCompletionMenu{}, false
	}
	return slashCompletionMenuForInput(input, m.width, m.skillSlashCommands, m.promptTemplates)
}

func (m *Model) ensureSlashCompletionSelection(menu slashCompletionMenu) int {
	if m.slashCompletion.key != menu.key {
		m.slashCompletion.key = menu.key
		m.slashCompletion.index = 0
	}
	m.slashCompletion.index = clampSlashCompletionIndex(m.slashCompletion.index, len(menu.completions))
	return m.slashCompletion.index
}

func (m *Model) resetSlashCompletionDismissalForInput(input string) {
	if m.slashCompletion.dismissedFor != "" && m.slashCompletion.dismissedFor != input {
		m.slashCompletion.dismissedFor = ""
	}
}

func renderSlashCompletionMenu(input string, width int) string {
	return renderSlashCompletionMenuWithSkin(input, width, DefaultHermesSkin())
}

func renderSlashCompletionMenuWithSkin(input string, width int, skin HermesSkin) string {
	return renderSlashCompletionMenuWithTemplates(input, width, skin, prompttemplates.Catalog{})
}

func renderSlashCompletionMenuWithTemplates(input string, width int, skin HermesSkin, catalog prompttemplates.Catalog) string {
	return renderSlashCompletionMenuWithDynamic(input, width, skin, nil, catalog)
}

func renderSlashCompletionMenuWithDynamic(input string, width int, skin HermesSkin, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) string {
	return renderSlashCompletionMenuWithDynamicSelected(input, width, skin, commands, catalog, 0)
}

func renderSlashCompletionMenuWithDynamicSelected(input string, width int, skin HermesSkin, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog, selected int) string {
	menu, ok := slashCompletionMenuForInput(input, width, commands, catalog)
	if !ok {
		return ""
	}
	selected = clampSlashCompletionIndex(selected, len(menu.completions))
	styles := SkinStylesFor(skin)
	bodyWidth := width - 4
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	nameW := 0
	for _, c := range menu.completions {
		display := slashCompletionDisplay(c)
		if w := len([]rune(display)); w > nameW {
			nameW = w
		}
	}
	if nameW > 24 {
		nameW = 24
	}
	if nameW < 8 {
		nameW = 8
	}
	descW := bodyWidth - nameW - 5
	if descW < 8 {
		descW = 8
	}
	lines := make([]string, 0, len(menu.completions)+3)
	query := strings.TrimSpace(menu.request.Text)
	if query == "" {
		query = input
	}
	lines = append(lines, styles.Accent.Render(truncateEllipsis("╭─ Search "+query, bodyWidth)))
	for idx, c := range menu.completions {
		marker := "  "
		rowStyle := styles.Normal
		if idx == selected {
			marker = "❯ "
			rowStyle = styles.Selected
		}
		if !c.Available {
			rowStyle = styles.Dim
		}
		displayText := padRightRunes(truncateEllipsis(slashCompletionDisplay(c), nameW), nameW)
		desc := strings.TrimSpace(c.Description)
		if !c.Available {
			if desc != "" {
				desc = "⚡ " + desc
			} else {
				desc = "⚡ recognized, unavailable"
			}
		}
		line := "│ " + marker + rowStyle.Render(displayText)
		if desc != "" {
			line += "  " + styles.Dim.Render(truncateEllipsis(desc, descW))
		}
		lines = append(lines, line)
	}
	if extra := menu.total - len(menu.completions); extra > 0 {
		lines = append(lines, styles.Dim.Render(fmt.Sprintf("│ … +%d more matches", extra)))
	}
	footer := "╰─ ↑/↓ select · Enter complete · Esc close"
	lines = append(lines, styles.Dim.Render(truncateEllipsis(footer, bodyWidth)))
	return strings.Join(lines, "\n")
}

func slashCompletionMenuForInput(input string, width int, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) (slashCompletionMenu, bool) {
	req, ok := CompletionRequestForInput(input)
	if !ok || req.Method != TUICompletionSlash {
		return slashCompletionMenu{}, false
	}
	completions := HermesSlashSubcommandCompletions(input)
	if len(completions) == 0 {
		completions = SlashCompletionsWithDynamic(input, commands, catalog)
	}
	if len(completions) == 0 {
		return slashCompletionMenu{}, false
	}
	limit := len(completions)
	if visible := slashCompletionVisibleLimit(width); limit > visible {
		limit = visible
	}
	visible := append([]SlashCompletion(nil), completions[:limit]...)
	return slashCompletionMenu{
		request:     req,
		completions: visible,
		total:       len(completions),
		key:         slashCompletionCandidateKey(req, visible),
	}, true
}

func slashCompletionCandidateKey(req TUICompletionRequest, completions []SlashCompletion) string {
	var b strings.Builder
	b.WriteString(string(req.Method))
	b.WriteByte('|')
	for _, c := range completions {
		b.WriteString(c.Name)
		b.WriteByte('\x00')
		b.WriteString(c.Display)
		b.WriteByte('\x00')
		b.WriteString(c.ArgumentHint)
		b.WriteByte('\x00')
		if c.Available {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		b.WriteByte('|')
	}
	return b.String()
}

func slashCompletionVisibleLimit(width int) int {
	switch {
	case width < 40:
		return 3
	case width < 64:
		return 5
	default:
		return 8
	}
}

func clampSlashCompletionIndex(index, count int) int {
	if count <= 0 || index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

func wrapSlashCompletionIndex(index, delta, count int) int {
	if count <= 0 {
		return 0
	}
	return (index + delta + count) % count
}

func slashCompletionAcceptedText(input string, completion SlashCompletion, trigger slashCompletionAcceptTrigger) (string, bool) {
	return slashcompletion.AcceptedText(input, completion, trigger == slashCompletionAcceptTab)
}

func slashCompletionDisplay(c SlashCompletion) string {
	display := strings.TrimSpace(c.Display)
	if display == "" {
		if strings.HasPrefix(c.Name, "/") {
			display = c.Name
		} else {
			display = c.Name
		}
	}
	if hint := strings.TrimSpace(c.ArgumentHint); hint != "" {
		return display + " " + hint
	}
	return display
}

func padRightRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	for len([]rune(value)) < width {
		value += " "
	}
	return value
}

// HermesSlashAutoSuggest returns the inline ghost-text suffix Hermes'
// SlashCommandAutoSuggest would render for the given editor buffer text. The
// returned string is empty whenever no unique unambiguous completion exists:
// non-slash input, multiple matches, an already-complete name, or an unknown
// prefix.
//
// Behavior, mirroring hermes_cli/commands.py:SlashCommandAutoSuggest:
//   - for "/<word>" with no trailing space, returns the suffix of the unique
//     matching canonical command name (aliases participate in disambiguation
//     but the suggestion text is the alias's own tail when it is the unique
//     match — Hermes iterates COMMANDS keys in declaration order; we adopt
//     deterministic alphabetical order so two callers see the same answer).
//   - for "/<cmd> <word>" with the command resolved, returns the suffix of
//     the unique matching subcommand (Hermes order preserved by registry).
//   - returns "" for ambiguous, exact, or unrecognized input.
func HermesSlashAutoSuggest(input string) string {
	return slashcompletion.AutoSuggest(input)
}
