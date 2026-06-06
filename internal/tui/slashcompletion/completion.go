package slashcompletion

import (
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
)

// Completion is one entry in a slash-completion menu.
type Completion struct {
	Name         string
	Display      string
	Description  string
	ArgumentHint string
	Available    bool
}

func CommandCompletions(input string) []Completion {
	prefix, ok := slashCompletionPrefix(input)
	if !ok {
		return nil
	}
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
	out := make([]Completion, 0, len(names))
	for _, name := range names {
		h := seen[name]
		out = append(out, Completion{
			Name:        name,
			Display:     "/" + name,
			Description: h.entry.Description,
			Available:   h.entry.ActiveTurnPolicy != cli.ActiveTurnPolicyUnavailable,
		})
	}
	return out
}

func PromptTemplateCompletions(input string, catalog prompttemplates.Catalog) []Completion {
	prefix, ok := slashCompletionPrefix(input)
	if !ok {
		return nil
	}
	var candidates []Completion
	for _, tmpl := range catalog.Templates {
		if !completionNameMatches(tmpl.Name, prefix) {
			continue
		}
		candidates = append(candidates, Completion{
			Name:         tmpl.Name,
			Display:      "/" + tmpl.Name,
			Description:  tmpl.Description,
			ArgumentHint: tmpl.ArgumentHint,
			Available:    true,
		})
	}
	return uniqueSortedCompletions(candidates)
}

func SkillCompletions(input string, commands []skills.SkillSlashCommand) []Completion {
	prefix, ok := slashCompletionPrefix(input)
	if !ok {
		return nil
	}
	var candidates []Completion
	for _, command := range commands {
		name := completionKey(command.Command)
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		candidates = append(candidates, Completion{
			Name:        name,
			Display:     "/" + name,
			Description: command.Description,
			Available:   true,
		})
	}
	return uniqueSortedCompletions(candidates)
}

func WithPromptTemplates(input string, catalog prompttemplates.Catalog) []Completion {
	return WithDynamic(input, nil, catalog)
}

func slashCompletionPrefix(input string) (string, bool) {
	if !strings.HasPrefix(input, "/") || strings.ContainsAny(input, " \t") {
		return "", false
	}
	return strings.ToLower(strings.TrimPrefix(input, "/")), true
}

func completionNameMatches(name, prefix string) bool {
	return strings.HasPrefix(completionKey(name), prefix)
}

func WithDynamic(input string, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) []Completion {
	groups := [][]Completion{
		CommandCompletions(input),
		SkillCompletions(input, commands),
		PromptTemplateCompletions(input, catalog),
	}
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	if count == 0 {
		return nil
	}
	merged := make([]Completion, 0, count)
	for _, group := range groups {
		merged = append(merged, group...)
	}
	return uniqueSortedCompletions(merged)
}

func uniqueSortedCompletions(candidates []Completion) []Completion {
	if len(candidates) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]Completion, 0, len(candidates))
	for _, c := range candidates {
		key := completionKey(c.Name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return completionKey(out[i].Name) < completionKey(out[j].Name) })
	return out
}

func completionKey(name string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "/"))
}

func SubcommandCompletions(input string) []Completion {
	base, subText, ok := splitSubcommandInput(input)
	if !ok {
		return nil
	}
	policy, ok := cli.ResolveCommandPolicy(base)
	if !ok || len(policy.Subcommands) == 0 {
		return nil
	}
	prefix := strings.ToLower(subText)
	out := make([]Completion, 0, len(policy.Subcommands))
	for _, sub := range policy.Subcommands {
		if !strings.HasPrefix(sub, prefix) {
			continue
		}
		out = append(out, Completion{Name: sub, Display: sub, Available: true})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func AcceptedText(input string, completion Completion, acceptExact bool) (string, bool) {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return input, false
	}
	if base, ok := subcommandBase(input); ok {
		next := base + " " + name
		return next, next != input
	}

	next := "/" + name
	exact := strings.TrimSpace(input) == next
	if exact && !acceptExact {
		return input, false
	}
	if shouldAppendSpace(completion) && (acceptExact || !exact) {
		next += " "
	}
	return next, next != input
}

func subcommandBase(input string) (string, bool) {
	base, _, ok := splitSubcommandInput(input)
	if !ok {
		return "", false
	}
	policy, ok := cli.ResolveCommandPolicy(base)
	if !ok || len(policy.Subcommands) == 0 {
		return "", false
	}
	return base, true
}

func splitSubcommandInput(input string) (base string, subText string, ok bool) {
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}
	sep := strings.IndexFunc(input, func(r rune) bool { return r == ' ' || r == '\t' })
	if sep < 0 {
		return "", "", false
	}
	base = input[:sep]
	subText = strings.TrimLeft(input[sep+1:], " \t")
	if strings.ContainsAny(subText, " \t") {
		return "", "", false
	}
	return base, subText, true
}

func shouldAppendSpace(completion Completion) bool {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return false
	}
	if _, ok := noTrailingSpaceCommands[name]; ok {
		return false
	}
	if strings.TrimSpace(completion.ArgumentHint) != "" {
		return true
	}
	policy, ok := cli.ResolveCommandPolicy(name)
	if !ok {
		return false
	}
	if _, ok := noTrailingSpaceCommands[policy.Name]; ok {
		return false
	}
	if len(policy.Subcommands) > 0 {
		return true
	}
	_, ok = argumentCommandNames[policy.Name]
	return ok
}

var noTrailingSpaceCommands = map[string]struct{}{
	"model":       {},
	"personality": {},
	"provider":    {},
	"skin":        {},
}

var argumentCommandNames = map[string]struct{}{
	"approve":    {},
	"background": {},
	"branch":     {},
	"commands":   {},
	"compress":   {},
	"copy":       {},
	"cron":       {},
	"curator":    {},
	"goal":       {},
	"image":      {},
	"insights":   {},
	"new":        {},
	"platform":   {},
	"queue":      {},
	"quit":       {},
	"resume":     {},
	"rollback":   {},
	"sessions":   {},
	"snapshot":   {},
	"steer":      {},
	"subgoal":    {},
	"title":      {},
	"topic":      {},
	"tools":      {},
}

func AutoSuggest(input string) string {
	if !strings.HasPrefix(input, "/") {
		return ""
	}
	if !strings.ContainsAny(input, " \t") {
		word := strings.ToLower(strings.TrimPrefix(input, "/"))
		if word == "" {
			return ""
		}
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
		var extending []string
		for _, m := range matches {
			if m != word {
				extending = append(extending, m)
			}
		}
		if len(extending) != 1 {
			return ""
		}
		unique := extending[0]
		return unique[len(word):]
	}

	base, subText, splitOK := splitSubcommandInput(input)
	if !splitOK {
		return ""
	}
	policy, ok := cli.ResolveCommandPolicy(base)
	if !ok || len(policy.Subcommands) == 0 {
		return ""
	}
	prefix := strings.ToLower(subText)
	var matches []string
	for _, sub := range policy.Subcommands {
		if strings.HasPrefix(sub, prefix) && sub != prefix {
			matches = append(matches, sub)
		}
	}
	if len(matches) != 1 {
		return ""
	}
	return matches[0][len(prefix):]
}
