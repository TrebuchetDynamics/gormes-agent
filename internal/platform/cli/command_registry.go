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
