package gateway

import (
	"slices"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/commandregistry"
)

// CommandDef is the canonical slash-command definition shared by gateway
// parsing, help text, and per-platform command exposure helpers.
type CommandDef = commandregistry.CommandDef

// CommandActiveTurnPolicy documents how a command behaves while a gateway turn
// is already active.
type CommandActiveTurnPolicy = commandregistry.CommandActiveTurnPolicy

const (
	CommandActiveTurnPolicyImmediate   = commandregistry.CommandActiveTurnPolicyImmediate
	CommandActiveTurnPolicyReject      = commandregistry.CommandActiveTurnPolicyReject
	CommandActiveTurnPolicyDrain       = commandregistry.CommandActiveTurnPolicyDrain
	CommandActiveTurnPolicyUnavailable = commandregistry.CommandActiveTurnPolicyUnavailable
)

// PlatformCommand is the platform-facing command/menu shape used for channel
// exposure helpers such as Telegram bot menus.
type PlatformCommand = commandregistry.PlatformCommand

// CommandRegistry is the single source of truth for gateway slash commands.
var CommandRegistry = commandregistry.CommandRegistry

// ResolveCommand maps a slash command or alias to its canonical definition.
func ResolveCommand(name string) (CommandDef, bool) {
	return commandregistry.ResolveCommand(name)
}

func slashCommandName(name string) string {
	return commandregistry.SlashCommandName(name)
}

func isRecognizedUnavailableSlashCommand(name string) bool {
	return commandregistry.IsRecognizedUnavailableSlashCommand(name)
}

// GatewayCommandDispatch is the gateway-visible command-token normalization.
// It keeps typed raw_command/raw_args evidence for hooks and diagnostics while
// exposing the canonical command that handlers must dispatch against.
type GatewayCommandDispatch = commandregistry.GatewayCommandDispatch

// ParseInboundText normalizes a channel message into a shared EventKind/body
// pair. Plain text becomes EventSubmit; recognized slash commands become their
// mapped EventKind; unknown slash commands become EventUnknown.
func ParseInboundText(text string) (EventKind, string) {
	return commandregistry.ParseInboundText(text)
}

// ParseInboundTextPreserveUnknown matches ParseInboundText except that the raw
// slash text is preserved as the body for unknown commands. Manager-backed
// adapters use this so the manager can resolve dynamic runtime slash commands
// (for example /skill-name) before falling back to unknown-command guidance,
// while legacy adapters can keep ParseInboundText's empty unknown body to avoid
// accidentally submitting unknown slash text.
func ParseInboundTextPreserveUnknown(text string) (EventKind, string) {
	return commandregistry.ParseInboundTextPreserveUnknown(text)
}

// ResolveGatewayCommandDispatch maps a typed gateway slash command or alias to
// the canonical command definition while preserving the typed command and raw
// argument tail for hook contexts.
func ResolveGatewayCommandDispatch(text string) GatewayCommandDispatch {
	return commandregistry.ResolveGatewayCommandDispatch(text)
}

// UnknownSlashCommandGuidance matches the bounded Hermes gateway guidance: it
// tells users how to inspect commands or resend literal text without allowing
// the slash token to fall through as a provider prompt.
func UnknownSlashCommandGuidance(name string) string {
	return commandregistry.UnknownSlashCommandGuidance(name)
}

// GatewayHelpLines renders registry-driven help output in canonical order,
// excluding commands marked unavailable (CLI-only or not yet built for gateway).
func GatewayHelpLines() []string {
	return commandregistry.GatewayHelpLines()
}

func gatewayHelpText() string {
	return commandregistry.GatewayHelpText()
}

// TelegramBotCommands returns the canonical Telegram command menu in registry
// order. Aliases are intentionally excluded from the menu surface.
func TelegramBotCommands() []PlatformCommand {
	return commandregistry.TelegramBotCommands()
}

func telegramMenuCommandVisible(cmd CommandDef) bool {
	return commandregistry.TelegramMenuCommandVisible(cmd)
}

// TelegramBotCommandsWith returns the canonical Telegram menu plus dynamic
// commands. Dynamic command names are normalized for Telegram's underscore-only
// command shape and sorted for deterministic platform registration.
func TelegramBotCommandsWith(dynamic []PlatformCommand) []PlatformCommand {
	return commandregistry.TelegramBotCommandsWith(dynamic)
}

// SlackSubcommandMap returns the canonical slash mapping Slack should expose.
// Both canonical names and aliases resolve to their slash-prefixed entry.
func SlackSubcommandMap() map[string]string {
	return commandregistry.SlackSubcommandMap()
}

// SlackSubcommandMapWith returns the canonical Slack command mapping plus
// dynamic commands. Callers are responsible for passing only enabled commands.
func SlackSubcommandMapWith(dynamic []PlatformCommand) map[string]string {
	return commandregistry.SlackSubcommandMapWith(dynamic)
}

func normalizeTelegramCommandName(name string) string {
	return commandregistry.NormalizeTelegramCommandName(name)
}

func normalizeSlackCommandName(name string) string {
	return commandregistry.NormalizeSlackCommandName(name)
}

func splitGatewayCommandLine(input string) (token, args string) {
	return commandregistry.SplitGatewayCommandLine(input)
}

func slashCommandKindCarriesBody(kind EventKind) bool {
	return commandregistry.SlashCommandKindCarriesBody(kind)
}

func platformCommandNameSet(commands []PlatformCommand) map[string]bool {
	out := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		out[cmd.Name] = true
	}
	return out
}

func sortedPlatformCommands(commands []PlatformCommand) []PlatformCommand {
	out := slices.Clone(commands)
	commandregistry.SortPlatformCommands(out)
	return out
}

func trimPlatformCommandDescription(description string) string {
	return strings.TrimSpace(description)
}
