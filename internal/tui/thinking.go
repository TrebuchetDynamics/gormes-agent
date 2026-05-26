// Package tui — Thinking display and ToolTrail renderers.
//
// These renderers are pure functions (state in → string out). They own no
// goroutines, never read time.Now(), and never store secret bytes. They are
// Go-native ports of upstream Hermes thinking.tsx components:
//
//   - Thinking          ↔ thinking.tsx:Thinking
//   - ToolTrail         ↔ thinking.tsx:ToolTrail
//   - Spinner           ↔ thinking.tsx:Spinner
//   - SubagentAccordion ↔ thinking.tsx:SubagentAccordion
package tui

import (
	"fmt"
	"strings"
	"time"
)

// ToolCallStatus represents the lifecycle state of a tool call.
type ToolCallStatus int

const (
	// ToolCallRunning indicates the tool is currently executing.
	ToolCallRunning ToolCallStatus = iota
	// ToolCallDone indicates the tool completed successfully.
	ToolCallDone
	// ToolCallError indicates the tool failed with an error.
	ToolCallError
)

// ToolCallNode represents a single tool call node in the tool tree.
// It supports nested children for subagent delegations.
type ToolCallNode struct {
	Name     string
	Status   ToolCallStatus
	Duration time.Duration
	Depth    int
	Children []ToolCallNode
}

// ThinkingState carries the data needed to render a thinking/reasoning block.
type ThinkingState struct {
	// Visible indicates whether the thinking block should be rendered.
	Visible bool
	// Content is the actual reasoning/thinking text.
	Content string
	// Truncated indicates the content exceeded the display threshold.
	Truncated bool
}

// TruncationThreshold is the maximum number of characters before content
// is marked as truncated in truncated mode.
const TruncationThreshold = 500

// RenderThinking renders a thinking/reasoning block with a 🤔 header when
// actively thinking and content with truncation indicator if over threshold.
// Mirrors thinking.tsx:Thinking component behavior.
func RenderThinking(state ThinkingState) string {
	return renderThinkingWithStyles(state, SkinStyles{}, false)
}

func RenderThinkingWithSkin(state ThinkingState, skin HermesSkin) string {
	return renderThinkingWithStyles(state, SkinStylesFor(skin), true)
}

func renderThinkingWithStyles(state ThinkingState, styles SkinStyles, styled bool) string {
	if !state.Visible {
		return ""
	}

	content := strings.TrimSpace(state.Content)
	if content == "" {
		// Empty thinking during active reasoning — show the thinking indicator
		// with a subtle cursor blink space to indicate live activity.
		return "  🤔 " + renderSkinStyle(styled, styles.Title, "Reasoning...")
	}

	// Apply truncation if indicated and content exceeds threshold.
	display := content
	if state.Truncated && len(content) > TruncationThreshold {
		display = content[:TruncationThreshold] + "…"
	}

	truncated := state.Truncated && len(content) > TruncationThreshold
	lines := strings.Split(display, "\n")
	if len(lines) > 1 {
		// Multi-line thinking: show header + content block.
		var b strings.Builder
		b.WriteString("  🤔 ")
		b.WriteString(renderSkinStyle(styled, styles.Title, "Reasoning"))
		if truncated {
			b.WriteString(" ")
			b.WriteString(renderSkinStyle(styled, styles.Dim, "[truncated]"))
		}
		b.WriteString("\n")
		for _, line := range lines {
			b.WriteString("  ")
			b.WriteString(renderSkinStyle(styled, styles.Text, line))
			b.WriteString("\n")
		}
		return b.String()
	}

	// Single-line thinking: header + inline content.
	result := "  🤔 " + renderSkinStyle(styled, styles.Title, "Reasoning")
	if truncated {
		result += " " + renderSkinStyle(styled, styles.Dim, "[truncated]")
	}
	result += "  " + renderSkinStyle(styled, styles.Text, content)
	return result
}

// treeRails builds the indentation prefix for a given depth.
// For depth 0, returns empty string.
// For depth > 0, returns "│   " for each level except the last.
// Uses the same tree rail pattern as thinking.tsx:treeLead.
func treeRails(depth int) string {
	if depth <= 0 {
		return ""
	}
	return strings.Repeat("│   ", depth-1)
}

// treeBranch returns the branch indicator for a node at given depth.
// "├─ " for non-last nodes, "└─ " for last nodes.
func treeBranch(isLast bool) string {
	if isLast {
		return "└─ "
	}
	return "├─ "
}

// toolIcon returns the emoji icon for a tool based on its name.
// Mirrors the icon mapping used in thinking.tsx for tool types.
func toolIcon(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "skill"):
		return "📚"
	case strings.Contains(name, "todo"):
		return "📋"
	case strings.Contains(name, "memory"):
		return "🧠"
	case strings.Contains(name, "search"):
		return "🔎"
	case strings.Contains(name, "read") || strings.Contains(name, "file"):
		return "📖"
	case strings.Contains(name, "write") || strings.Contains(name, "edit") || strings.Contains(name, "patch"):
		return "✏️"
	case strings.Contains(name, "bash") || strings.Contains(name, "shell") || strings.Contains(name, "exec"):
		return "⚡"
	case strings.Contains(name, "web") || strings.Contains(name, "http") || strings.Contains(name, "fetch"):
		return "🌐"
	case strings.Contains(name, "code") || strings.Contains(name, "programming"):
		return "💻"
	case strings.Contains(name, "delegate") || strings.Contains(name, "subagent") || strings.Contains(name, "spawn"):
		return "🔄"
	case strings.Contains(name, "database") || strings.Contains(name, "db"):
		return "🗄️"
	case strings.Contains(name, "api"):
		return "🔌"
	default:
		return "●"
	}
}

// statusIcon returns the emoji status indicator for a tool call status.
func statusIcon(status ToolCallStatus) string {
	switch status {
	case ToolCallRunning:
		return "⏳"
	case ToolCallDone:
		return "✅"
	case ToolCallError:
		return "❌"
	default:
		return "●"
	}
}

// formatDuration returns a human-readable duration string.
// Mirrors thinking.tsx:fmtElapsed behavior.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	sec := d.Seconds()
	if sec < 10 {
		return fmt.Sprintf("%.1fs", sec)
	}
	return fmt.Sprintf("%ds", int(sec))
}

// RenderToolTrail renders a tree of tool calls with tree rails, icons,
// status indicators, and duration display for completed tools.
// Mirrors thinking.tsx:ToolTrail component behavior.
func RenderToolTrail(nodes []ToolCallNode) string {
	return renderToolTrailWithStyles(nodes, SkinStyles{}, false)
}

func RenderToolTrailWithSkin(nodes []ToolCallNode, skin HermesSkin) string {
	return renderToolTrailWithStyles(nodes, SkinStylesFor(skin), true)
}

func renderToolTrailWithStyles(nodes []ToolCallNode, styles SkinStyles, styled bool) string {
	if len(nodes) == 0 {
		return ""
	}

	var b strings.Builder
	renderToolNodes(&b, nodes, true, styles, styled)
	return b.String()
}

// renderToolNodes recursively renders tool call nodes with proper tree formatting.
func renderToolNodes(b *strings.Builder, nodes []ToolCallNode, parentIsLast bool, styles SkinStyles, styled bool) {
	for i, node := range nodes {
		isNodeLast := i == len(nodes)-1

		// Build the line prefix with tree rails
		// If parent was not last, we need continuation rails at this depth
		prefix := treeRailsWithParent(node.Depth, parentIsLast) + treeBranch(isNodeLast)

		// Get icon and status
		icon := toolIcon(node.Name)
		status := statusIcon(node.Status)

		// Format duration if completed
		var durationStr string
		if node.Status == ToolCallDone && node.Duration > 0 {
			durationStr = " (" + formatDuration(node.Duration) + ")"
		}

		statusStyle := styles.Warn
		switch node.Status {
		case ToolCallDone:
			statusStyle = styles.Good
		case ToolCallError:
			statusStyle = styles.Bad
		}

		// Write the tool line
		b.WriteString(renderSkinStyle(styled, styles.Dim, prefix))
		b.WriteString(renderSkinStyle(styled, styles.Accent, icon))
		b.WriteString(" ")
		b.WriteString(renderSkinStyle(styled, statusStyle, status))
		b.WriteString(" ")
		b.WriteString(renderSkinStyle(styled, styles.Text, node.Name))
		if durationStr != "" {
			b.WriteString(renderSkinStyle(styled, styles.Dim, durationStr))
		}
		b.WriteString("\n")

		// Recursively render children
		if len(node.Children) > 0 {
			renderToolNodes(b, node.Children, isNodeLast, styles, styled)
		}
	}
}

// treeRailsWithParent builds tree rails considering whether the parent was last.
// If parent was last, no continuation rail is needed.
// If parent was not last, we need "│   " at the child's depth.
func treeRailsWithParent(depth int, parentIsLast bool) string {
	if depth <= 0 {
		return ""
	}
	if parentIsLast {
		return strings.Repeat("│   ", depth-1)
	}
	// Parent not last: add continuation rail at this depth
	return strings.Repeat("│   ", depth)
}

// SpinnerFrame represents a single frame of a braille spinner animation.
type SpinnerFrame struct {
	Chars string
}

// ThinkSpinnerFrames are the braille spinner frames for thinking mode.
// Based on unicode-animations helix/breathe patterns used in thinking.tsx.
var ThinkSpinnerFrames = []SpinnerFrame{
	SpinnerFrame{Chars: "⠋"},
	SpinnerFrame{Chars: "⠙"},
	SpinnerFrame{Chars: "⠹"},
	SpinnerFrame{Chars: "⠸"},
	SpinnerFrame{Chars: "⠼"},
	SpinnerFrame{Chars: "⠴"},
	SpinnerFrame{Chars: "⠦"},
	SpinnerFrame{Chars: "⠧"},
	SpinnerFrame{Chars: "⠇"},
	SpinnerFrame{Chars: "⠏"},
}

// ToolSpinnerFrames are the braille spinner frames for tool mode.
// Based on unicode-animations cascade/scan patterns used in thinking.tsx.
var ToolSpinnerFrames = []SpinnerFrame{
	SpinnerFrame{Chars: "⠁"},
	SpinnerFrame{Chars: "⠁"},
	SpinnerFrame{Chars: "⠉"},
	SpinnerFrame{Chars: "⠙"},
	SpinnerFrame{Chars: "⠚"},
	SpinnerFrame{Chars: "⠛"},
	SpinnerFrame{Chars: "⠜"},
	SpinnerFrame{Chars: "⠞"},
	SpinnerFrame{Chars: "⠟"},
	SpinnerFrame{Chars: "⠠"},
	SpinnerFrame{Chars: "⠤"},
	SpinnerFrame{Chars: "⠴"},
	SpinnerFrame{Chars: "⠦"},
	SpinnerFrame{Chars: "⠬"},
	SpinnerFrame{Chars: "⠰"},
	SpinnerFrame{Chars: "⠸"},
}

// RenderSpinner returns the unicode braille spinner for the given variant.
// Mirrors thinking.tsx:Spinner component behavior.
// variant should be "think" for reasoning spinners or "tool" for tool spinners.
func RenderSpinner(variant string) string {
	var frames []SpinnerFrame
	if variant == "tool" {
		frames = ToolSpinnerFrames
	} else {
		frames = ThinkSpinnerFrames
	}
	// Return the first frame as a static render
	// The caller should cycle through frames for animation
	if len(frames) > 0 {
		return frames[0].Chars
	}
	return "○"
}

// RenderSpinnerFrame returns a specific spinner frame by index.
// This allows callers to implement animation by cycling through frame indices.
func RenderSpinnerFrame(variant string, frameIndex int) string {
	var frames []SpinnerFrame
	if variant == "tool" {
		frames = ToolSpinnerFrames
	} else {
		frames = ThinkSpinnerFrames
	}
	if len(frames) == 0 {
		return "○"
	}
	// Modulo to cycle through frames
	idx := frameIndex % len(frames)
	return frames[idx].Chars
}
