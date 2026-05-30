// Package tui preserves the public Thinking/ToolTrail renderer seam while the
// pure implementations live in internal/tui/reasoning.
package tui

import (
	"time"

	reasoning "github.com/TrebuchetDynamics/gormes-agent/internal/tui/reasoning"
)

// ToolCallStatus represents the lifecycle state of a tool call.
type ToolCallStatus = reasoning.ToolCallStatus

const (
	ToolCallRunning ToolCallStatus = reasoning.ToolCallRunning
	ToolCallDone    ToolCallStatus = reasoning.ToolCallDone
	ToolCallError   ToolCallStatus = reasoning.ToolCallError
)

// ToolCallNode represents a single tool call node in the tool tree.
type ToolCallNode = reasoning.ToolCallNode

// ThinkingState carries the data needed to render a thinking/reasoning block.
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

// SpinnerFrame represents a single frame of a braille spinner animation.
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
