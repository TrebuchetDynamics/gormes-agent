package registry

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/registry/active"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/registry/catalog"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/registry/prompt"
)

// ActiveTurnPolicy classifies how a recognized slash command behaves when an
// agent turn is already active.
type ActiveTurnPolicy = catalog.ActiveTurnPolicy

const (
	ActiveTurnPolicyBypass      = catalog.ActiveTurnPolicyBypass
	ActiveTurnPolicyQueue       = catalog.ActiveTurnPolicyQueue
	ActiveTurnPolicyBusyReject  = catalog.ActiveTurnPolicyBusyReject
	ActiveTurnPolicyUnavailable = catalog.ActiveTurnPolicyUnavailable
)

// CommandSurface tags where a command is exposed.
type CommandSurface = catalog.CommandSurface

const (
	CommandSurfaceShared  = catalog.CommandSurfaceShared
	CommandSurfaceCLI     = catalog.CommandSurfaceCLI
	CommandSurfaceGateway = catalog.CommandSurfaceGateway
)

// CommandPolicy is the canonical CLI-side declaration for a slash command.
type CommandPolicy = catalog.CommandPolicy

// CommandRegistry is the single source of truth for the active-turn behavior
// of every slash command Gormes recognizes.
var CommandRegistry = catalog.CommandRegistry

// ActiveTurnVerdict is the result of evaluating a slash command against the
// current active-turn state.
type ActiveTurnVerdict = active.ActiveTurnVerdict

// ResolveCommandPolicy normalizes a slash command token and returns the
// matching command policy.
func ResolveCommandPolicy(name string) (CommandPolicy, bool) {
	return catalog.ResolveCommandPolicy(name)
}

// NormalizeCommandToken returns the canonical command token without a leading
// slash, arguments, surrounding whitespace, or mixed case.
func NormalizeCommandToken(raw string) string {
	return catalog.NormalizeCommandToken(raw)
}

// EvaluateActiveTurnVerdict returns the dispatch decision for a slash command
// in the current active-turn state.
func EvaluateActiveTurnVerdict(name string, busy bool) ActiveTurnVerdict {
	return active.EvaluateActiveTurnVerdict(name, busy)
}

// IsSlashCommandText reports whether text starts with a slash command marker
// after operator-facing whitespace is ignored.
func IsSlashCommandText(text string) bool { return prompt.IsSlashCommandText(text) }

// SlashLeaksToModelPrompt reports whether inbound text would be forwarded to
// the model kernel as ordinary prompt content.
func SlashLeaksToModelPrompt(text string) bool { return prompt.SlashLeaksToModelPrompt(text) }
