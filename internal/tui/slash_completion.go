package tui

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/extensibility/skills"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/prompttemplates"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/slashcompletion"
)

// SlashCompletion is one entry in a slash-completion menu. It carries enough
// metadata for a downstream Bubble Tea menu binding to render canonical
// commands, their aliases, recognized-but-unavailable commands, and static
// subcommands without re-querying the registry. Helpers in this file are pure:
// no goroutines, no terminal IO, no provider/config/plugin filesystem
// dependency, and no shared mutable state across calls.
type SlashCompletion = slashcompletion.Completion

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
	return slashcompletion.CommandCompletions(input)
}

// PromptTemplateSlashCompletions returns prompt-template command completions.
func PromptTemplateSlashCompletions(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.PromptTemplateCompletions(input, catalog)
}

// SkillSlashCompletions returns enabled dynamic skill slash completions.
func SkillSlashCompletions(input string, commands []skills.SkillSlashCommand) []SlashCompletion {
	return slashcompletion.SkillCompletions(input, commands)
}

// SlashCompletionsWithPromptTemplates merges built-in Hermes/Gormes slash
// completions with non-shadowing prompt-template completions.
func SlashCompletionsWithPromptTemplates(input string, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.WithPromptTemplates(input, catalog)
}

// SlashCompletionsWithDynamic merges built-in Hermes/Gormes slash completions,
// dynamic skill invocations, and prompt-template completions in precedence
// order. Later sources cannot shadow earlier ones.
func SlashCompletionsWithDynamic(input string, commands []skills.SkillSlashCommand, catalog prompttemplates.Catalog) []SlashCompletion {
	return slashcompletion.WithDynamic(input, commands, catalog)
}

// HermesSlashSubcommandCompletions returns static subcommand completions for
// inputs of the form "/cmd <prefix>" where the resolved command declares a
// non-empty Subcommands inventory in cli.CommandRegistry. Dynamic per-runtime
// menus (/model, /skin, /personality) are intentionally not surfaced here —
// they remain dependent rows that bind live config sources.
func HermesSlashSubcommandCompletions(input string) []SlashCompletion {
	return slashcompletion.SubcommandCompletions(input)
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
	return slashcompletion.AcceptedText(input, completion, trigger == slashCompletionAcceptTab)
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
	return slashcompletion.AutoSuggest(input)
}
