package commands

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/alias"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/busy"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/registry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands/typo"
)

type ActiveTurnPolicy = registry.ActiveTurnPolicy

const (
	ActiveTurnPolicyBypass      = registry.ActiveTurnPolicyBypass
	ActiveTurnPolicyQueue       = registry.ActiveTurnPolicyQueue
	ActiveTurnPolicyBusyReject  = registry.ActiveTurnPolicyBusyReject
	ActiveTurnPolicyUnavailable = registry.ActiveTurnPolicyUnavailable
)

type CommandSurface = registry.CommandSurface

const (
	CommandSurfaceShared  = registry.CommandSurfaceShared
	CommandSurfaceCLI     = registry.CommandSurfaceCLI
	CommandSurfaceGateway = registry.CommandSurfaceGateway
)

type CommandPolicy = registry.CommandPolicy

var CommandRegistry = registry.CommandRegistry

func ResolveCommandPolicy(name string) (CommandPolicy, bool) {
	return registry.ResolveCommandPolicy(name)
}

type ActiveTurnVerdict = registry.ActiveTurnVerdict

func EvaluateActiveTurnVerdict(name string, isBusy bool) ActiveTurnVerdict {
	return registry.EvaluateActiveTurnVerdict(name, isBusy)
}

func SlashLeaksToModelPrompt(text string) bool { return registry.SlashLeaksToModelPrompt(text) }

var ErrBusyCommandActive = busy.ErrBusyCommandActive
var ErrBusyCommandInvalid = busy.ErrBusyCommandInvalid

type BusyCommandGuard = busy.BusyCommandGuard

type BusyInputVerdict = busy.BusyInputVerdict

func NewBusyCommandGuard() *BusyCommandGuard { return busy.NewBusyCommandGuard() }

type CommandAliasKind = alias.CommandAliasKind

const (
	CommandAliasExact     = alias.CommandAliasExact
	CommandAliasAlias     = alias.CommandAliasAlias
	CommandAliasPrefix    = alias.CommandAliasPrefix
	CommandAliasAmbiguous = alias.CommandAliasAmbiguous
	CommandAliasUnknown   = alias.CommandAliasUnknown
)

type CommandAliasResolution = alias.CommandAliasResolution

func ResolveCommandAlias(input string) CommandAliasResolution {
	return alias.ResolveCommandAlias(input)
}

type QuickCommandAlias = alias.QuickCommandAlias
type QuickCommandAliasKind = alias.QuickCommandAliasKind

const (
	QuickCommandAliasUnknown           = alias.QuickCommandAliasUnknown
	QuickCommandAliasResolved          = alias.QuickCommandAliasResolved
	QuickCommandAliasCycle             = alias.QuickCommandAliasCycle
	QuickCommandAliasUnsupportedTarget = alias.QuickCommandAliasUnsupportedTarget
	QuickCommandAliasUnsupportedType   = alias.QuickCommandAliasUnsupportedType
	QuickCommandAliasMissingTarget     = alias.QuickCommandAliasMissingTarget
)

type QuickCommandAliasResolution = alias.QuickCommandAliasResolution

func ResolveQuickCommandAlias(input string, quick map[string]QuickCommandAlias) QuickCommandAliasResolution {
	return alias.ResolveQuickCommandAlias(input, quick)
}

func TypoSuggestion(args []string) (string, bool) { return typo.TypoSuggestion(args) }
