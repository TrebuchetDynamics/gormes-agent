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
	{Name: "retry", Description: "Retry the last message (resend to agent)", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "undo", Description: "Remove the last user/assistant exchange", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "title", Description: "Set a title for the current session", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "branch", Description: "Branch the current session (explore a different path)", Kind: EventUnknown, Aliases: []string{"fork"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "compress", Description: "Manually compress conversation context", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "rollback", Description: "List or restore filesystem checkpoints", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "snapshot", Description: "Create or restore state snapshots of Hermes config/state", Kind: EventUnknown, Aliases: []string{"snap"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "approve", Description: "Approve a pending dangerous command", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "deny", Description: "Deny a pending dangerous command", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "background", Description: "Run a prompt in the background", Kind: EventUnknown, Aliases: []string{"bg"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "btw", Description: "Ephemeral side question using session context (no tools, not persisted)", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "agents", Description: "Show active agents and running tasks", Kind: EventUnknown, Aliases: []string{"tasks"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "queue", Description: "Queue a prompt for the next turn (doesn't interrupt)", Kind: EventUnknown, Aliases: []string{"q"}, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
	{Name: "status", Description: "Show session info", Kind: EventUnknown, ActiveTurnPolicy: CommandActiveTurnPolicyUnavailable},
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

// ParseInboundText normalizes a channel message into a shared EventKind/body
// pair. Plain text becomes EventSubmit; recognized slash commands become their
// mapped EventKind; unknown slash commands become EventUnknown.
func ParseInboundText(text string) (EventKind, string) {
	body := strings.TrimSpace(text)
	if !strings.HasPrefix(body, "/") {
		return EventSubmit, body
	}
	cmd, ok := ResolveCommand(body)
	if !ok {
		return EventUnknown, ""
	}
	if cmd.Kind == EventSteer {
		return cmd.Kind, body
	}
	return cmd.Kind, ""
}

// GatewayHelpLines renders registry-driven help output in canonical order.
func GatewayHelpLines() []string {
	lines := make([]string, 0, len(CommandRegistry))
	for _, cmd := range CommandRegistry {
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
		out = append(out, PlatformCommand{
			Name:        strings.ReplaceAll(cmd.Name, "-", "_"),
			Description: cmd.Description,
		})
	}
	return out
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
