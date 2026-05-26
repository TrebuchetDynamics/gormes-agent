package cli

import "strings"

// ActiveTurnPolicy classifies how a recognized slash command behaves when an
// agent turn is already active. The set is closed: every entry in
// CommandRegistry must declare exactly one of these values, and unknown command
// names are evaluated as ActiveTurnPolicyUnavailable so they never reach the
// kernel as ordinary prompt text.
type ActiveTurnPolicy string

const (
	// ActiveTurnPolicyBypass marks a command that is safe to dispatch while a
	// turn is active. Help, status, and turn-control commands fall here.
	ActiveTurnPolicyBypass ActiveTurnPolicy = "bypass"
	// ActiveTurnPolicyQueue marks a command that should defer until the current
	// turn finishes rather than mutate runtime state mid-turn.
	ActiveTurnPolicyQueue ActiveTurnPolicy = "queue"
	// ActiveTurnPolicyBusyReject marks a mutating command that must not run
	// during an active turn; the operator receives a busy notice with /stop
	// guidance instead.
	ActiveTurnPolicyBusyReject ActiveTurnPolicy = "busy_reject"
	// ActiveTurnPolicyUnavailable marks a command that is recognized in the
	// registry (so its slash form does not leak to the model) but not yet
	// implemented in Gormes. Operators see explicit unavailable evidence.
	ActiveTurnPolicyUnavailable ActiveTurnPolicy = "unavailable"
)

// CommandSurface tags where a command is exposed.
type CommandSurface string

const (
	// CommandSurfaceShared is exposed in both CLI and gateway adapters.
	CommandSurfaceShared CommandSurface = "shared"
	// CommandSurfaceCLI is exposed only in the local CLI/TUI surface.
	CommandSurfaceCLI CommandSurface = "cli"
	// CommandSurfaceGateway is exposed only in gateway/messaging adapters.
	CommandSurfaceGateway CommandSurface = "gateway"
)

// CommandPolicy is the canonical CLI-side declaration for a slash command.
// The struct is immutable data; callers must not mutate fields after init.
type CommandPolicy struct {
	Name             string
	Description      string
	Aliases          []string
	Surface          CommandSurface
	ActiveTurnPolicy ActiveTurnPolicy
	// Ported reports whether Gormes has a real handler. Unported entries stay
	// in the registry so unknown-vs-unavailable can be distinguished, but they
	// always render unavailable evidence regardless of busy state.
	Ported bool
	// Subcommands is the Hermes-canonical, order-preserving list of static
	// sub-tokens a TUI completer should surface after the command name plus a
	// space (e.g. /reasoning ⏎ none, minimal, low, …). Order matches upstream
	// hermes_cli/commands.py CommandDef.subcommands. Dynamic per-runtime menus
	// (/model, /skin, /personality) are intentionally not encoded here.
	Subcommands []string
}

// CommandRegistry is the single source of truth for the active-turn behavior
// of every slash command Gormes recognizes. Entries that are intentionally not
// yet ported declare ActiveTurnPolicyUnavailable so callers can return visible
// "unavailable" evidence instead of letting the slash text reach the kernel.
//
// The registry mirrors the surface-level inventory of upstream
// hermes_cli/commands.py at edc78e25 but only commits to active-turn semantics
// here. Handler ports land in dedicated rows.
var CommandRegistry = []CommandPolicy{
	// Ported (gateway-handled) commands.
	{
		Name:             "help",
		Description:      "Show available commands",
		Aliases:          []string{"start"},
		Surface:          CommandSurfaceShared,
		ActiveTurnPolicy: ActiveTurnPolicyBypass,
		Ported:           true,
	},
	{
		Name:             "new",
		Description:      "Start a fresh session",
		Aliases:          []string{"reset"},
		Surface:          CommandSurfaceShared,
		ActiveTurnPolicy: ActiveTurnPolicyBusyReject,
		Ported:           true,
	},
	{
		Name:             "stop",
		Description:      "Cancel the active turn",
		Surface:          CommandSurfaceShared,
		ActiveTurnPolicy: ActiveTurnPolicyBypass,
		Ported:           true,
	},
	{
		Name:             "restart",
		Description:      "Gracefully restart the gateway after draining active runs",
		Surface:          CommandSurfaceShared,
		ActiveTurnPolicy: ActiveTurnPolicyBypass,
		Ported:           true,
	},
	{
		Name:             "reasoning",
		Description:      "Manage reasoning effort and display",
		Surface:          CommandSurfaceShared,
		ActiveTurnPolicy: ActiveTurnPolicyQueue,
		Ported:           true,
		Subcommands:      []string{"none", "minimal", "low", "medium", "high", "xhigh", "show", "hide", "on", "off"},
	},

	// Recognized-but-not-yet-ported commands. These mirror the upstream
	// hermes_cli/commands.py inventory so unknown slash text cannot bypass
	// recognition by typing a yet-to-land command. Each entry is intentionally
	// unavailable until its dedicated implementation row lands.
	{Name: "agents", Description: "Show active agents and running tasks", Aliases: []string{"tasks"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "approve", Description: "Approve a pending dangerous command", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "background", Description: "Run a prompt in the background", Aliases: []string{"bg", "btw"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "branch", Description: "Branch the current session", Aliases: []string{"fork"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true},
	{Name: "browser", Description: "Connect browser tools to your live Chrome via CDP", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"status", "connect"}},
	{Name: "busy", Description: "Control what Enter does while Gormes is working", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"queue", "steer", "interrupt", "status"}},
	{Name: "clear", Description: "Clear screen and start a new session", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true},
	{Name: "commands", Description: "Browse all commands and skills", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "compress", Description: "Manually compress conversation context", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "compact", Description: "Toggle compact transcript", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"on", "off", "toggle"}},
	{Name: "config", Description: "Show current configuration", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "copy", Description: "Copy the last assistant response to clipboard", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "cron", Description: "Manage scheduled tasks", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "debug", Description: "Upload debug report and get shareable links", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "deny", Description: "Deny a pending dangerous command", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "details", Description: "Toggle detailed activity sections", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"hidden", "collapsed", "expanded", "cycle", "toggle", "thinking", "tools", "subagents", "activity"}},
	{Name: "fast", Description: "Toggle fast mode", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "goal", Description: "Set a standing goal Gormes works on across turns until achieved", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"pause", "resume", "clear", "status"}},
	{Name: "history", Description: "Show conversation history", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "image", Description: "Attach a local image file for your next prompt", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "insights", Description: "Show usage insights and analytics", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "kanban", Description: "Manage the durable multi-agent task board", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"init", "create", "list", "show", "claim", "complete", "block", "unblock", "link", "log", "tail"}},
	{Name: "logs", Description: "View gateway logs", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "mouse", Description: "Toggle terminal mouse tracking", Aliases: []string{"scroll"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true, Subcommands: []string{"on", "off", "toggle"}},
	{Name: "model", Description: "Switch model for this session", Aliases: []string{"provider"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "paste", Description: "Attach clipboard image", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "personality", Description: "Set a predefined personality", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "platform", Description: "Pause, resume, or list a failing gateway platform", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"list", "pause", "resume"}},
	{Name: "platforms", Description: "Show gateway/messaging platform status", Aliases: []string{"gateway"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "plugins", Description: "List installed plugins", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "profile", Description: "Show active profile name and home directory", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "queue", Description: "Queue a prompt for the next turn", Aliases: []string{"q"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "quit", Description: "Exit the CLI", Aliases: []string{"exit"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true},
	{Name: "reload", Description: "Reload gateway config without restarting", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "reload-skills", Description: "Re-scan installed skills for newly added or removed skill commands", Aliases: []string{"reload_skills"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "reload-mcp", Description: "Reload MCP servers from config", Aliases: []string{"reload_mcp"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "resume", Description: "Resume a previously-named session", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true},
	{Name: "retry", Description: "Retry the last message", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "rollback", Description: "List or restore filesystem checkpoints", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "save", Description: "Save the current conversation", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBusyReject, Ported: true},
	{Name: "sessions", Description: "List or show sessions", Aliases: []string{"session"}, Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "sethome", Description: "Set this chat as the home channel", Aliases: []string{"set-home"}, Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "skills", Description: "List or inspect installed skills", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"list", "inspect"}},
	{Name: "spawn", Description: "Spawn a channel-native agent thread", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "skin", Description: "Show or change the display skin/theme", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "snapshot", Description: "Create or restore state snapshots", Aliases: []string{"snap"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "status", Description: "Show session info", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "statusbar", Description: "Toggle the context/model status bar", Aliases: []string{"sb"}, Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"on", "off", "top", "bottom", "toggle"}},
	{Name: "steer", Description: "Inject a message after the next tool call", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "title", Description: "Set a title for the current session", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "topic", Description: "Manage Telegram multi-session topics", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"help", "off"}},
	{Name: "tools", Description: "Manage tools", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "toolsets", Description: "List available toolsets", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "tts", Description: "Configure text-to-speech", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"on", "off", "speed", "voice", "engine", "language"}},
	{Name: "undo", Description: "Remove the last user/assistant exchange", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "update", Description: "Update Gormes to the latest version", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "usage", Description: "Show token usage and rate limits", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "verbose", Description: "Cycle tool progress display", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
	{Name: "voice", Description: "Toggle voice mode", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "yolo", Description: "Toggle YOLO mode", Surface: CommandSurfaceShared, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "footer", Description: "Toggle the gateway footer", Surface: CommandSurfaceGateway, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "gquota", Description: "Show Google Gemini Code Assist quota usage", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyUnavailable},
	{Name: "indicator", Description: "Pick TUI busy-indicator style", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"ascii", "emoji", "kaomoji", "unicode"}},
	{Name: "curator", Description: "Background skill maintenance", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true, Subcommands: []string{"status", "run", "pause", "resume", "pin", "unpin", "backup", "rollback", "restore", "archive", "list-archived", "prune"}},
	{Name: "redraw", Description: "Redraw the terminal screen", Surface: CommandSurfaceCLI, ActiveTurnPolicy: ActiveTurnPolicyBypass, Ported: true},
}

var commandPolicyLookup = buildCommandPolicyLookup()

func buildCommandPolicyLookup() map[string]CommandPolicy {
	out := make(map[string]CommandPolicy, len(CommandRegistry)*2)
	for _, cmd := range CommandRegistry {
		out[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			out[alias] = cmd
		}
	}
	return out
}

// ResolveCommandPolicy normalizes a slash command token (with or without the
// leading slash, in any case, possibly padded with whitespace) and returns the
// matching CommandPolicy. The second return is false when the token does not
// resolve to a recognized command.
func ResolveCommandPolicy(name string) (CommandPolicy, bool) {
	key := normalizeCommandToken(name)
	if key == "" {
		return CommandPolicy{}, false
	}
	cmd, ok := commandPolicyLookup[key]
	return cmd, ok
}

func normalizeCommandToken(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexAny(key, " \t\r\n"); i >= 0 {
		key = key[:i]
	}
	return key
}

// ActiveTurnVerdict is the result of evaluating a slash command against the
// current active-turn state. Allowed reports whether the command should be
// dispatched immediately; Evidence is the operator-facing reason when the
// command is not allowed (busy, queue, or unavailable).
type ActiveTurnVerdict struct {
	Name     string
	Policy   ActiveTurnPolicy
	Known    bool
	Allowed  bool
	Evidence string
}

// EvaluateActiveTurnVerdict returns the dispatch decision for a slash command
// in the current active-turn state. Unknown commands evaluate as
// ActiveTurnPolicyUnavailable with explicit evidence so the caller never lets
// the original slash text reach the kernel as ordinary prompt content.
func EvaluateActiveTurnVerdict(name string, busy bool) ActiveTurnVerdict {
	cmd, ok := ResolveCommandPolicy(name)
	if !ok {
		return ActiveTurnVerdict{
			Name:     normalizeCommandToken(name),
			Policy:   ActiveTurnPolicyUnavailable,
			Known:    false,
			Allowed:  false,
			Evidence: "unknown command — no slash command by that name is available",
		}
	}
	switch cmd.ActiveTurnPolicy {
	case ActiveTurnPolicyUnavailable:
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyUnavailable,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " is recognized but unavailable in this build",
		}
	case ActiveTurnPolicyBypass:
		return ActiveTurnVerdict{
			Name:    cmd.Name,
			Policy:  ActiveTurnPolicyBypass,
			Known:   true,
			Allowed: true,
		}
	case ActiveTurnPolicyQueue:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  ActiveTurnPolicyQueue,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyQueue,
			Known:    true,
			Allowed:  false,
			Evidence: "/" + cmd.Name + " was queued — it will run after the current turn finishes",
		}
	case ActiveTurnPolicyBusyReject:
		if !busy {
			return ActiveTurnVerdict{
				Name:    cmd.Name,
				Policy:  ActiveTurnPolicyBusyReject,
				Known:   true,
				Allowed: true,
			}
		}
		return ActiveTurnVerdict{
			Name:     cmd.Name,
			Policy:   ActiveTurnPolicyBusyReject,
			Known:    true,
			Allowed:  false,
			Evidence: "Gormes is busy — finish the current turn or send /stop before /" + cmd.Name,
		}
	}
	// Unreachable for valid registry entries; treat as unavailable defensively.
	return ActiveTurnVerdict{
		Name:     cmd.Name,
		Policy:   ActiveTurnPolicyUnavailable,
		Known:    true,
		Allowed:  false,
		Evidence: "/" + cmd.Name + " has no defined active-turn policy",
	}
}

// SlashLeaksToModelPrompt reports whether the given inbound text would be
// forwarded to the model kernel as ordinary prompt content. Plain text leaks
// (that is the intended path); slash commands — recognized or not — are
// handled by the dispatcher and must not enter the prompt as command text.
func SlashLeaksToModelPrompt(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	return !strings.HasPrefix(trimmed, "/")
}
