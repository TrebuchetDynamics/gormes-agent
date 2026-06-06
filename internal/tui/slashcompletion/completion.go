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
	return commandCompletionCandidates(prefix)
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

func sortedCompletionKeys[T any](m map[string]T) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

type parsedSlashInput struct {
	commandPrefix string
	base          string
	subPrefix     string
	hasSubcommand bool
}

func slashCompletionPrefix(input string) (string, bool) {
	parsed, ok := parseSlashInput(input)
	if !ok || parsed.hasSubcommand {
		return "", false
	}
	return parsed.commandPrefix, true
}

func parseSlashInput(input string) (parsedSlashInput, bool) {
	if !strings.HasPrefix(input, "/") {
		return parsedSlashInput{}, false
	}
	sep := strings.IndexFunc(input, func(r rune) bool { return r == ' ' || r == '\t' })
	if sep < 0 {
		return parsedSlashInput{commandPrefix: completionKey(input)}, true
	}
	base := completionKey(input[:sep])
	if base == "" {
		return parsedSlashInput{}, false
	}
	subText := strings.TrimLeft(input[sep+1:], " \t")
	if strings.ContainsAny(subText, " \t") {
		return parsedSlashInput{}, false
	}
	return parsedSlashInput{
		base:          "/" + base,
		subPrefix:     strings.ToLower(subText),
		hasSubcommand: true,
	}, true
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
	return uniqueSortedCompletions(flattenCompletionGroups(groups))
}

func flattenCompletionGroups(groups [][]Completion) []Completion {
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
	return merged
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
	parsed, ok := parseSlashInput(input)
	if !ok || !parsed.hasSubcommand {
		return nil
	}
	policy, ok := cli.ResolveCommandPolicy(parsed.base)
	if !ok || len(policy.Subcommands) == 0 {
		return nil
	}
	out := make([]Completion, 0, len(policy.Subcommands))
	for _, sub := range policy.Subcommands {
		if !strings.HasPrefix(completionKey(sub), parsed.subPrefix) {
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
	parsed, ok := parseSlashInput(input)
	if !ok || !parsed.hasSubcommand {
		return "", "", false
	}
	return parsed.base, parsed.subPrefix, true
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
	parsed, splitOK := parseSlashInput(input)
	if !splitOK || !parsed.hasSubcommand {
		return ""
	}
	policy, ok := cli.ResolveCommandPolicy(parsed.base)
	if !ok || len(policy.Subcommands) == 0 {
		return ""
	}
	var matches []string
	for _, sub := range policy.Subcommands {
		key := completionKey(sub)
		if strings.HasPrefix(key, parsed.subPrefix) && key != parsed.subPrefix {
			matches = append(matches, sub)
		}
	}
	if len(matches) != 1 {
		return ""
	}
	return matches[0][len(parsed.subPrefix):]
}
