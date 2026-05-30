package commands

import (
	"fmt"
	"sort"
	"strings"
)

// CommandAliasKind classifies how a typed slash token resolves before command
// dispatch. It intentionally covers only visible command-name behavior; model
// aliases and provider aliases live in their own selector/config rows.
type CommandAliasKind string

const (
	CommandAliasExact     CommandAliasKind = "exact"
	CommandAliasAlias     CommandAliasKind = "alias"
	CommandAliasPrefix    CommandAliasKind = "prefix"
	CommandAliasAmbiguous CommandAliasKind = "ambiguous"
	CommandAliasUnknown   CommandAliasKind = "unknown"
)

// CommandAliasResolution is the normalized command-token result shared by CLI,
// gateway, and TUI tests. RawCommand/RawArgs preserve what the operator typed;
// Canonical/Rewrite are the dispatch-safe command name and slash line.
type CommandAliasResolution struct {
	Kind       CommandAliasKind
	RawCommand string
	RawArgs    string
	Canonical  string
	Rewrite    string
	Matches    []string
	Policy     CommandPolicy
}

// ResolveCommandAlias resolves a slash command token using Hermes-visible
// command-name semantics: exact canonical names and aliases win first, then a
// unique prefix may expand while preserving arguments, and ambiguous/unknown
// prefixes stay explicit guidance states.
func ResolveCommandAlias(input string) CommandAliasResolution {
	token, args := splitCommandLine(input)
	rawCommand := normalizeCommandToken(token)
	out := CommandAliasResolution{
		Kind:       CommandAliasUnknown,
		RawCommand: rawCommand,
		RawArgs:    args,
	}
	if rawCommand == "" {
		return out
	}

	if policy, ok := ResolveCommandPolicy(rawCommand); ok {
		out.Policy = policy
		out.Canonical = policy.Name
		out.Rewrite = joinSlashCommand(policy.Name, args)
		if rawCommand == policy.Name {
			out.Kind = CommandAliasExact
		} else {
			out.Kind = CommandAliasAlias
		}
		return out
	}

	matches := commandAliasPrefixMatches(rawCommand)
	switch len(matches) {
	case 0:
		return out
	case 1:
		match := matches[0]
		out.Kind = CommandAliasPrefix
		out.Policy = match.policy
		out.Canonical = match.policy.Name
		out.Rewrite = joinSlashCommand(match.policy.Name, args)
		return out
	default:
		out.Kind = CommandAliasAmbiguous
		out.Matches = make([]string, len(matches))
		for i, match := range matches {
			out.Matches[i] = "/" + match.name
		}
		return out
	}
}

type commandAliasMatch struct {
	name   string
	policy CommandPolicy
}

func commandAliasPrefixMatches(prefix string) []commandAliasMatch {
	seen := make(map[string]CommandPolicy)
	for _, cmd := range CommandRegistry {
		if strings.HasPrefix(cmd.Name, prefix) {
			seen[cmd.Name] = cmd
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				seen[alias] = cmd
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]commandAliasMatch, 0, len(names))
	for _, name := range names {
		out = append(out, commandAliasMatch{name: name, policy: seen[name]})
	}
	return out
}

// QuickCommandAlias mirrors the command-name subset of Hermes quick_commands
// entries needed for deterministic alias tests. Exec quick commands are runtime
// actions; this helper only rewrites aliases.
type QuickCommandAlias struct {
	Type   string
	Target string
}

type QuickCommandAliasKind string

const (
	QuickCommandAliasUnknown           QuickCommandAliasKind = "unknown"
	QuickCommandAliasResolved          QuickCommandAliasKind = "resolved"
	QuickCommandAliasCycle             QuickCommandAliasKind = "cycle"
	QuickCommandAliasUnsupportedTarget QuickCommandAliasKind = "unsupported_target"
	QuickCommandAliasUnsupportedType   QuickCommandAliasKind = "unsupported_type"
	QuickCommandAliasMissingTarget     QuickCommandAliasKind = "missing_target"
)

type QuickCommandAliasResolution struct {
	Kind       QuickCommandAliasKind
	RawCommand string
	RawArgs    string
	Canonical  string
	Rewrite    string
	Chain      []string
	Matches    []string
	Evidence   string
}

// ResolveQuickCommandAlias rewrites a configured quick-command alias into a
// dispatchable slash command while preserving user arguments and refusing
// recursion loops. Targets may point at another quick alias before landing on a
// real slash command.
func ResolveQuickCommandAlias(input string, quick map[string]QuickCommandAlias) QuickCommandAliasResolution {
	token, args := splitCommandLine(input)
	start := normalizeCommandToken(token)
	out := QuickCommandAliasResolution{
		Kind:       QuickCommandAliasUnknown,
		RawCommand: start,
		RawArgs:    args,
	}
	if start == "" || len(quick) == 0 {
		return out
	}
	current := start
	currentArgs := args
	visited := make(map[string]struct{})
	for {
		if _, seen := visited[current]; seen {
			out.Kind = QuickCommandAliasCycle
			out.Evidence = "quick-command alias cycle detected: " + strings.Join(append(out.Chain, current), " -> ")
			return out
		}
		def, ok := quick[current]
		if !ok {
			return out
		}
		visited[current] = struct{}{}
		out.Chain = append(out.Chain, current)

		if strings.ToLower(strings.TrimSpace(def.Type)) != "alias" {
			out.Kind = QuickCommandAliasUnsupportedType
			out.Evidence = fmt.Sprintf("quick command '/%s' has unsupported type %q for alias rewrite", current, def.Type)
			return out
		}
		target := strings.TrimSpace(def.Target)
		if target == "" {
			out.Kind = QuickCommandAliasMissingTarget
			out.Evidence = fmt.Sprintf("quick command '/%s' has no alias target", current)
			return out
		}
		if !strings.HasPrefix(target, "/") {
			target = "/" + target
		}
		targetToken, targetArgs := splitCommandLine(target)
		targetName := normalizeCommandToken(targetToken)
		if targetName == "" {
			out.Kind = QuickCommandAliasUnsupportedTarget
			out.Evidence = fmt.Sprintf("quick command '/%s' has unsupported alias target %q", current, def.Target)
			return out
		}
		nextArgs := joinArgs(targetArgs, currentArgs)
		if _, ok := quick[targetName]; ok {
			current = targetName
			currentArgs = nextArgs
			continue
		}

		resolved := ResolveCommandAlias(joinSlashCommand(targetName, nextArgs))
		switch resolved.Kind {
		case CommandAliasExact, CommandAliasAlias, CommandAliasPrefix:
			out.Kind = QuickCommandAliasResolved
			out.Canonical = resolved.Canonical
			out.Rewrite = resolved.Rewrite
			return out
		case CommandAliasAmbiguous:
			out.Kind = QuickCommandAliasUnsupportedTarget
			out.Matches = resolved.Matches
			out.Evidence = fmt.Sprintf("quick command '/%s' target %q is ambiguous", current, def.Target)
			return out
		default:
			out.Kind = QuickCommandAliasUnsupportedTarget
			out.Evidence = fmt.Sprintf("quick command '/%s' target %q is not a recognized slash command", current, def.Target)
			return out
		}
	}
}

func splitCommandLine(input string) (token, args string) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", ""
	}
	for i, r := range trimmed {
		switch r {
		case ' ', '\t', '\n', '\r':
			return trimmed[:i], strings.TrimSpace(trimmed[i:])
		}
	}
	return trimmed, ""
}

func joinSlashCommand(command, args string) string {
	command = strings.TrimPrefix(strings.TrimSpace(command), "/")
	args = strings.TrimSpace(args)
	if command == "" {
		return ""
	}
	if args == "" {
		return "/" + command
	}
	return "/" + command + " " + args
}

func joinArgs(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, " ")
}
