package tui

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

// SlashCompletion is one entry in a slash-completion menu. It carries enough
// metadata for a downstream Bubble Tea menu binding to render canonical
// commands, their aliases, recognized-but-unavailable commands, and static
// subcommands without re-querying the registry. Helpers in this file are pure:
// no goroutines, no terminal IO, no provider/config/plugin filesystem
// dependency, and no shared mutable state across calls.
type SlashCompletion struct {
	// Name is the bare token a completer would insert (no leading slash for
	// command tokens; the literal subcommand for subcommand tokens).
	Name string
	// Display is the user-visible label. For commands it includes the
	// leading slash to mirror the Hermes prompt_toolkit drop-down; for
	// subcommands it equals Name.
	Display string
	// Description carries the registry's command description, or an empty
	// string for subcommands.
	Description string
	// Available reports whether the command's active-turn policy is
	// anything other than ActiveTurnPolicyUnavailable. Recognized-but-not-
	// yet-ported commands surface in completions with Available=false so the
	// menu can dim them while still letting users discover the inventory.
	Available bool
}

// HermesSlashCommandCompletions returns the slash-command completions a Hermes
// prompt_toolkit completer would surface for the given editor buffer text. The
// helper is pure, deterministic, and case-insensitive on the typed prefix.
//
// Behavior, mirroring hermes_cli/commands.py:SlashCommandCompleter:
//   - input that does not start with "/" returns nil (no completions).
//   - the leading "/" is stripped and the remaining text is matched as a
//     case-insensitive prefix against every canonical command name and alias
//     in cli.CommandRegistry.
//   - exact matches are still returned so the menu can stay open while the
//     user keeps editing.
//   - results are sorted alphabetically and de-duplicated, so two callers see
//     the same list every time and a freshly returned slice does not share a
//     backing array with any cached state.
//   - recognized-but-unavailable commands appear in the list with
//     Available=false; only outright unknown prefixes return nil.
func HermesSlashCommandCompletions(input string) []SlashCompletion {
	if !strings.HasPrefix(input, "/") {
		return nil
	}
	// Subcommand text is handled by HermesSlashSubcommandCompletions; the
	// command-prefix completer only fires on the bare command token.
	if strings.ContainsAny(input, " \t") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimPrefix(input, "/"))

	type hit struct {
		name  string
		entry cli.CommandPolicy
	}
	seen := map[string]hit{}
	for _, cmd := range cli.CommandRegistry {
		if strings.HasPrefix(cmd.Name, prefix) {
			seen[cmd.Name] = hit{name: cmd.Name, entry: cmd}
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, prefix) {
				seen[alias] = hit{name: alias, entry: cmd}
			}
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	out := make([]SlashCompletion, 0, len(names))
	for _, name := range names {
		h := seen[name]
		out = append(out, SlashCompletion{
			Name:        name,
			Display:     "/" + name,
			Description: h.entry.Description,
			Available:   h.entry.ActiveTurnPolicy != cli.ActiveTurnPolicyUnavailable,
		})
	}
	return out
}

// HermesSlashSubcommandCompletions returns static subcommand completions for
// inputs of the form "/cmd <prefix>" where the resolved command declares a
// non-empty Subcommands inventory in cli.CommandRegistry. Dynamic per-runtime
// menus (/model, /skin, /personality) are intentionally not surfaced here —
// they remain dependent rows that bind live config sources.
//
// Behavior, mirroring hermes_cli/commands.py:SlashCommandCompleter subcommand
// branch:
//   - returns nil when input lacks a leading "/", lacks any whitespace
//     (still typing the command name), or has whitespace inside the
//     sub-token (past the first sub-token boundary).
//   - returns nil when the command does not resolve or has no static
//     Subcommands.
//   - preserves the registry-defined Hermes order; it does NOT sort.
//   - prefix matching is case-insensitive; exact matches are still returned
//     so the dropdown can stay open like Hermes does.
func HermesSlashSubcommandCompletions(input string) []SlashCompletion {
	if !strings.HasPrefix(input, "/") {
		return nil
	}
	parts := strings.SplitN(input, " ", 2)
	if len(parts) != 2 {
		return nil
	}
	policy, ok := cli.ResolveCommandPolicy(parts[0])
	if !ok || len(policy.Subcommands) == 0 {
		return nil
	}
	subText := parts[1]
	// Past the first sub-token (i.e. another space inside the args) — the
	// static menu is no longer relevant.
	if strings.ContainsAny(subText, " \t") {
		return nil
	}
	prefix := strings.ToLower(subText)
	out := make([]SlashCompletion, 0, len(policy.Subcommands))
	for _, sub := range policy.Subcommands {
		if !strings.HasPrefix(sub, prefix) {
			continue
		}
		out = append(out, SlashCompletion{
			Name:      sub,
			Display:   sub,
			Available: true,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// HermesSlashAutoSuggest returns the inline ghost-text suffix Hermes'
// SlashCommandAutoSuggest would render for the given editor buffer text. The
// returned string is empty whenever no unique unambiguous completion exists:
// non-slash input, multiple matches, an already-complete name, or an unknown
// prefix.
//
// Behavior, mirroring hermes_cli/commands.py:SlashCommandAutoSuggest:
//   - for "/<word>" with no trailing space, returns the suffix of the unique
//     matching canonical command name (aliases participate in disambiguation
//     but the suggestion text is the alias's own tail when it is the unique
//     match — Hermes iterates COMMANDS keys in declaration order; we adopt
//     deterministic alphabetical order so two callers see the same answer).
//   - for "/<cmd> <word>" with the command resolved, returns the suffix of
//     the unique matching subcommand (Hermes order preserved by registry).
//   - returns "" for ambiguous, exact, or unrecognized input.
func HermesSlashAutoSuggest(input string) string {
	if !strings.HasPrefix(input, "/") {
		return ""
	}
	if !strings.ContainsAny(input, " \t") {
		word := strings.ToLower(strings.TrimPrefix(input, "/"))
		if word == "" {
			return ""
		}
		var unique string
		seen := map[string]struct{}{}
		for _, cmd := range cli.CommandRegistry {
			if strings.HasPrefix(cmd.Name, word) {
				seen[cmd.Name] = struct{}{}
			}
			for _, alias := range cmd.Aliases {
				if strings.HasPrefix(alias, word) {
					seen[alias] = struct{}{}
				}
			}
		}
		matches := make([]string, 0, len(seen))
		for n := range seen {
			matches = append(matches, n)
		}
		sort.Strings(matches)
		// Hermes only suggests when exactly one candidate strictly extends
		// the typed word.
		var extending []string
		for _, m := range matches {
			if m != word {
				extending = append(extending, m)
			}
		}
		if len(extending) != 1 {
			return ""
		}
		unique = extending[0]
		return unique[len(word):]
	}

	// "/cmd <subprefix>" branch.
	parts := strings.SplitN(input, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	policy, ok := cli.ResolveCommandPolicy(parts[0])
	if !ok || len(policy.Subcommands) == 0 {
		return ""
	}
	subText := parts[1]
	if strings.ContainsAny(subText, " \t") {
		return ""
	}
	subLower := strings.ToLower(subText)
	var extending []string
	for _, sub := range policy.Subcommands {
		if strings.HasPrefix(sub, subLower) && sub != subLower {
			extending = append(extending, sub)
		}
	}
	if len(extending) != 1 {
		return ""
	}
	return extending[0][len(subText):]
}
