package slash

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
)

type Fallback struct {
	Handled bool
	Status  string
}

func FallbackForInput(input string) Fallback {
	resolved := cli.ResolveCommandAlias(input)
	if resolved.RawCommand == "" {
		return Fallback{}
	}
	switch resolved.Kind {
	case cli.CommandAliasExact, cli.CommandAliasAlias, cli.CommandAliasPrefix:
		return Fallback{Handled: true, Status: KnownUnhandledStatus(resolved.RawCommand, resolved.Policy)}
	case cli.CommandAliasAmbiguous:
		return Fallback{Handled: true, Status: AmbiguousNameStatus(resolved.Matches)}
	case cli.CommandAliasUnknown:
		return Fallback{
			Handled: true,
			Status:  fmt.Sprintf("unknown command /%s — no slash command by that name is available", resolved.RawCommand),
		}
	}
	return Fallback{}
}

func KnownUnhandledStatus(typed string, policy cli.CommandPolicy) string {
	display := "/" + policy.Name
	if typed != policy.Name {
		display = fmt.Sprintf("/%s -> /%s", typed, policy.Name)
	}
	switch policy.Surface {
	case cli.CommandSurfaceGateway:
		return display + " is recognized but requires gateway support in the native TUI"
	default:
		return display + " is recognized but unavailable in the native TUI"
	}
}

func AmbiguousNameStatus(matches []string) string {
	limit := len(matches)
	if limit > 6 {
		limit = 6
	}
	names := append([]string(nil), matches[:limit]...)
	suffix := ""
	if len(matches) > limit {
		suffix = ", ..."
	}
	return "ambiguous command: " + strings.Join(names, ", ") + suffix
}
