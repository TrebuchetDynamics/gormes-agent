package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
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
	// ArgumentHint carries optional template argument guidance displayed next
	// to the slash command name, e.g. `/review <scope>`.
	ArgumentHint string
	// Available reports whether the command's active-turn policy is
	// anything other than ActiveTurnPolicyUnavailable. Recognized-but-not-
	// yet-ported commands surface in completions with Available=false so the
	// menu can dim them while still letting users discover the inventory.
	Available bool
}

type slashCompletionState struct {
	key          string
	index        int
	dismissedFor string
}

type slashCompletionMenu struct {
	request     TUICompletionRequest
	completions []SlashCompletion
	total       int
	key         string
}

type slashCompletionAcceptTrigger int

const (
	slashCompletionAcceptEnter slashCompletionAcceptTrigger = iota
	slashCompletionAcceptTab
)

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

// PromptTemplateSlashCompletions returns prompt-template command completions.
func PromptTemplateSlashCompletions(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	if !strings.HasPrefix(input, "/") || strings.ContainsAny(input, " \t") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimPrefix(input, "/"))
	var out []SlashCompletion
	for _, tmpl := range catalog.Templates {
		if !strings.HasPrefix(tmpl.Name, prefix) {
			continue
		}
		out = append(out, SlashCompletion{
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

// SkillSlashCompletions returns enabled dynamic skill slash completions.
func SkillSlashCompletions(input string, commands []skills.SkillSlashCommand) []SlashCompletion {
	if !strings.HasPrefix(input, "/") || strings.ContainsAny(input, " \t") {
		return nil
	}
	prefix := strings.ToLower(strings.TrimPrefix(input, "/"))
	var out []SlashCompletion
	for _, command := range commands {
		name := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(command.Command)), "/")
		if name == "" || !strings.HasPrefix(name, prefix) {
			continue
		}
		out = append(out, SlashCompletion{
			Name:        name,
			Display:     "/" + name,
			Description: command.Description,
			Available:   true,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SlashCompletionsWithPromptTemplates merges built-in Hermes/Gormes slash
// completions with non-shadowing prompt-template completions.
func SlashCompletionsWithPromptTemplates(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	return SlashCompletionsWithDynamic(input, nil, catalog)
}

// SlashCompletionsWithDynamic merges built-in Hermes/Gormes slash completions,
// dynamic skill invocations, and prompt-template completions in precedence
// order. Later sources cannot shadow earlier ones.
func SlashCompletionsWithDynamic(input string, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) []SlashCompletion {
	groups := [][]SlashCompletion{
		HermesSlashCommandCompletions(input),
		SkillSlashCompletions(input, commands),
		PromptTemplateSlashCompletions(input, catalog),
	}
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	if count == 0 {
		return nil
	}
	seen := make(map[string]struct{}, count)
	out := make([]SlashCompletion, 0, count)
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

// HermesSlashSubcommandCompletions returns static subcommand completions for
// inputs of the form "/cmd <prefix>" where the resolved command declares a
// non-empty Subcommands inventory in cli.CommandRegistry. Dynamic per-runtime
// menus (/model, /skin, /personality) are intentionally not surfaced here —
// they remain dependent rows that bind live config sources.
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

func (m Model) renderActiveSlashCompletionMenu(input string) string {
	if m.slashCompletion.dismissedFor == input {
		return ""
	}
	menu, ok := slashCompletionMenuForInput(input, m.width, m.skillSlashCommands, m.promptTemplates)
	if !ok {
		return ""
	}
	selected := 0
	if m.slashCompletion.key == menu.key {
		selected = clampSlashCompletionIndex(m.slashCompletion.index, len(menu.completions))
	}
	return renderSlashCompletionMenuWithDynamicSelected(input, m.width, m.currentSkin(), m.skillSlashCommands, m.promptTemplates, selected)
}

func (m *Model) activeSlashCompletionMenu() (slashCompletionMenu, bool) {
	input := m.editor.Value()
	if m.slashCompletion.dismissedFor == input {
		return slashCompletionMenu{}, false
	}
	return slashCompletionMenuForInput(input, m.width, m.skillSlashCommands, m.promptTemplates)
}

func (m *Model) ensureSlashCompletionSelection(menu slashCompletionMenu) int {
	if m.slashCompletion.key != menu.key {
		m.slashCompletion.key = menu.key
		m.slashCompletion.index = 0
	}
	m.slashCompletion.index = clampSlashCompletionIndex(m.slashCompletion.index, len(menu.completions))
	return m.slashCompletion.index
}

func (m *Model) resetSlashCompletionDismissalForInput(input string) {
	if m.slashCompletion.dismissedFor != "" && m.slashCompletion.dismissedFor != input {
		m.slashCompletion.dismissedFor = ""
	}
}

func renderSlashCompletionMenu(input string, width int) string {
	return renderSlashCompletionMenuWithSkin(input, width, DefaultHermesSkin())
}

func renderSlashCompletionMenuWithSkin(input string, width int, skin HermesSkin) string {
	return renderSlashCompletionMenuWithTemplates(input, width, skin, prompttemplates.Catalog{})
}

func renderSlashCompletionMenuWithTemplates(input string, width int, skin HermesSkin, catalog prompttemplates.Catalog) string {
	return renderSlashCompletionMenuWithDynamic(input, width, skin, nil, catalog)
}

func renderSlashCompletionMenuWithDynamic(input string, width int, skin HermesSkin, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) string {
	return renderSlashCompletionMenuWithDynamicSelected(input, width, skin, commands, catalog, 0)
}

func renderSlashCompletionMenuWithDynamicSelected(input string, width int, skin HermesSkin, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog, selected int) string {
	menu, ok := slashCompletionMenuForInput(input, width, commands, catalog)
	if !ok {
		return ""
	}
	selected = clampSlashCompletionIndex(selected, len(menu.completions))
	styles := SkinStylesFor(skin)
	bodyWidth := width - 4
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	nameW := 0
	for _, c := range menu.completions {
		display := slashCompletionDisplay(c)
		if w := len([]rune(display)); w > nameW {
			nameW = w
		}
	}
	if nameW > 24 {
		nameW = 24
	}
	if nameW < 8 {
		nameW = 8
	}
	descW := bodyWidth - nameW - 5
	if descW < 8 {
		descW = 8
	}
	lines := make([]string, 0, len(menu.completions)+3)
	query := strings.TrimSpace(menu.request.Text)
	if query == "" {
		query = input
	}
	lines = append(lines, styles.Accent.Render(truncateEllipsis("╭─ Search "+query, bodyWidth)))
	for idx, c := range menu.completions {
		marker := "  "
		rowStyle := styles.Normal
		if idx == selected {
			marker = "❯ "
			rowStyle = styles.Selected
		}
		if !c.Available {
			rowStyle = styles.Dim
		}
		displayText := padRightRunes(truncateEllipsis(slashCompletionDisplay(c), nameW), nameW)
		desc := strings.TrimSpace(c.Description)
		if !c.Available {
			if desc != "" {
				desc = "⚡ " + desc
			} else {
				desc = "⚡ recognized, unavailable"
			}
		}
		line := "│ " + marker + rowStyle.Render(displayText)
		if desc != "" {
			line += "  " + styles.Dim.Render(truncateEllipsis(desc, descW))
		}
		lines = append(lines, line)
	}
	if extra := menu.total - len(menu.completions); extra > 0 {
		lines = append(lines, styles.Dim.Render(fmt.Sprintf("│ … +%d more matches", extra)))
	}
	footer := "╰─ ↑/↓ select · Enter complete · Esc close"
	lines = append(lines, styles.Dim.Render(truncateEllipsis(footer, bodyWidth)))
	return strings.Join(lines, "\n")
}

func slashCompletionMenuForInput(input string, width int, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) (slashCompletionMenu, bool) {
	req, ok := CompletionRequestForInput(input)
	if !ok || req.Method != TUICompletionSlash {
		return slashCompletionMenu{}, false
	}
	completions := HermesSlashSubcommandCompletions(input)
	if len(completions) == 0 {
		completions = SlashCompletionsWithDynamic(input, commands, catalog)
	}
	if len(completions) == 0 {
		return slashCompletionMenu{}, false
	}
	limit := len(completions)
	if visible := slashCompletionVisibleLimit(width); limit > visible {
		limit = visible
	}
	visible := append([]SlashCompletion(nil), completions[:limit]...)
	return slashCompletionMenu{
		request:     req,
		completions: visible,
		total:       len(completions),
		key:         slashCompletionCandidateKey(req, visible),
	}, true
}

func slashCompletionCandidateKey(req TUICompletionRequest, completions []SlashCompletion) string {
	var b strings.Builder
	b.WriteString(string(req.Method))
	b.WriteByte('|')
	for _, c := range completions {
		b.WriteString(c.Name)
		b.WriteByte('\x00')
		b.WriteString(c.Display)
		b.WriteByte('\x00')
		b.WriteString(c.ArgumentHint)
		b.WriteByte('\x00')
		if c.Available {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
		b.WriteByte('|')
	}
	return b.String()
}

func slashCompletionVisibleLimit(width int) int {
	switch {
	case width < 40:
		return 3
	case width < 64:
		return 5
	default:
		return 8
	}
}

func clampSlashCompletionIndex(index, count int) int {
	if count <= 0 || index < 0 {
		return 0
	}
	if index >= count {
		return count - 1
	}
	return index
}

func wrapSlashCompletionIndex(index, delta, count int) int {
	if count <= 0 {
		return 0
	}
	return (index + delta + count) % count
}

func slashCompletionAcceptedText(input string, completion SlashCompletion, trigger slashCompletionAcceptTrigger) (string, bool) {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return input, false
	}
	if base, ok := slashCompletionSubcommandBase(input); ok {
		next := base + " " + name
		return next, next != input
	}

	next := "/" + name
	exact := strings.TrimSpace(input) == next
	if trigger == slashCompletionAcceptEnter && exact {
		return input, false
	}
	if slashCompletionShouldAppendSpace(completion) && (trigger == slashCompletionAcceptTab || !exact) {
		next += " "
	}
	return next, next != input
}

func slashCompletionSubcommandBase(input string) (string, bool) {
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

func slashCompletionShouldAppendSpace(completion SlashCompletion) bool {
	name := strings.TrimSpace(strings.TrimPrefix(completion.Name, "/"))
	if name == "" {
		return false
	}
	if _, ok := slashCompletionNoTrailingSpaceCommands[name]; ok {
		return false
	}
	if strings.TrimSpace(completion.ArgumentHint) != "" {
		return true
	}
	policy, ok := cli.ResolveCommandPolicy(name)
	if !ok {
		return false
	}
	if _, ok := slashCompletionNoTrailingSpaceCommands[policy.Name]; ok {
		return false
	}
	if len(policy.Subcommands) > 0 {
		return true
	}
	_, ok = slashCompletionArgumentCommandNames[policy.Name]
	return ok
}

var slashCompletionNoTrailingSpaceCommands = map[string]struct{}{
	"model":       {},
	"personality": {},
	"provider":    {},
	"skin":        {},
}

var slashCompletionArgumentCommandNames = map[string]struct{}{
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

func slashCompletionDisplay(c SlashCompletion) string {
	display := strings.TrimSpace(c.Display)
	if display == "" {
		if strings.HasPrefix(c.Name, "/") {
			display = c.Name
		} else {
			display = c.Name
		}
	}
	if hint := strings.TrimSpace(c.ArgumentHint); hint != "" {
		return display + " " + hint
	}
	return display
}

func padRightRunes(value string, width int) string {
	if width <= 0 {
		return ""
	}
	for len([]rune(value)) < width {
		value += " "
	}
	return value
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
