package slashcompletion

import (
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
	req, ok := parseCompletionRequest(input)
	if !ok || !req.commandOnly() {
		return nil
	}
	return commandCompletionCandidates(req.commandPrefix)
}

type commandCompletionHit struct {
	name  string
	entry cli.CommandPolicy
}

func commandCompletionCandidates(prefix string) []Completion {
	seen := map[string]commandCompletionHit{}
	for _, cmd := range cli.CommandRegistry {
		addCommandCompletionHit(seen, prefix, cmd.Name, cmd)
		for _, alias := range cmd.Aliases {
			addCommandCompletionHit(seen, prefix, alias, cmd)
		}
	}
	if len(seen) == 0 {
		return nil
	}
	names := sortedCompletionKeys(seen)
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

func addCommandCompletionHit(seen map[string]commandCompletionHit, prefix, rawName string, entry cli.CommandPolicy) {
	name := completionKey(rawName)
	if strings.HasPrefix(name, prefix) {
		seen[name] = commandCompletionHit{name: name, entry: entry}
	}
}

func PromptTemplateCompletions(input string, catalog prompttemplates.Catalog) []Completion {
	req, ok := parseCompletionRequest(input)
	if !ok || !req.commandOnly() {
		return nil
	}
	var candidates []Completion
	for _, tmpl := range catalog.Templates {
		if !completionNameMatches(tmpl.Name, req.commandPrefix) {
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
	req, ok := parseCompletionRequest(input)
	if !ok || !req.commandOnly() {
		return nil
	}
	var candidates []Completion
	for _, command := range commands {
		name := completionKey(command.Command)
		if name == "" || !strings.HasPrefix(name, req.commandPrefix) {
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

func WithDynamic(input string, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) []Completion {
	groups := [][]Completion{
		CommandCompletions(input),
		SkillCompletions(input, commands),
		PromptTemplateCompletions(input, catalog),
	}
	return uniqueSortedCompletions(flattenCompletionGroups(groups))
}

func SubcommandCompletions(input string) []Completion {
	req, ok := parseCompletionRequest(input)
	if !ok || !req.subcommandOnly() {
		return nil
	}
	policy, ok := cli.ResolveCommandPolicy(req.base)
	if !ok || len(policy.Subcommands) == 0 {
		return nil
	}
	return matchingSubcommandCompletions(policy.Subcommands, req.subPrefix)
}

type subcommandCandidate struct {
	name string
	key  string
}

func matchingSubcommandCandidates(subcommands []string, prefix string) []subcommandCandidate {
	if len(subcommands) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(subcommands))
	out := make([]subcommandCandidate, 0, len(subcommands))
	for _, sub := range subcommands {
		key := completionKey(sub)
		if key == "" || !strings.HasPrefix(key, prefix) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, subcommandCandidate{name: sub, key: key})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func matchingSubcommandCompletions(subcommands []string, prefix string) []Completion {
	candidates := matchingSubcommandCandidates(subcommands, prefix)
	if len(candidates) == 0 {
		return nil
	}
	out := make([]Completion, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, Completion{Name: candidate.name, Display: candidate.name, Available: true})
	}
	return out
}

func AutoSuggest(input string) string {
	if !strings.HasPrefix(input, "/") {
		return ""
	}
	if !strings.ContainsAny(input, " \t") {
		return singleCommandSuffix(input)
	}
	return singleSubcommandSuffix(input)
}

func singleCommandSuffix(input string) string {
	word := strings.ToLower(strings.TrimPrefix(input, "/"))
	if word == "" {
		return ""
	}
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		addAutoSuggestMatch(seen, word, cmd.Name)
		for _, alias := range cmd.Aliases {
			addAutoSuggestMatch(seen, word, alias)
		}
	}
	matches := sortedCompletionKeys(seen)
	var extending []string
	for _, match := range matches {
		if match != word {
			extending = append(extending, match)
		}
	}
	if len(extending) != 1 {
		return ""
	}
	unique := extending[0]
	return unique[len(word):]
}

func addAutoSuggestMatch(seen map[string]struct{}, word, rawName string) {
	name := completionKey(rawName)
	if strings.HasPrefix(name, word) {
		seen[name] = struct{}{}
	}
}

func singleSubcommandSuffix(input string) string {
	req, splitOK := parseCompletionRequest(input)
	if !splitOK || !req.subcommandOnly() {
		return ""
	}
	policy, ok := cli.ResolveCommandPolicy(req.base)
	if !ok || len(policy.Subcommands) == 0 {
		return ""
	}
	return singleSubcommandCandidateSuffix(req.subPrefix, matchingSubcommandCandidates(policy.Subcommands, req.subPrefix))
}

func singleSubcommandCandidateSuffix(prefix string, matches []subcommandCandidate) string {
	var extending []subcommandCandidate
	for _, match := range matches {
		if match.key != prefix {
			extending = append(extending, match)
		}
	}
	if len(extending) != 1 {
		return ""
	}
	return extending[0].key[len(prefix):]
}
