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
	if !strings.HasPrefix(input, "/") {
		return nil
	}
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
	if !strings.HasPrefix(input, "/") || strings.ContainsAny(input, " \t") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimPrefix(input, "/"))
	var out []Completion
	for _, tmpl := range catalog.Templates {
		if !strings.HasPrefix(tmpl.Name, prefix) {
			continue
		}
		out = append(out, Completion{
			Name:         tmpl.Name,
			Display:      "/" + tmpl.Name,
			Description:  tmpl.Description,
			ArgumentHint: tmpl.ArgumentHint,
			Available:    true,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func SkillCompletions(input string, commands []skills.SkillSlashCommand) []Completion {
	if !strings.HasPrefix(input, "/") || strings.ContainsAny(input, " \t") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimPrefix(input, "/"))
	var out []Completion
	for _, command := range commands {
		name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(command.Command)), "/")
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		out = append(out, Completion{
			Name:        name,
			Display:     "/" + name,
			Description: command.Description,
			Available:   true,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func WithPromptTemplates(input string, catalog prompttemplates.Catalog) []Completion {
	return WithDynamic(input, nil, catalog)
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
	seen := make(map[string]struct{}, count)
	out := make([]Completion, 0, count)
	for _, group := range groups {
		for _, c := range group {
			if _, ok := seen[c.Name]; ok {
				continue
			}
			seen[c.Name] = struct{}{}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func SubcommandCompletions(input string) []Completion {
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
	if strings.ContainsAny(subText, " \t") {
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
	parts := strings.SplitN(input, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	policy, ok := cli.ResolveCommandPolicy(parts[0])
	if !ok || len(policy.Subcommands) == 0 {
		return "", false
	}
	if strings.ContainsAny(parts[1], " \t") {
		return "", false
	}
	return parts[0], true
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

	parts := strings.SplitN(input, " ", 2)
	if len(parts) != 2 || strings.ContainsAny(parts[1], " \t") {
		return ""
	}
	policy, ok := cli.ResolveCommandPolicy(parts[0])
	if !ok || len(policy.Subcommands) == 0 {
		return ""
	}
	prefix := strings.ToLower(parts[1])
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
