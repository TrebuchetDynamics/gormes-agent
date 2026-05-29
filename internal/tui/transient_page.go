package tui

import "github.com/TrebuchetDynamics/gormes-agent/internal/tui/transientpage"

// TransientPageState is a local, dismissible page rendered above the composer.
// It gives slash handlers a small interface for read-only transcript pages
// without coupling them to modal approval/secret state or the kernel.
type TransientPageState = transientpage.State

func RenderTransientPage(page TransientPageState, width, height int) string {
	return renderTransientPage(page, width, height, SkinStyles{}, false)
}

func RenderTransientPageWithSkin(page TransientPageState, width, height int, skin HermesSkin) string {
	return renderTransientPage(page, width, height, SkinStylesFor(skin), true)
}

func renderTransientPage(page TransientPageState, width, height int, styles SkinStyles, styled bool) string {
	return transientpage.Render(page, width, height, transientPageStyles(styles), styled, RenderMarkdownSoftWrapTrim)
}

func transientPageStyles(styles SkinStyles) transientpage.Styles {
	return transientpage.Styles{
		Title:     styles.Title,
		Dim:       styles.Dim,
		Separator: styles.Separator,
		Text:      styles.Text,
	}
}

func transientPageLines(body string, width int) []string {
	return transientpage.Lines(body, width, RenderMarkdownSoftWrapTrim)
}
