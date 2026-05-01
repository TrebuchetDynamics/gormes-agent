package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Markdown rendering styles
var (
	codeBlockStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1).
			Margin(1)

	codeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("250"))

	boldStyle = lipgloss.NewStyle().Bold(true)

	italicStyle = lipgloss.NewStyle().Italic(true)

	headingStyle1 = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Foreground(lipgloss.Color("228"))

	headingStyle2 = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("228"))

	headingStyle3 = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("228"))

	blockquoteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("242")).
			MarginLeft(2)

	quoteBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Bold(true)

	listStyle = lipgloss.NewStyle().
			MarginLeft(2)

	orderedListStyle = lipgloss.NewStyle().
			MarginLeft(2)

	hrStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	tableBorderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	tableHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("75"))

	tableCellStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// RenderMarkdown renders markdown text as a string with lipgloss styling.
// It handles: fenced code blocks, inline bold/italic/code, headers,
// bullet/numbered lists, blockquotes, horizontal rules, and tables.
func RenderMarkdown(text string) string {
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
			output = append(output, renderCodeBlock(fence, lang, codeLines))
			continue
		}

		// Horizontal rules - match --- *** ___ on their own lines
		if isHorizontalRule(line) {
			output = append(output, hrStyle.Render(strings.Repeat("─", 40)))
			i++
			continue
		}

		// Headers
		if matches := headingRe.FindStringSubmatch(line); len(matches) > 0 {
			level := len(matches[1])
			content := strings.TrimSpace(matches[2])
			output = append(output, renderHeading(level, content))
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
			output = append(output, renderBlockquote(quoteLines))
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
			output = append(output, renderBulletList(listLines))
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
			output = append(output, renderNumberedList(separators, listItems))
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
			output = append(output, renderTable(rows))
			continue
		}

		// Regular paragraph - apply inline formatting
		output = append(output, renderInline(line))
		i++
	}

	return strings.Join(output, "\n")
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

func renderCodeBlock(fence, lang string, lines []string) string {
	var header string
	if lang != "" {
		header = "─ " + lang
	}
	code := strings.Join(lines, "\n")
	styled := codeStyle.Render(code)

	if header != "" {
		return codeBlockStyle.Render(header + "\n" + styled)
	}
	return codeBlockStyle.Render(styled)
}

func renderHeading(level int, content string) string {
	inline := renderInline(content)
	switch level {
	case 1:
		return headingStyle1.Render(inline)
	case 2:
		return headingStyle2.Render(inline)
	default:
		return headingStyle3.Render(inline)
	}
}

func renderBlockquote(lines []string) string {
	var output []string
	for _, line := range lines {
		output = append(output, quoteBarStyle.Render("│")+" "+renderInline(line))
	}
	return blockquoteStyle.Render(strings.Join(output, "\n"))
}

func renderBulletList(items []string) string {
	var output []string
	for _, item := range items {
		output = append(output, listStyle.Render("• "+renderInline(item)))
	}
	return strings.Join(output, "\n")
}

func renderNumberedList(separators []string, items []string) string {
	var output []string
	for i, item := range items {
		sep := "."
		if i < len(separators) {
			sep = separators[i]
		}
		output = append(output, orderedListStyle.Render(item+sep+" "+renderInline(item)))
	}
	return strings.Join(output, "\n")
}

func renderTable(rows [][]string) string {
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
				cells = append(cells, tableHeaderStyle.Render(cell))
			} else {
				cells = append(cells, tableCellStyle.Render(cell))
			}
		}

		// Add separator after header
		if ri == 0 && len(rows) > 1 {
			sepCells := make([]string, colCount)
			for ci, w := range widths {
				sepCells[ci] = tableBorderStyle.Render(strings.Repeat("─", w))
			}
			lines = append(lines, strings.Join(cells, "  "))
			lines = append(lines, strings.Join(sepCells, "  "))
		} else {
			lines = append(lines, strings.Join(cells, "  "))
		}
	}

	return strings.Join(lines, "\n")
}

func renderInline(text string) string {
	if text == "" {
		return text
	}

	// Process inline code first (before other formatting)
	text = inlineCodeRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 2 {
			return codeStyle.Render(match[1 : len(match)-1])
		}
		return match
	})

	// Process bold
	text = boldRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 4 {
			return boldStyle.Render(match[2 : len(match)-2])
		}
		return match
	})

	// Process italic - simple asterisk pattern
	text = italicRe.ReplaceAllStringFunc(text, func(match string) string {
		if len(match) >= 2 {
			return italicStyle.Render(match[1 : len(match)-1])
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
