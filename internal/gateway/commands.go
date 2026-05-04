package gateway

import (
	"fmt"
	"sort"
	"strings"
)

// CommandDef is the canonical slash-command definition shared by gateway
// parsing, help text, and per-platform command exposure helpers.
type CommandDef struct {
	Name             string
	Description      string
	Kind             EventKind
	Aliases          []string
	ActiveTurnPolicy CommandActiveTurnPolicy
}

// CommandActiveTurnPolicy documents how a command behaves while a gateway turn
// is already active.
type CommandActiveTurnPolicy string

const (
	CommandActiveTurnPolicyImmediate   CommandActiveTurnPolicy = "immediate"
	CommandActiveTurnPolicyReject      CommandActiveTurnPolicy = "reject"
	CommandActiveTurnPolicyDrain       CommandActiveTurnPolicy = "drain"
	CommandActiveTurnPolicyUnavailable CommandActiveTurnPolicy = "unavailable"
)

// PlatformCommand is the platform-facing command/menu shape used for channel
// exposure helpers such as Telegram bot menus.
type PlatformCommand struct {
	Name        string
	Description string
}

// CommandRegistry is the single source of truth for gateway slash commands.
var CommandRegistry = []CommandDef{
	{
		Name:             "help",
		Description:      "Show available commands",
		Kind:             EventStart,
		Aliases:          []string{"start"},
		ActiveTurnPolicy: CommandActiveTurnPolicyImmediate,
	},
	{
		Name:             "new",
		Description:      "Start a fresh session",
		Kind:             EventReset,
		Aliases:          []string{"reset"},
		ActiveTurnPolicy: CommandActiveTurnPolicyReject,
	},
	{
		Name:             "stop",
		Description:      "Cancel the active turn",
		Kind:             EventCancel,
		ActiveTurnPolicy: CommandActiveTurnPolicyImmediate,
	},
	{
		Name:             "restart",
		Description:      "Gracefully restart the gateway",
		Kind:             EventRestart,
		ActiveTurnPolicy: CommandActiveTurnPolicyDrain,
	},
	{
		Name:             "steer",
		Description:      "Queue steering guidance for the active turn",
		Kind:             EventSteer,
		ActiveTurnPolicy: CommandActiveTurnPolicyDrain,
	},
	{
		Name:             "usage",
		Description:      "Show runtime and provider account usage",
		Kind:             EventUsage,
		ActiveTurnPolicy: CommandActiveTurnPolicyImmediate,
	},
	{Name: "reasoning", Description: "Manage reasoning effort and display", Kind: EventReasoning, ActiveTurnPolicy: CommandActiveTurnPolicyDrain},
	{Name: "browser", Description: "Connect browser tools to your live Chrome via CDP", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "busy", Description: "Control what Enter does while Gormes is working", Kind: EventBusy, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "tts", Description: "Configure text-to-speech", Kind: EventTTS, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "clear", Description: "Clear screen and start a new session", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "commands", Description: "Browse all commands and skills", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "config", Description: "Show current configuration", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "copy", Description: "Copy the last assistant response to clipboard", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "cron", Description: "Manage scheduled tasks", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "debug", Description: "Upload debug report and get shareable links", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "details", Description: "Toggle detailed activity sections", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "fast", Description: "Toggle fast mode", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "goal", Description: "Set a standing goal Gormes works on across turns until achieved", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "history", Description: "Show conversation history", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "image", Description: "Attach a local image file for your next prompt", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "insights", Description: "Show usage insights and analytics", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "model", Description: "Show current model and provider", Kind: EventModel, Aliases: []string{"provider"}, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "paste", Description: "Attach clipboard image", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "personality", Description: "Set a predefined personality", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "platforms", Description: "Show gateway/messaging platform status", Kind: EventGateway, Aliases: []string{"gateway"}, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "plugins", Description: "List installed plugins", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "profile", Description: "Show active profile name and home directory", Kind: EventProfile, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "quit", Description: "Exit the CLI", Kind: EventUnknown, Aliases: []string{"exit"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "reload", Description: "Reload .env variables into the running session", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "reload-mcp", Description: "Reload MCP servers from config", Kind: EventUnknown, Aliases: []string{"reload_mcp"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "resume", Description: "Resume a previously-named session", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "save", Description: "Save the current conversation", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "sessions", Description: "List or show sessions", Kind: EventSessions, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate, Aliases: []string{"session"}},
	{Name: "sethome", Description: "Set this chat as the home channel", Kind: EventUnknown, Aliases: []string{"set-home"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "skills", Description: "List or inspect installed skills", Kind: EventSkills, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "skin", Description: "Show or change the display skin/theme", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "statusbar", Description: "Toggle the context/model status bar", Kind: EventUnknown, Aliases: []string{"sb"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "tools", Description: "Manage tools", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "toolsets", Description: "List available toolsets", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "update", Description: "Update Gormes to the latest version", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "verbose", Description: "Cycle tool progress display", Kind: EventVerbose, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "voice", Description: "Toggle voice mode", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "yolo", Description: "Toggle YOLO mode", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "retry", Description: "Retry the last message (resend to agent)", Kind: EventRetry, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "undo", Description: "Remove the last user/assistant exchange", Kind: EventUndo, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "title", Description: "Set a title for the current session", Kind: EventTitle, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "branch", Description: "Branch the current session (explore a different path)", Kind: EventUnknown, Aliases: []string{"fork"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "compress", Description: "Manually compress conversation context", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "rollback", Description: "List or restore filesystem checkpoints", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "snapshot", Description: "Create or restore state snapshots of Gormes config/state", Kind: EventUnknown, Aliases: []string{"snap"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "approve", Description: "Approve a pending dangerous command", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "deny", Description: "Deny a pending dangerous command", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "background", Description: "Run a prompt in the background", Kind: EventUnknown, Aliases: []string{"bg"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "btw", Description: "Ephemeral side question using session context (no tools, not persisted)", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "agents", Description: "Show active agents and running tasks", Kind: EventUnknown, Aliases: []string{"tasks"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "queue", Description: "Queue a prompt for the next turn (doesn't interrupt)", Kind: EventUnknown, Aliases: []string{"q"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "status", Description: "Show session info", Kind: EventStatus, ActiveTurnPolicy: CommandActiveTurnPolicyImmediate},
	{Name: "footer", Description: "Toggle the gateway footer", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "gquota", Description: "Show Google Gemini Code Assist quota usage", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "indicator", Description: "Pick TUI busy-indicator style", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "curator", Description: "Background skill maintenance", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "redraw", Description: "Redraw the terminal screen", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
}

var telegramMenuUnavailableCommands = map[string]struct{}{
	"agents":     {},
	"approve":    {},
	"background": {},
	"branch":     {},
	"btw":        {},
	"compress":   {},
	"deny":       {},
	"queue":      {},
	"rollback":   {},
	"snapshot":   {},
	"undo":       {},
}

var commandLookup = buildCommandLookup()

var recognizedUnavailableSlashCommands = buildRecognizedUnavailableSlashCommands()

func buildCommandLookup() map[string]CommandDef {
	lookup := make(map[string]CommandDef, len(CommandRegistry)*2)
	for _, cmd := range CommandRegistry {
		lookup[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			lookup[alias] = cmd
		}
	}
	return lookup
}

func buildRecognizedUnavailableSlashCommands() map[string]struct{} {
	lookup := make(map[string]struct{}, len(CommandRegistry))
	for _, cmd := range CommandRegistry {
		if cmd.ActiveTurnPolicy != CommandActiveTurnPolicyUnavailable {
			continue
		}
		lookup[slashCommandName(cmd.Name)] = struct{}{}
		for _, alias := range cmd.Aliases {
			lookup[slashCommandName(alias)] = struct{}{}
		}
	}
	return lookup
}

// ResolveCommand maps a slash command or alias to its canonical definition.
func ResolveCommand(name string) (CommandDef, bool) {
	key := slashCommandName(name)
	cmd, ok := commandLookup[key]
	return cmd, ok
}

func slashCommandName(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexAny(key, " \t\r\n"); i >= 0 {
		key = key[:i]
	}
	return key
}

func isRecognizedUnavailableSlashCommand(name string) bool {
	_, ok := recognizedUnavailableSlashCommands[slashCommandName(name)]
	return ok
}

// GatewayCommandDispatch is the gateway-visible command-token normalization.
// It keeps typed raw_command/raw_args evidence for hooks and diagnostics while
// exposing the canonical command that handlers must dispatch against.
type GatewayCommandDispatch struct {
	Known      bool
	Alias      bool
	RawCommand string
	RawArgs    string
	Canonical  string
	Kind       EventKind
	Command    CommandDef
}

// ParseInboundText normalizes a channel message into a shared EventKind/body
// pair. Plain text becomes EventSubmit; recognized slash commands become their
// mapped EventKind; unknown slash commands become EventUnknown.
func ParseInboundText(text string) (EventKind, string) {
	body := strings.TrimSpace(text)
	if !strings.HasPrefix(body, "/") {
		return EventSubmit, body
	}
	resolved := ResolveGatewayCommandDispatch(body)
	if !resolved.Known {
		return EventUnknown, ""
	}
	cmd := resolved.Command
	if cmd.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
		return EventSubmit, body
	}
	if cmd.Kind == EventSteer || cmd.Kind == EventTitle || cmd.Kind == EventSessions || cmd.Kind == EventProfile || cmd.Kind == EventSkills || cmd.Kind == EventReasoning || cmd.Kind == EventBusy || cmd.Kind == EventTTS || cmd.Kind == EventRetry {
		return cmd.Kind, body
	}
	return cmd.Kind, ""
}

// ResolveGatewayCommandDispatch maps a typed gateway slash command or alias to
// the canonical command definition while preserving the typed command and raw
// argument tail for hook contexts.
func ResolveGatewayCommandDispatch(text string) GatewayCommandDispatch {
	token, args := splitGatewayCommandLine(text)
	raw := slashCommandName(token)
	out := GatewayCommandDispatch{RawCommand: raw, RawArgs: args}
	if raw == "" {
		return out
	}
	cmd, ok := ResolveCommand(raw)
	if !ok {
		return out
	}
	out.Known = true
	out.Alias = raw != cmd.Name
	out.Canonical = cmd.Name
	out.Kind = cmd.Kind
	out.Command = cmd
	return out
}

// UnknownSlashCommandGuidance matches the bounded Hermes gateway guidance: it
// tells users how to inspect commands or resend literal text without allowing
// the slash token to fall through as a provider prompt.
func UnknownSlashCommandGuidance(name string) string {
	cleaned := slashCommandName(name)
	if cleaned == "" {
		cleaned = "unknown"
	}
	return fmt.Sprintf("unknown command `/%s`. Type /commands to see what's available, or resend without the leading slash to send as a regular message.", cleaned)
}

// GatewayHelpLines renders registry-driven help output in canonical order,
// excluding commands marked unavailable (CLI-only or not yet built for gateway).
func GatewayHelpLines() []string {
	lines := make([]string, 0, len(CommandRegistry))
	for _, cmd := range CommandRegistry {
		if cmd.ActiveTurnPolicy == CommandActiveTurnPolicyUnavailable {
			continue
		}
		aliasNote := ""
		if len(cmd.Aliases) > 0 {
			aliases := make([]string, len(cmd.Aliases))
			for i, alias := range cmd.Aliases {
				aliases[i] = "`/" + alias + "`"
			}
			aliasNote = " (alias: " + strings.Join(aliases, ", ") + ")"
		}
		lines = append(lines, fmt.Sprintf("`/%s` -- %s%s", cmd.Name, cmd.Description, aliasNote))
	}
	return lines
}

func gatewayHelpText() string {
	return "Gormes is online. Available commands:\n" + strings.Join(GatewayHelpLines(), "\n")
}

// TelegramBotCommands returns the canonical Telegram command menu in registry
// order. Aliases are intentionally excluded from the menu surface.
func TelegramBotCommands() []PlatformCommand {
	out := make([]PlatformCommand, 0, len(CommandRegistry))
	for _, cmd := range CommandRegistry {
		if !telegramMenuCommandVisible(cmd) {
			continue
		}
		out = append(out, PlatformCommand{
			Name:        strings.ReplaceAll(cmd.Name, "-", "_"),
			Description: cmd.Description,
		})
	}
	return out
}

func telegramMenuCommandVisible(cmd CommandDef) bool {
	if cmd.ActiveTurnPolicy != CommandActiveTurnPolicyUnavailable {
		return true
	}
	_, ok := telegramMenuUnavailableCommands[cmd.Name]
	return ok
}

// TelegramBotCommandsWith returns the canonical Telegram menu plus dynamic
// commands. Dynamic command names are normalized for Telegram's underscore-only
// command shape and sorted for deterministic platform registration.
func TelegramBotCommandsWith(dynamic []PlatformCommand) []PlatformCommand {
	out := TelegramBotCommands()
	seen := platformCommandNameSet(out)
	for _, cmd := range sortedPlatformCommands(dynamic) {
		name := normalizeTelegramCommandName(cmd.Name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, PlatformCommand{Name: name, Description: strings.TrimSpace(cmd.Description)})
	}
	return out
}

// SlackSubcommandMap returns the canonical slash mapping Slack should expose.
// Both canonical names and aliases resolve to their slash-prefixed entry.
func SlackSubcommandMap() map[string]string {
	out := make(map[string]string, len(CommandRegistry)*2)
	for _, cmd := range CommandRegistry {
		out[cmd.Name] = "/" + cmd.Name
		for _, alias := range cmd.Aliases {
			out[alias] = "/" + alias
		}
	}
	return out
}

// SlackSubcommandMapWith returns the canonical Slack command mapping plus
// dynamic commands. Callers are responsible for passing only enabled commands.
func SlackSubcommandMapWith(dynamic []PlatformCommand) map[string]string {
	out := SlackSubcommandMap()
	for _, cmd := range sortedPlatformCommands(dynamic) {
		name := normalizeSlackCommandName(cmd.Name)
		if name == "" {
			continue
		}
		out[name] = "/" + name
	}
	return out
}

func platformCommandNameSet(commands []PlatformCommand) map[string]bool {
	out := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		out[cmd.Name] = true
	}
	return out
}

func sortedPlatformCommands(commands []PlatformCommand) []PlatformCommand {
	out := append([]PlatformCommand(nil), commands...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Description < out[j].Description
	})
	return out
}

func normalizeTelegramCommandName(name string) string {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "/")
	name = strings.ReplaceAll(name, "-", "_")
	return strings.Trim(name, "_")
}

func normalizeSlackCommandName(name string) string {
	name = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "/")
	name = strings.ReplaceAll(name, "_", "-")
	return strings.Trim(name, "-")
}

func splitGatewayCommandLine(input string) (token, args string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ""
	}
	for i, r := range trimmed {
		switch r {
		case ' ', '\t', '\n', '\r':
			return trimmed[:i], strings.TrimSpace(trimmed[i:])
		}
	}
	return trimmed, ""
}
