package catalog

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
