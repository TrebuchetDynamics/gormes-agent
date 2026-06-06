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
	name      string
	entry     cli.CommandPolicy
	canonical bool
}

func commandCompletionCandidates(prefix completionPrefix) []Completion {
	plan := planCommandCompletionCandidates(prefix, cli.CommandRegistry)
	if plan.empty() {
		return nil
	}
	return renderCommandCompletionPlan(plan)
}

type commandCompletionPlan struct {
	Hits        map[string]commandCompletionHit
	SortedNames []string
}

func (p commandCompletionPlan) empty() bool {
	return len(p.SortedNames) == 0
}

func planCommandCompletionCandidates(prefix completionPrefix, registry []cli.CommandPolicy) commandCompletionPlan {
	hits := map[string]commandCompletionHit{}
	for _, cmd := range registry {
		addCommandCompletionHit(hits, prefix, cmd.Name, cmd, true)
		for _, alias := range cmd.Aliases {
			addCommandCompletionHit(hits, prefix, alias, cmd, false)
		}
	}
	return commandCompletionPlan{Hits: hits, SortedNames: sortedCompletionKeys(hits)}
}

func renderCommandCompletionPlan(plan commandCompletionPlan) []Completion {
	out := make([]Completion, 0, len(plan.SortedNames))
	for _, name := range plan.SortedNames {
		h := plan.Hits[name]
		out = append(out, Completion{
			Name:        name,
			Display:     "/" + name,
			Description: h.entry.Description,
			Available:   h.entry.ActiveTurnPolicy != cli.ActiveTurnPolicyUnavailable,
		})
	}
	return out
}

func addCommandCompletionHit(seen map[string]commandCompletionHit, prefix completionPrefix, rawName string, entry cli.CommandPolicy, canonical bool) {
	name := completionKey(rawName)
	if !prefix.matches(rawName) {
		return
	}
	if existing, ok := seen[name]; ok && (existing.canonical || !canonical) {
		return
	}
	seen[name] = commandCompletionHit{name: name, entry: entry, canonical: canonical}
}

func PromptTemplateCompletions(input string, catalog prompttemplates.Catalog) []Completion {
	req, ok := parseCompletionRequest(input)
	if !ok || !req.commandOnly() {
		return nil
	}
	var candidates []Completion
	for _, tmpl := range catalog.Templates {
		name := completionName(tmpl.Name)
		if name == "" || !completionNameMatches(name, req.commandPrefix) {
			continue
		}
		candidates = append(candidates, Completion{
			Name:         name,
			Display:      "/" + name,
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
		if name == "" || !req.commandPrefix.matches(command.Command) {
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
	flow, ok := resolveSubcommandFlow(input)
	if !ok {
		return nil
	}
	return matchingSubcommandCompletions(flow.Subcommands, flow.Prefix)
}

type subcommandCandidate struct {
	name string
	key  string
}

func matchingSubcommandCandidates(subcommands []string, prefix completionPrefix) []subcommandCandidate {
	if len(subcommands) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(subcommands))
	out := make([]subcommandCandidate, 0, len(subcommands))
	for _, sub := range subcommands {
		key := completionKey(sub)
		if key == "" || !prefix.matches(sub) {
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

func matchingSubcommandCompletions(subcommands []string, prefix completionPrefix) []Completion {
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
	word := commandAutoSuggestWord(input)
	plan := commandAutoSuggestPlanFor(word)
	if !plan.shouldExtend() {
		return ""
	}
	unique := plan.Extending[0]
	return unique[len(word):]
}

func commandAutoSuggestWord(input string) string {
	return newCompletionPrefix(input).string()
}

type commandAutoSuggestPlan struct {
	Word      string
	Exact     bool
	Extending []string
}

func (p commandAutoSuggestPlan) shouldExtend() bool {
	return p.Word != "" && !p.Exact && len(p.Extending) == 1
}

func commandAutoSuggestPlanFor(word string) commandAutoSuggestPlan {
	if word == "" {
		return commandAutoSuggestPlan{}
	}
	seen := map[string]struct{}{}
	for _, cmd := range cli.CommandRegistry {
		addAutoSuggestMatch(seen, word, cmd.Name)
		for _, alias := range cmd.Aliases {
			addAutoSuggestMatch(seen, word, alias)
		}
	}
	plan := commandAutoSuggestPlan{Word: word}
	for _, match := range sortedCompletionKeys(seen) {
		if match == word {
			plan.Exact = true
			continue
		}
		plan.Extending = append(plan.Extending, match)
	}
	return plan
}

func addAutoSuggestMatch(seen map[string]struct{}, word, rawName string) {
	name := completionKey(rawName)
	if strings.HasPrefix(name, word) {
		seen[name] = struct{}{}
	}
}

func singleSubcommandSuffix(input string) string {
	flow, ok := resolveSubcommandFlow(input)
	if !ok {
		return ""
	}
	return singleSubcommandCandidateSuffix(flow.Prefix, matchingSubcommandCandidates(flow.Subcommands, flow.Prefix))
}

func singleSubcommandCandidateSuffix(prefix completionPrefix, matches []subcommandCandidate) string {
	prefixText := prefix.string()
	var suffixes []string
	for _, match := range matches {
		suffix, ok := subcommandCandidateSuffix(prefixText, match.key)
		if ok {
			suffixes = append(suffixes, suffix)
		}
	}
	if len(suffixes) != 1 {
		return ""
	}
	return suffixes[0]
}

func subcommandCandidateSuffix(prefixText, candidateKey string) (string, bool) {
	if candidateKey == prefixText || !strings.HasPrefix(candidateKey, prefixText) {
		return "", false
	}
	return candidateKey[len(prefixText):], true
}
