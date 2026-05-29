package tui

import diffview "github.com/TrebuchetDynamics/gormes-agent/internal/tui/diff"

func RenderDiff(skin HermesSkin, diffText string, maxLines int) string {
	styles := SkinStylesFor(skin)
	return diffview.Render(diffview.Styles{
		Minus: styles.Bad,
		Plus:  styles.Good,
		Hunk:  styles.Separator,
		File:  styles.Label,
	}, diffText, maxLines)
}
