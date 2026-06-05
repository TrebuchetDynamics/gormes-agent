package tui

import (
	"github.com/charmbracelet/lipgloss"

	markdownrenderer "github.com/TrebuchetDynamics/gormes-agent/internal/tui/markdown"
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

func (styles markdownStyles) rendererStyles() markdownrenderer.Styles {
	return markdownrenderer.Styles{
		CodeBlock:  styles.codeBlock,
		Code:       styles.code,
		Bold:       styles.bold,
		Italic:     styles.italic,
		Heading1:   styles.heading1,
		Heading2:   styles.heading2,
		Heading3:   styles.heading3,
		Blockquote: styles.blockquote,
		QuoteBar:   styles.quoteBar,
		List:       styles.list,
		Ordered:    styles.ordered,
		HR:         styles.hr,
		TableRule:  styles.tableRule,
		TableHead:  styles.tableHead,
		TableCell:  styles.tableCell,
	}
}

// RenderMarkdown renders markdown text as a string with lipgloss styling.
// It handles: fenced code blocks, inline bold/italic/code, headers,
// bullet/numbered lists, blockquotes, horizontal rules, and tables.
func RenderMarkdown(text string) string {
	return RenderMarkdownWithSkin(text, DefaultHermesSkin())
}

func RenderMarkdownWithSkin(text string, skin HermesSkin) string {
	return markdownrenderer.Render(text, markdownStylesFor(skin).rendererStyles())
}

func RenderMarkdownStable(text string, cache *string) string {
	return markdownrenderer.RenderStable(text, cache, markdownStylesFor(DefaultHermesSkin()).rendererStyles())
}

// RenderMarkdownSoftWrapTrim wraps prose using Hermes Ink's soft-boundary
// trimming rule: remove exactly one whitespace character introduced at each
// soft-wrap boundary while preserving source-line indentation and extra spaces.
func RenderMarkdownSoftWrapTrim(text string, width int) string {
	return markdownrenderer.SoftWrapTrim(text, width)
}
