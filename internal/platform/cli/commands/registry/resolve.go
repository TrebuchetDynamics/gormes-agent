package registry

import (
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

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
	key := NormalizeCommandToken(name)
	if key == "" {
		return CommandPolicy{}, false
	}
	cmd, ok := commandPolicyLookup[key]
	return cmd, ok
}

func NormalizeCommandToken(raw string) string {
	key := textvalue.LowerTrim(raw)
	key = strings.TrimPrefix(key, "/")
	if i := strings.IndexAny(key, " \t\r\n"); i >= 0 {
		key = key[:i]
	}
	return key
}
