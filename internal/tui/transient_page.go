package tui

import (
	"fmt"
	"strings"
)

// TransientPageState is a local, dismissible page rendered above the composer.
// It gives slash handlers a small interface for read-only transcript pages
// without coupling them to modal approval/secret state or the kernel.
type TransientPageState struct {
	Title  string
	Body   string
	Offset int
}

func RenderTransientPage(page TransientPageState, width, height int) string {
	return renderTransientPage(page, width, height, SkinStyles{}, false)
}

func RenderTransientPageWithSkin(page TransientPageState, width, height int, skin HermesSkin) string {
	return renderTransientPage(page, width, height, SkinStylesFor(skin), true)
}

func renderTransientPage(page TransientPageState, width, height int, styles SkinStyles, styled bool) string {
	title := strings.TrimSpace(page.Title)
	if title == "" {
		title = "Page"
	}
	bodyWidth := width - 4
	if bodyWidth < 20 {
		bodyWidth = 20
	}
	lines := transientPageLines(page.Body, bodyWidth)
	if page.Offset > 0 && page.Offset < len(lines) {
		lines = lines[page.Offset:]
	}
	if height > 0 && len(lines) > height {
		omitted := len(lines) - height
		if height <= 1 {
			lines = []string{fmt.Sprintf("... %d more lines ...", omitted+1)}
		} else {
			lines = append(lines[:height-1], fmt.Sprintf("... %d more lines ...", omitted+1))
		}
	}

	out := make([]string, 0, len(lines)+2)
	out = append(out, renderSkinStyle(styled, styles.Title, "╭─ "+title))
	if len(lines) == 0 {
		out = append(out, renderSkinStyle(styled, styles.Dim, "│ (empty)"))
	} else {
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				out = append(out, renderSkinStyle(styled, styles.Separator, "│"))
				continue
			}
			for _, wrapped := range strings.Split(RenderMarkdownSoftWrapTrim(line, bodyWidth), "\n") {
				out = append(out, renderSkinStyle(styled, styles.Separator, "│ ")+renderSkinStyle(styled, styles.Text, wrapped))
			}
		}
	}
	out = append(out, renderSkinStyle(styled, styles.Dim, "╰─ Esc to close"))
	return strings.Join(out, "\n")
}

func transientPageLines(body string, width int) []string {
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return nil
	}
	raw := strings.Split(body, "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if width > 0 {
			line = RenderMarkdownSoftWrapTrim(line, width)
		}
		parts := strings.Split(line, "\n")
		lines = append(lines, parts...)
	}
	return lines
}
