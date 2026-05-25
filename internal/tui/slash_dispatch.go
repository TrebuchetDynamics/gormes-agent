package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
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
	return strings.ToLower(strings.TrimPrefix(name, "/"))
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

func slashFallbackResult(input string) SlashResult {
	resolved := cli.ResolveCommandAlias(input)
	if resolved.RawCommand == "" {
		return SlashResult{}
	}
	switch resolved.Kind {
	case cli.CommandAliasExact, cli.CommandAliasAlias, cli.CommandAliasPrefix:
		return SlashResult{Handled: true, StatusMessage: slashKnownUnhandledStatus(resolved.RawCommand, resolved.Policy)}
	case cli.CommandAliasAmbiguous:
		return SlashResult{Handled: true, StatusMessage: slashAmbiguousNameStatus(resolved.Matches)}
	case cli.CommandAliasUnknown:
		return SlashResult{
			Handled:       true,
			StatusMessage: fmt.Sprintf("unknown command /%s — no slash command by that name is available", resolved.RawCommand),
		}
	}
	return SlashResult{}
}

func slashKnownUnhandledStatus(typed string, policy cli.CommandPolicy) string {
	display := "/" + policy.Name
	if typed != policy.Name {
		display = fmt.Sprintf("/%s -> /%s", typed, policy.Name)
	}
	switch policy.Surface {
	case cli.CommandSurfaceGateway:
		return display + " is recognized but requires gateway support in the native TUI"
	default:
		return display + " is recognized but unavailable in the native TUI"
	}
}

func slashAmbiguousStatus(matches []SlashCompletion) string {
	limit := len(matches)
	if limit > 6 {
		limit = 6
	}
	names := make([]string, 0, limit)
	for _, match := range matches[:limit] {
		names = append(names, "/"+match.Name)
	}
	suffix := ""
	if len(matches) > limit {
		suffix = ", ..."
	}
	return "ambiguous command: " + strings.Join(names, ", ") + suffix
}

func slashAmbiguousNameStatus(matches []string) string {
	limit := len(matches)
	if limit > 6 {
		limit = 6
	}
	names := append([]string(nil), matches[:limit]...)
	suffix := ""
	if len(matches) > limit {
		suffix = ", ..."
	}
	return "ambiguous command: " + strings.Join(names, ", ") + suffix
}

// mouseSlashHandler adapts the existing parseMouseTrackingSlash result into
// the SlashResult shape, mutating the model's mouseTracking field and
// emitting the terminal-mode Cmd only when the requested state differs from
// the current one, preserving the dedup behavior asserted by
// TestMouseSlashUpdatesRuntimeWithoutSubmitting.
func mouseSlashHandler(input string, model *Model) SlashResult {
	parsed := parseMouseTrackingSlash(input, model.mouseTracking)
	if !parsed.handled {
		return SlashResult{}
	}
	if !parsed.valid {
		return SlashResult{Handled: true, StatusMessage: parsed.message}
	}

	statusMessage := "mouse tracking on"
	if !parsed.next {
		statusMessage = "mouse tracking off"
	}

	var cmd tea.Cmd
	if parsed.next != model.mouseTracking {
		model.mouseTracking = parsed.next
		cmd = model.emitMouseModeCmd(parsed.next)
	}
	return SlashResult{Handled: true, StatusMessage: statusMessage, Cmd: cmd}
}

func copySlashHandler(input string, model *Model) SlashResult {
	if model.clipboardWrite == nil {
		return SlashResult{Handled: true, StatusMessage: "copy: clipboard unavailable"}
	}
	fields := strings.Fields(input)
	arg := ""
	if len(fields) > 1 {
		arg = fields[1]
	}
	result := SelectComposerCopyText(model.frame.History, arg)
	if !result.OK {
		return SlashResult{Handled: true, StatusMessage: copyStatusForEvidence(result)}
	}
	if err := model.clipboardWrite(result.Text); err != nil {
		return SlashResult{Handled: true, StatusMessage: "copy: clipboard failed: " + err.Error()}
	}
	return SlashResult{
		Handled:       true,
		StatusMessage: fmt.Sprintf("Copied assistant response #%d to clipboard", result.ResponseNumber),
	}
}

func copyStatusForEvidence(result ComposerCopyResult) string {
	switch result.Evidence {
	case "tui_ingress_copy_invalid_index":
		return "copy: invalid response number"
	case "tui_ingress_copy_empty_response":
		return fmt.Sprintf("copy: assistant response #%d has no visible text", result.ResponseNumber)
	default:
		return "copy: nothing to copy"
	}
}

func skillsSlashHandler(input string, _ *Model) SlashResult {
	return SlashResult{Handled: true, StatusMessage: strings.TrimSpace(gateway.HandleSkillsCommand(input))}
}

func quitSlashHandler(_ string, _ *Model) SlashResult {
	return SlashResult{Handled: true, Cmd: tea.Quit}
}

func redrawSlashHandler(_ string, model *Model) SlashResult {
	model.forceLocalRedraw()
	return SlashResult{Handled: true, StatusMessage: "ui redrawn"}
}
