package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"

type ActiveTurnPolicy = commands.ActiveTurnPolicy

const (
	ActiveTurnPolicyBypass      = commands.ActiveTurnPolicyBypass
	ActiveTurnPolicyQueue       = commands.ActiveTurnPolicyQueue
	ActiveTurnPolicyBusyReject  = commands.ActiveTurnPolicyBusyReject
	ActiveTurnPolicyUnavailable = commands.ActiveTurnPolicyUnavailable
)

type CommandSurface = commands.CommandSurface

const (
	CommandSurfaceShared  = commands.CommandSurfaceShared
	CommandSurfaceCLI     = commands.CommandSurfaceCLI
	CommandSurfaceGateway = commands.CommandSurfaceGateway
)

type CommandPolicy = commands.CommandPolicy

var CommandRegistry = commands.CommandRegistry

func ResolveCommandPolicy(name string) (CommandPolicy, bool) {
	return commands.ResolveCommandPolicy(name)
}

type ActiveTurnVerdict = commands.ActiveTurnVerdict

func EvaluateActiveTurnVerdict(name string, busy bool) ActiveTurnVerdict {
	return commands.EvaluateActiveTurnVerdict(name, busy)
}

func SlashLeaksToModelPrompt(text string) bool { return commands.SlashLeaksToModelPrompt(text) }

var ErrBusyCommandActive = commands.ErrBusyCommandActive
var ErrBusyCommandInvalid = commands.ErrBusyCommandInvalid

type BusyCommandGuard = commands.BusyCommandGuard

type BusyInputVerdict = commands.BusyInputVerdict

func NewBusyCommandGuard() *BusyCommandGuard { return commands.NewBusyCommandGuard() }

// CommandAliasKind classifies how a typed slash token resolves before command
// dispatch.
type CommandAliasKind = commands.CommandAliasKind

const (
	CommandAliasExact     = commands.CommandAliasExact
	CommandAliasAlias     = commands.CommandAliasAlias
	CommandAliasPrefix    = commands.CommandAliasPrefix
	CommandAliasAmbiguous = commands.CommandAliasAmbiguous
	CommandAliasUnknown   = commands.CommandAliasUnknown
)

type CommandAliasResolution = commands.CommandAliasResolution

func ResolveCommandAlias(input string) CommandAliasResolution {
	return commands.ResolveCommandAlias(input)
}

type QuickCommandAlias = commands.QuickCommandAlias
type QuickCommandAliasKind = commands.QuickCommandAliasKind

const (
	QuickCommandAliasUnknown           = commands.QuickCommandAliasUnknown
	QuickCommandAliasResolved          = commands.QuickCommandAliasResolved
	QuickCommandAliasCycle             = commands.QuickCommandAliasCycle
	QuickCommandAliasUnsupportedTarget = commands.QuickCommandAliasUnsupportedTarget
	QuickCommandAliasUnsupportedType   = commands.QuickCommandAliasUnsupportedType
	QuickCommandAliasMissingTarget     = commands.QuickCommandAliasMissingTarget
)

type QuickCommandAliasResolution = commands.QuickCommandAliasResolution

func ResolveQuickCommandAlias(input string, quick map[string]QuickCommandAlias) QuickCommandAliasResolution {
	return commands.ResolveQuickCommandAlias(input, quick)
}

// TypoSuggestion is the pre-Cobra extension point for deterministic,
// secret-safe guidance on removed command spellings.
func TypoSuggestion(args []string) (string, bool) { return commands.TypoSuggestion(args) }
