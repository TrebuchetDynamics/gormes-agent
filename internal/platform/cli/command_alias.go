package cli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/commands"

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
