package tui

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

type markdownStyles struct {
	codeBlock  lipgloss.Style
	code       lipgloss.Style
	bold       lipgloss.Style
	italic     lipgloss.Style
	heading1   lipgloss.Style
	heading2   lipgloss.Style
	heading3   lipgloss.Style
	blockquote lipgloss.Style
	quoteBar   lipgloss.Style
	list       lipgloss.Style
	ordered    lipgloss.Style
	hr         lipgloss.Style
	tableRule  lipgloss.Style
	tableHead  lipgloss.Style
	tableCell  lipgloss.Style
}

func markdownStylesFor(skin HermesSkin) markdownStyles {
	shared := SkinStylesFor(skin)
	return markdownStyles{
		codeBlock: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(shared.Separator.GetForeground()).
			Padding(1).
			Margin(1),
		code:       shared.Text.Background(shared.FocusLine.GetBackground()),
		bold:       shared.Text.Bold(true),
		italic:     shared.Text.Italic(true),
		heading1:   shared.Title.Underline(true),
		heading2:   shared.Title,
		heading3:   shared.Accent,
		blockquote: shared.Dim.MarginLeft(2),
		quoteBar:   shared.Separator.Bold(true),
		list:       lipgloss.NewStyle().MarginLeft(2),
		ordered:    lipgloss.NewStyle().MarginLeft(2),
		hr:         shared.Separator,
		tableRule:  shared.Separator,
		tableHead:  shared.Selected,
		tableCell:  shared.Text,
	}
}

// RenderMarkdown renders markdown text as a string with lipgloss styling.
// It handles: fenced code blocks, inline bold/italic/code, headers,
// bullet/numbered lists, blockquotes, horizontal rules, and tables.
func RenderMarkdown(text string) string {
	return RenderMarkdownWithSkin(text, DefaultHermesSkin())
}

func RenderMarkdownWithSkin(text string, skin HermesSkin) string {
	return renderMarkdownWithStyles(text, markdownStylesFor(skin))
}

func renderMarkdownWithStyles(text string, styles markdownStyles) string {
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	var output []string
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Fenced code blocks
		if matches := fencedCodeBlockRe.FindStringSubmatch(line); len(matches) > 0 {
			fence := matches[1]
			lang := strings.TrimSpace(matches[2])
			var codeLines []string
			i++
			for i < len(lines) {
				if fencedCodeCloseRe.MatchString(lines[i]) {
					break
				}
				codeLines = append(codeLines, lines[i])
				i++
			}
			i++ // skip closing fence
			output = append(output, renderCodeBlock(fence, lang, codeLines, styles))
			continue
		}

		// Horizontal rules - match --- *** ___ on their own lines
		if isHorizontalRule(line) {
			output = append(output, styles.hr.Render(strings.Repeat("─", 40)))
			i++
			continue
		}

		// Headers
		if matches := headingRe.FindStringSubmatch(line); len(matches) > 0 {
			level := len(matches[1])
			content := strings.TrimSpace(matches[2])
			output = append(output, renderHeading(level, content, styles))
			i++
			continue
		}

		// Blockquotes
		if quoteRe.MatchString(line) {
			var quoteLines []string
			for i < len(lines) && quoteRe.MatchString(lines[i]) {
				// Extract content after > prefix
				content := quoteRe.ReplaceAllString(lines[i], "")
				content = strings.TrimLeft(content, " ")
				quoteLines = append(quoteLines, content)
				i++
			}
			output = append(output, renderBlockquote(quoteLines, styles))
			continue
		}

		// Bullet lists
		if matches := bulletRe.FindStringSubmatch(line); len(matches) > 0 {
			var listLines []string
			for i < len(lines) {
				if m := bulletRe.FindStringSubmatch(lines[i]); len(m) > 0 {
					listLines = append(listLines, m[2])
					i++
				} else {
					break
				}
			}
			output = append(output, renderBulletList(listLines, styles))
			continue
		}

		// Numbered lists
		if matches := numberedDotRe.FindStringSubmatch(line); len(matches) > 0 {
			var listItems []string
			var separators []string
			for i < len(lines) {
				if m := numberedDotRe.FindStringSubmatch(lines[i]); len(m) > 0 {
					separators = append(separators, ".")
					listItems = append(listItems, m[2])
					i++
				} else if m := numberedParenRe.FindStringSubmatch(lines[i]); len(m) > 0 {
					separators = append(separators, ")")
					listItems = append(listItems, m[2])
					i++
				} else {
					break
				}
			}
			output = append(output, renderNumberedList(separators, listItems, styles))
			continue
		}

		// Tables
		if strings.Contains(line, "|") && i+1 < len(lines) && tableDividerRe.MatchString(strings.Trim(lines[i+1], " ")) {
			var rows [][]string
			// Header row
			rows = append(rows, splitTableRow(line))
			i++
			// Skip divider
			i++
			// Data rows
			for i < len(lines) && strings.Contains(lines[i], "|") {
				row := splitTableRow(lines[i])
				if len(row) > 0 {
					rows = append(rows, row)
				}
				i++
			}
			output = append(output, renderTable(rows, styles))
			continue
		}

		// Regular paragraph - apply inline formatting
		output = append(output, renderInline(line, styles))
		i++
	}

	return strings.Join(output, "\n")
}

func RenderMarkdownStable(text string, cache *string) string {
	if text == "" {
		*cache = ""
		return ""
	}
	if cache != nil && *cache != "" && strings.HasPrefix(text, *cache) {
		suffix := text[len(*cache):]
		if !strings.Contains(suffix, "\n\n") && !strings.Contains(suffix, "```") {
			return RenderMarkdown(text)
		}
	}
	boundary := findStableBoundary(text)
	var stable, unstable string
	if boundary > 0 && boundary < len(text) {
		stable = text[:boundary]
		unstable = text[boundary:]
	} else {
		unstable = text
	}
	if cache != nil && stable != "" {
		*cache = stable
	}
	if stable == "" {
		return RenderMarkdown(unstable)
	}
	rendered := RenderMarkdown(stable)
	if unstable != "" {
		rendered += "\n" + RenderMarkdown(unstable)
	}
	return rendered
}

func findStableBoundary(text string) int {
	lastNewline := strings.LastIndex(text, "\n\n")
	if lastNewline > 0 {
		return lastNewline + 2
	}
	lastFence := strings.LastIndex(text, "\n```")
	if lastFence > 0 {
		return lastFence
	}
	return -1
}

// RenderMarkdownSoftWrapTrim wraps prose using Hermes Ink's soft-boundary
// trimming rule: remove exactly one whitespace character introduced at each
// soft-wrap boundary while preserving source-line indentation and extra spaces.
func RenderMarkdownSoftWrapTrim(text string, width int) string {
	if text == "" || width <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = wrapLineSoftTrim(line, width)
	}
	return strings.Join(lines, "\n")
}

func wrapLineSoftTrim(line string, width int) string {
	if line == "" || lipgloss.Width(line) <= width {
		return line
	}

	rest := []rune(line)
	var wrapped []string
	for len(rest) > 0 && runesWidth(rest) > width {
		if breakAt := lastWhitespaceBreak(rest, width); breakAt > 0 {
			wrapped = append(wrapped, string(rest[:breakAt]))
			rest = rest[breakAt+1:]
			continue
		}

		end := runeCountForWidth(rest, width)
		if end <= 0 || end > len(rest) {
			end = 1
		}
		wrapped = append(wrapped, string(rest[:end]))
		rest = rest[end:]
		if len(rest) > 0 && unicode.IsSpace(rest[0]) {
			rest = rest[1:]
		}
	}
	wrapped = append(wrapped, string(rest))
	return strings.Join(wrapped, "\n")
}

func lastWhitespaceBreak(runes []rune, width int) int {
	used := 0
	last := -1
	for i, r := range runes {
		if unicode.IsSpace(r) && used <= width {
			last = i
		}
		next := used + lipgloss.Width(string(r))
		if next > width {
			break
		}
		used = next
	}
	return last
}

func runeCountForWidth(runes []rune, width int) int {
	used := 0
	for i, r := range runes {
		next := used + lipgloss.Width(string(r))
		if next > width {
			return i
		}
		used = next
	}
	return len(runes)
}

func runesWidth(runes []rune) int {
	return lipgloss.Width(string(runes))
}

// isHorizontalRule checks if a line is a horizontal rule.
func isHorizontalRule(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}
	return (strings.Count(line, "-") >= 3 && strings.Trim(line, "-") == "") ||
		(strings.Count(line, "*") >= 3 && strings.Trim(line, "*") == "") ||
		(strings.Count(line, "_") >= 3 && strings.Trim(line, "_") == "")
}

// Regular expressions for markdown parsing
var (
	fencedCodeBlockRe = regexp.MustCompile(`^\s*(` + "`" + `{3,}|~{3,})(.*)$`)
	fencedCodeCloseRe = regexp.MustCompile(`^\s*(` + "`" + `{3,}|~{3,})\s*$`)
	headingRe         = regexp.MustCompile(`^\s*(#{1,6})\s+(.*?)(?:\s+#+\s*)?$`)
	quoteRe           = regexp.MustCompile(`^\s*(>\s*)+`)
	bulletRe          = regexp.MustCompile(`^(\s*)[-+*]\s+(.*)$`)
	numberedDotRe     = regexp.MustCompile(`^\s*(\d+)\.\s+(.*)$`)
	numberedParenRe   = regexp.MustCompile(`^\s*(\d+)\)\s+(.*)$`)
	tableDividerRe    = regexp.MustCompile(`^:?[\s-]+:?$`)
	inlineCodeRe      = regexp.MustCompile("`([^`]+)`")
	boldRe            = regexp.MustCompile(`\*\*(.+?)\*\*`)
	italicRe          = regexp.MustCompile(`\*([^*]+)\*`)
	linkRe            = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
)

func renderCodeBlock(fence, lang string, lines []string, styles markdownStyles) string {
	var header string
	if lang != "" {
		header = styles.tableRule.Render("─") + " " + styles.tableHead.Render(lang)
	}
	code := strings.Join(lines, "\n")
	styled := styles.code.Render(code)

	if header != "" {
		return styles.codeBlock.Render(header + "\n" + styled)
	}
	return styles.codeBlock.Render(styled)
}

func renderHeading(level int, content string, styles markdownStyles) string {
	inline := renderInline(content, styles)
	switch level {
	case 1:
		return styles.heading1.Render(inline)
	case 2:
		return styles.heading2.Render(inline)
	default:
		return styles.heading3.Render(inline)
	}
}

func renderBlockquote(lines []string, styles markdownStyles) string {
	var output []string
	for _, line := range lines {
		output = append(output, styles.quoteBar.Render("│")+" "+renderInline(line, styles))
	}
	return styles.blockquote.Render(strings.Join(output, "\n"))
}

func renderBulletList(items []string, styles markdownStyles) string {
	var output []string
	for _, item := range items {
		output = append(output, styles.list.Render(styles.tableRule.Render("• ")+renderInline(item, styles)))
	}
	return strings.Join(output, "\n")
}

func renderNumberedList(separators []string, items []string, styles markdownStyles) string {
	var output []string
	for i, item := range items {
		sep := "."
		if i < len(separators) {
			sep = separators[i]
		}
		output = append(output, styles.ordered.Render(item+sep+" "+renderInline(item, styles)))
	}
	return strings.Join(output, "\n")
}

func renderTable(rows [][]string, styles markdownStyles) string {
	if len(rows) < 1 {
		return ""
	}

	// Calculate column widths
	colCount := len(rows[0])
	widths := make([]int, colCount)
	for _, row := range rows {
		for j := 0; j < len(row) && j < colCount; j++ {
			cellLen := len(stripMarkdown(row[j]))
			if cellLen > widths[j] {
				widths[j] = cellLen
			}
		}
	}

	var lines []string

	for ri, row := range rows {
		var cells []string
		for ci := 0; ci < colCount; ci++ {
			cell := ""
			if ci < len(row) {
				cell = row[ci]
			}
			stripped := stripMarkdown(cell)
			padding := widths[ci] - len(stripped)
			if padding > 0 {
				cell = cell + strings.Repeat(" ", padding)
			}

			if ri == 0 {
				// Header row
				cells = append(cells, styles.tableHead.Render(cell))
			} else {
				cells = append(cells, styles.tableCell.Render(cell))
			}
		}

		// Add separator after header
		if ri == 0 && len(rows) > 1 {
			sepCells := make([]string, colCount)
			for ci, w := range widths {
				sepCells[ci] = styles.tableRule.Render(strings.Repeat("─", w))
			}
			lines = append(lines, strings.Join(cells, "  "))
			lines = append(lines, strings.Join(sepCells, "  "))
		} else {
			lines = append(lines, strings.Join(cells, "  "))
		}
	}

	return strings.Join(lines, "\n")
}

func renderInline(text string, styles markdownStyles) string {
	if text == "" {
		return text
	}

	// Process inline code first (before other formatting)
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 2 {
			return styles.code.Render(match[1 : len(match)-1])
		}
		return match
	})

	// Process bold
	text = boldRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 4 {
			return styles.bold.Render(match[2 : len(match)-2])
		}
		return match
	})

	// Process italic - simple asterisk pattern
	text = italicRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 2 {
			return styles.italic.Render(match[1 : len(match)-1])
		}
		return match
	})

	return text
}

func splitTableRow(line string) []string {
	line = strings.Trim(line, " ")
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	var result []string
	for _, p := range parts {
		result = append(result, strings.TrimSpace(p))
	}
	return result
}

func stripMarkdown(text string) string {
	// Remove common markdown formatting for width calculation
	text = boldRe.ReplaceAllString(text, "$1")
	text = italicRe.ReplaceAllString(text, "$1")
	text = inlineCodeRe.ReplaceAllString(text, "$1")
	text = linkRe.ReplaceAllString(text, "$1")
	return text
}
