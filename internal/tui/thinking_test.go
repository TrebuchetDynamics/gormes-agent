package tui

import (
	"strings"
	"testing"
	"time"
)

// TestThinking_RenderEmpty verifies that RenderThinking returns empty for invisible state.
func TestThinking_RenderEmpty(t *testing.T) {
	state := ThinkingState{
		Visible:   false,
		Content:   "some thinking",
		Truncated: false,
	}
	got := RenderThinking(state)
	if got != "" {
		t.Fatalf("RenderThinking(Visible=false) = %q, want empty string", got)
	}
}

// TestThinking_RenderEmptyContent verifies that empty content shows "Reasoning..." indicator.
func TestThinking_RenderEmptyContent(t *testing.T) {
	state := ThinkingState{
		Visible:   true,
		Content:   "",
		Truncated: false,
	}
	got := RenderThinking(state)
	if !strings.Contains(got, "🤔") {
		t.Fatalf("RenderThinking empty content missing 🤔 icon in %q", got)
	}
	if !strings.Contains(got, "Reasoning...") {
		t.Fatalf("RenderThinking empty content missing Thinking... in %q", got)
	}
}

// TestThinking_RenderWithContent verifies thinking block renders with content.
func TestThinking_RenderWithContent(t *testing.T) {
	state := ThinkingState{
		Visible:   true,
		Content:   "Let me think about this carefully",
		Truncated: false,
	}
	got := RenderThinking(state)
	if !strings.Contains(got, "🤔") {
		t.Fatalf("RenderThinking missing 🤔 icon in %q", got)
	}
	if !strings.Contains(got, "Reasoning") {
		t.Fatalf("RenderThinking missing Thinking label in %q", got)
	}
	if !strings.Contains(got, "Let me think about this carefully") {
		t.Fatalf("RenderThinking missing content in %q", got)
	}
}

// TestThinking_TruncationIndicator verifies truncation indicator appears when marked truncated.
func TestThinking_TruncationIndicator(t *testing.T) {
	// Create content that exceeds the truncation threshold
	longContent := strings.Repeat("x", TruncationThreshold+100)
	state := ThinkingState{
		Visible:   true,
		Content:   longContent,
		Truncated: true,
	}
	got := RenderThinking(state)
	if !strings.Contains(got, "[truncated]") {
		t.Fatalf("RenderThinking with Truncated=true missing [truncated] indicator in %q", got)
	}
}

// TestThinking_NoTruncationWhenNotMarked verifies no truncation when Truncated is false.
func TestThinking_NoTruncationWhenNotMarked(t *testing.T) {
	// Create content that exceeds the truncation threshold but is not marked truncated
	longContent := strings.Repeat("x", TruncationThreshold+100)
	state := ThinkingState{
		Visible:   true,
		Content:   longContent,
		Truncated: false,
	}
	got := RenderThinking(state)
	// Content should be included without truncation
	if !strings.Contains(got, longContent) {
		t.Fatalf("RenderThinking should include full content when Truncated=false in %q", got)
	}
}

// TestToolTrail_SingleNode verifies rendering a single tool call node.
func TestToolTrail_SingleNode(t *testing.T) {
	nodes := []ToolCallNode{
		{
			Name:     "search",
			Status:   ToolCallDone,
			Duration: 1500 * time.Millisecond,
			Depth:    0,
			Children: nil,
		},
	}

	got := RenderToolTrail(nodes)
	if got == "" {
		t.Fatalf("RenderToolTrail with single node returned empty")
	}
	if !strings.Contains(got, "search") {
		t.Fatalf("RenderToolTrail missing tool name 'search' in %q", got)
	}
	if !strings.Contains(got, "🔎") {
		t.Fatalf("RenderToolTrail missing search icon in %q", got)
	}
	if !strings.Contains(got, "✅") {
		t.Fatalf("RenderToolTrail missing done status in %q", got)
	}
}

// TestToolTrail_MultipleNodes verifies rendering multiple tool call nodes.
func TestToolTrail_MultipleNodes(t *testing.T) {
	nodes := []ToolCallNode{
		{
			Name:     "memory",
			Status:   ToolCallDone,
			Duration: 500 * time.Millisecond,
			Depth:    0,
			Children: nil,
		},
		{
			Name:     "bash",
			Status:   ToolCallRunning,
			Duration: 0,
			Depth:    0,
			Children: nil,
		},
	}

	got := RenderToolTrail(nodes)

	// First node should use └─ (last branch)
	if !strings.Contains(got, "🧠") {
		t.Fatalf("RenderToolTrail missing memory icon in %q", got)
	}
	if !strings.Contains(got, "⚡") {
		t.Fatalf("RenderToolTrail missing bash icon in %q", got)
	}
	// Running status
	if !strings.Contains(got, "⏳") {
		t.Fatalf("RenderToolTrail missing running status in %q", got)
	}
}

// TestToolTrail_NestedTree verifies rendering nested tool call tree (subagent delegation).
// Uses two root nodes to ensure the first root is a "mid" branch, which causes
// continuation rails to appear for its children at depth 1.
func TestToolTrail_NestedTree(t *testing.T) {
	nodes := []ToolCallNode{
		{
			Name:     "bash",
			Status:   ToolCallDone,
			Duration: 500 * time.Millisecond,
			Depth:    0,
			Children: nil,
		},
		{
			Name:     "delegate",
			Status:   ToolCallRunning,
			Duration: 0,
			Depth:    0,
			Children: []ToolCallNode{
				{
					Name:     "search",
					Status:   ToolCallDone,
					Duration: 2000 * time.Millisecond,
					Depth:    1,
					Children: []ToolCallNode{
						{
							Name:     "read",
							Status:   ToolCallDone,
							Duration: 800 * time.Millisecond,
							Depth:    2,
							Children: nil,
						},
					},
				},
				{
					Name:     "memory",
					Status:   ToolCallDone,
					Duration: 300 * time.Millisecond,
					Depth:    1,
					Children: nil,
				},
			},
		},
	}

	got := RenderToolTrail(nodes)

	// Parent delegate should be present
	if !strings.Contains(got, "delegate") {
		t.Fatalf("RenderToolTrail missing parent 'delegate' in %q", got)
	}
	if !strings.Contains(got, "🔄") {
		t.Fatalf("RenderToolTrail missing delegate icon in %q", got)
	}

	// Child nodes should be present with tree rails
	if !strings.Contains(got, "search") {
		t.Fatalf("RenderToolTrail missing child 'search' in %q", got)
	}
	if !strings.Contains(got, "read") {
		t.Fatalf("RenderToolTrail missing child 'read' in %q", got)
	}

	// Should have tree rail characters for nested depth
	// Since delegate is a "mid" branch (second of two root nodes), its children
	// at depth 1 should show continuation rails
	if !strings.Contains(got, "│") {
		t.Fatalf("RenderToolTrail missing tree rails for nested depth in %q", got)
	}
}

// TestToolTrail_EmptyList verifies empty node list returns empty string.
func TestToolTrail_EmptyList(t *testing.T) {
	nodes := []ToolCallNode{}
	got := RenderToolTrail(nodes)
	if got != "" {
		t.Fatalf("RenderToolTrail(empty) = %q, want empty string", got)
	}
}

// TestToolTrail_StatusTransitions verifies status transitions from running to done.
func TestToolTrail_StatusTransitions(t *testing.T) {
	// Test running state
	runningNode := []ToolCallNode{
		{
			Name:     "bash",
			Status:   ToolCallRunning,
			Duration: 0,
			Depth:    0,
			Children: nil,
		},
	}
	runningResult := RenderToolTrail(runningNode)
	if !strings.Contains(runningResult, "⏳") {
		t.Fatalf("Running tool missing ⏳ status in %q", runningResult)
	}
	if strings.Contains(runningResult, "s)") {
		t.Fatalf("Running tool should not show duration in %q", runningResult)
	}

	// Test done state
	doneNode := []ToolCallNode{
		{
			Name:     "bash",
			Status:   ToolCallDone,
			Duration: 2500 * time.Millisecond,
			Depth:    0,
			Children: nil,
		},
	}
	doneResult := RenderToolTrail(doneNode)
	if !strings.Contains(doneResult, "✅") {
		t.Fatalf("Done tool missing ✅ status in %q", doneResult)
	}
	if !strings.Contains(doneResult, "2.5s") && !strings.Contains(doneResult, "2s") {
		t.Fatalf("Done tool missing duration in %q", doneResult)
	}

	// Test error state
	errorNode := []ToolCallNode{
		{
			Name:     "bash",
			Status:   ToolCallError,
			Duration: 0,
			Depth:    0,
			Children: nil,
		},
	}
	errorResult := RenderToolTrail(errorNode)
	if !strings.Contains(errorResult, "❌") {
		t.Fatalf("Error tool missing ❌ status in %q", errorResult)
	}
}

// TestToolTrail_ErrorNode verifies error node rendering.
func TestToolTrail_ErrorNode(t *testing.T) {
	nodes := []ToolCallNode{
		{
			Name:     "api",
			Status:   ToolCallError,
			Duration: 0,
			Depth:    0,
			Children: nil,
		},
	}

	got := RenderToolTrail(nodes)
	if !strings.Contains(got, "❌") {
		t.Fatalf("RenderToolTrail error node missing ❌ in %q", got)
	}
	if !strings.Contains(got, "🔌") {
		t.Fatalf("RenderToolTrail error node missing api icon in %q", got)
	}
}

// TestToolIcon_Mappings verifies tool icon mappings.
func TestToolIcon_Mappings(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		wantIcon string
	}{
		{"memory tool", "memory", "🧠"},
		{"search tool", "search", "🔎"},
		{"read tool", "read", "📖"},
		{"write tool", "write", "✏️"},
		{"bash tool", "bash", "⚡"},
		{"shell tool", "shell", "⚡"},
		{"exec tool", "exec", "⚡"},
		{"web tool", "web", "🌐"},
		{"http tool", "http", "🌐"},
		{"fetch tool", "fetch", "🌐"},
		{"code tool", "code", "💻"},
		{"programming tool", "programming", "💻"},
		{"delegate tool", "delegate", "🔄"},
		{"subagent tool", "subagent", "🔄"},
		{"spawn tool", "spawn", "🔄"},
		{"database tool", "database", "🗄️"},
		{"db tool", "db", "🗄️"},
		{"api tool", "api", "🔌"},
		{"unknown tool", "someunknown", "●"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := toolIcon(tc.toolName)
			if got != tc.wantIcon {
				t.Errorf("toolIcon(%q) = %q, want %q", tc.toolName, got, tc.wantIcon)
			}
		})
	}
}

// TestStatusIcon_Mappings verifies status icon mappings.
func TestStatusIcon_Mappings(t *testing.T) {
	tests := []struct {
		name   string
		status ToolCallStatus
		want   string
	}{
		{"running", ToolCallRunning, "⏳"},
		{"done", ToolCallDone, "✅"},
		{"error", ToolCallError, "❌"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statusIcon(tc.status)
			if got != tc.want {
				t.Errorf("statusIcon(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

// TestFormatDuration verifies duration formatting.
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"zero", 0, ""},
		{"negative", -1 * time.Second, ""},
		{"sub-second", 500 * time.Millisecond, "0.5s"},
		{"exact second", 1 * time.Second, "1.0s"},
		{"9.5 seconds", 9500 * time.Millisecond, "9.5s"},
		{"10 seconds", 10 * time.Second, "10s"},
		{"one minute", 1 * time.Minute, "60s"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatDuration(tc.duration)
			if got != tc.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tc.duration, got, tc.want)
			}
		})
	}
}

// TestTreeRails verifies tree rail generation.
func TestTreeRails(t *testing.T) {
	tests := []struct {
		name       string
		depth      int
		wantPrefix string
	}{
		{"zero depth", 0, ""},
		{"depth 1", 1, ""},
		{"depth 2", 2, "│   "},
		{"depth 3", 3, "│   │   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := treeRails(tc.depth)
			if got != tc.wantPrefix {
				t.Errorf("treeRails(%d) = %q, want %q", tc.depth, got, tc.wantPrefix)
			}
		})
	}
}

// TestTreeBranch verifies tree branch indicator generation.
func TestTreeBranch(t *testing.T) {
	tests := []struct {
		name   string
		isLast bool
		want   string
	}{
		{"last branch", true, "└─ "},
		{"mid branch", false, "├─ "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := treeBranch(tc.isLast)
			if got != tc.want {
				t.Errorf("treeBranch(%v) = %q, want %q", tc.isLast, got, tc.want)
			}
		})
	}
}

// TestSpinner_RenderSpinner verifies spinner rendering.
func TestSpinner_RenderSpinner(t *testing.T) {
	tests := []struct {
		name    string
		variant string
		want    string
	}{
		{"think variant", "think", "⠋"},
		{"tool variant", "tool", "⠁"},
		{"unknown variant", "unknown", "⠋"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RenderSpinner(tc.variant)
			if got != tc.want {
				t.Errorf("RenderSpinner(%q) = %q, want %q", tc.variant, got, tc.want)
			}
		})
	}
}

// TestSpinner_RenderSpinnerFrame verifies spinner frame cycling.
func TestSpinner_RenderSpinnerFrame(t *testing.T) {
	// Test think spinner frames
	frames := make(map[string]bool)
	for i := 0; i < 20; i++ {
		frame := RenderSpinnerFrame("think", i)
		frames[frame] = true
	}
	// Should have multiple distinct frames from cycling
	if len(frames) < 3 {
		t.Errorf("RenderSpinnerFrame think cycling should produce multiple frames, got %d unique", len(frames))
	}

	// Test tool spinner frames
	toolFrames := make(map[string]bool)
	for i := 0; i < 20; i++ {
		frame := RenderSpinnerFrame("tool", i)
		toolFrames[frame] = true
	}
	if len(toolFrames) < 3 {
		t.Errorf("RenderSpinnerFrame tool cycling should produce multiple frames, got %d unique", len(toolFrames))
	}
}
