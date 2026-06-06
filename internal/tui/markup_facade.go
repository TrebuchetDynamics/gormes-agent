// Package tui — Markdown, diff, thinking, and spinner rendering facades.
package tui

import (
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/ansitext"
	diffview "github.com/TrebuchetDynamics/gormes-agent/internal/tui/diff"
	markdownrenderer "github.com/TrebuchetDynamics/gormes-agent/internal/tui/markdown"
	reasoning "github.com/TrebuchetDynamics/gormes-agent/internal/tui/reasoning"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tui/spinner"
)

// ─── Markdown rendering ─────────────────────────────────────────────────────

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

func RenderMarkdown(text string) string {
	return RenderMarkdownWithSkin(text, DefaultHermesSkin())
}

func RenderMarkdownWithSkin(text string, skin HermesSkin) string {
	return markdownrenderer.Render(text, markdownStylesFor(skin).rendererStyles())
}

func RenderMarkdownStable(text string, cache *string) string {
	return markdownrenderer.RenderStable(text, cache, markdownStylesFor(DefaultHermesSkin()).rendererStyles())
}

func RenderMarkdownSoftWrapTrim(text string, width int) string {
	return markdownrenderer.SoftWrapTrim(text, width)
}

// ─── Diff rendering ─────────────────────────────────────────────────────────

func RenderDiff(skin HermesSkin, diffText string, maxLines int) string {
	styles := SkinStylesFor(skin)
	return diffview.Render(diffview.Styles{
		Minus: styles.Bad,
		Plus:  styles.Good,
		Hunk:  styles.Separator,
		File:  styles.Label,
	}, diffText, maxLines)
}

// ─── Thinking/ToolTrail rendering ───────────────────────────────────────────

type ToolCallStatus = reasoning.ToolCallStatus

const (
	ToolCallRunning ToolCallStatus = reasoning.ToolCallRunning
	ToolCallDone    ToolCallStatus = reasoning.ToolCallDone
	ToolCallError   ToolCallStatus = reasoning.ToolCallError
)

type ToolCallNode = reasoning.ToolCallNode
type ThinkingState = reasoning.ThinkingState

const TruncationThreshold = reasoning.TruncationThreshold

func RenderThinking(state ThinkingState) string { return reasoning.RenderThinking(state) }

func RenderThinkingWithSkin(state ThinkingState, skin HermesSkin) string {
	return reasoning.RenderThinkingWithStyles(state, thinkingStylesForSkin(skin))
}

func RenderToolTrail(nodes []ToolCallNode) string { return reasoning.RenderToolTrail(nodes) }

func RenderToolTrailWithSkin(nodes []ToolCallNode, skin HermesSkin) string {
	return reasoning.RenderToolTrailWithStyles(nodes, thinkingStylesForSkin(skin))
}

type SpinnerFrame = reasoning.SpinnerFrame

var ThinkSpinnerFrames = reasoning.ThinkSpinnerFrames
var ToolSpinnerFrames = reasoning.ToolSpinnerFrames

func RenderSpinner(variant string) string { return reasoning.RenderSpinner(variant) }

func RenderSpinnerFrame(variant string, frameIndex int) string {
	return reasoning.RenderSpinnerFrame(variant, frameIndex)
}

func treeRails(depth int) string { return reasoning.TreeRails(depth) }
func treeBranch(isLast bool) string { return reasoning.TreeBranch(isLast) }
func toolIcon(name string) string { return reasoning.ToolIcon(name) }
func statusIcon(status ToolCallStatus) string { return reasoning.StatusIcon(status) }
func formatDuration(d time.Duration) string { return reasoning.FormatDuration(d) }

func thinkingStylesForSkin(skin HermesSkin) reasoning.Styles {
	styles := SkinStylesFor(skin)
	return reasoning.Styles{
		Accent: styles.Accent,
		Bad:    styles.Bad,
		Dim:    styles.Dim,
		Good:   styles.Good,
		Text:   styles.Text,
		Title:  styles.Title,
		Warn:   styles.Warn,
	}
}

// ─── Spinner rendering ──────────────────────────────────────────────────────

type SpinnerKind = spinner.Kind

const (
	SpinnerDots    SpinnerKind = spinner.Dots
	SpinnerBounce  SpinnerKind = spinner.Bounce
	SpinnerGrow    SpinnerKind = spinner.Grow
	SpinnerArrows  SpinnerKind = spinner.Arrows
	SpinnerStar    SpinnerKind = spinner.Star
	SpinnerMoon    SpinnerKind = spinner.Moon
	SpinnerPulse   SpinnerKind = spinner.Pulse
	SpinnerBrain   SpinnerKind = spinner.Brain
	SpinnerSparkle SpinnerKind = spinner.Sparkle
	SpinnerEmoji   SpinnerKind = spinner.Emoji
	SpinnerKaomoji SpinnerKind = spinner.Kaomoji
)

type SpinnerWing = spinner.Wing

func SpinnerFrames(kind SpinnerKind) []string { return spinner.Frames(kind) }
func WaitingFace(idx int) string { return spinner.WaitingFace(idx) }
func ThinkingFace(idx int) string { return spinner.ThinkingFace(idx) }
func ThinkingVerb(idx int) string { return spinner.ThinkingVerb(idx) }

func SpinnerWingsForSkin(skinName string) []SpinnerWing {
	return spinner.WingsForSkin(skinName)
}

func SpinnerRender(kind SpinnerKind, tick int, faceIdx int, verbIdx int, wingIdx int, skinName string, elapsed string) string {
	return spinner.Render(kind, tick, faceIdx, verbIdx, wingIdx, skinName, elapsed)
}

// ─── ANSI text helpers ──────────────────────────────────────────────────────

// StripANSIForTUI removes terminal control sequences before text enters
// cursor/source-of-truth calculations.
func StripANSIForTUI(s string) string {
	return ansitext.StripForTUI(s)
}

// SanitizeANSIForRender keeps SGR color sequences but strips cursor movement,
// OSC strings, malformed CSI, and C0 controls that can corrupt renderer state.
func SanitizeANSIForRender(s string) string {
	return ansitext.SanitizeForRender(s)
}

func HasANSI(s string) bool {
	return ansitext.HasANSI(s)
}

// trimToWidth trims text to fit within maxWidth using lipgloss width.
func trimToWidth(text string, maxWidth int) string {
	return ansitext.TrimToWidth(text, maxWidth)
}