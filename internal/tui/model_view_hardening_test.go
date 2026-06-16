package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestGormesChatModelView_TallDraftDoesNotOverflowHeight(t *testing.T) {
	// A long pasted/multiline draft must not grow the bottom-pinned composer
	// past the terminal height and push the status bar off-screen. The composer
	// height is capped; the conversation viewport absorbs the remaining budget.
	tallDraft := strings.Repeat("draft line\n", 40) + "draft line"

	for _, size := range []struct{ width, height int }{{40, 12}, {80, 20}, {120, 30}} {
		m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
		m.width = size.width
		m.height = size.height
		m.frame = kernel.RenderFrame{
			Phase:   kernel.PhaseIdle,
			Model:   "anthropic/claude-sonnet-4-20250514",
			History: []llm.Message{{Role: "assistant", Content: "ready"}},
		}
		m.editor.SetValue(tallDraft)

		got := m.View()
		lineCount := strings.Count(got, "\n") + 1
		if lineCount > size.height {
			t.Fatalf("Model.View rendered %d lines, exceeds terminal height %d at %+v:\n%s",
				lineCount, size.height, size, got)
		}
	}
}

func TestGormesChatModelView_UserReportHardening(t *testing.T) {
	longToken := strings.Repeat("z", 180)
	final := "final model view answer should appear once"

	cases := []struct {
		name      string
		frame     kernel.RenderFrame
		draft     string
		status    string
		want      []string
		wantOnce  string
		forbidden []string
	}{
		{
			name: "long draft and status stay inside full Bubble Tea chrome",
			frame: kernel.RenderFrame{
				Phase:   kernel.PhaseIdle,
				Model:   "anthropic/claude-sonnet-4-20250514",
				History: []llm.Message{{Role: "assistant", Content: "ready for next task"}},
			},
			draft:  "please inspect " + longToken,
			status: "runtime notice " + longToken,
			want:   []string{"ready for next task", "runtime notice", "please inspect"},
		},
		{
			name: "active quiet turn shows progress without losing prompt",
			frame: kernel.RenderFrame{
				Phase:   kernel.PhaseStreaming,
				Model:   "openai/gpt-4.1",
				History: []llm.Message{{Role: "user", Content: "start a long running task"}},
			},
			draft: "queued follow up",
			want:  []string{"start a long running task", "running", "queued follow up"},
		},
		{
			name: "final assistant is not duplicated by stale draft",
			frame: kernel.RenderFrame{
				Phase:     kernel.PhaseIdle,
				Model:     "local/test-model",
				DraftText: final,
				History: []llm.Message{
					{Role: "user", Content: "finish"},
					{Role: "assistant", Content: final},
				},
				SoulEvents: []kernel.SoulEntry{{Text: `tool: terminal: "secret command"`}},
			},
			want:      []string{"finish", "final model view answer"},
			wantOnce:  final,
			forbidden: []string{"secret command", "tool iteration limit exceeded"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, size := range []struct{ width, height int }{{20, 10}, {40, 12}, {80, 20}} {
				m := NewModel(make(chan kernel.RenderFrame), func(string) {}, func() {})
				m.width = size.width
				m.height = size.height
				m.frame = tc.frame
				m.statusMessage = tc.status
				if tc.draft != "" {
					m.editor.SetValue(tc.draft)
				}

				got := m.View()
				if strings.TrimSpace(got) == "" {
					t.Fatalf("Model.View returned blank output at %+v", size)
				}
				for _, line := range strings.Split(got, "\n") {
					if w := lipgloss.Width(line); w > size.width {
						t.Fatalf("Model.View line width %d exceeds terminal width %d:\n%q\n\nfull output:\n%s", w, size.width, line, got)
					}
				}
				collapsed := collapseWhitespace(got)
				for _, want := range tc.want {
					if !strings.Contains(collapsed, want) {
						t.Fatalf("Model.View missing %q at %+v:\n%s", want, size, got)
					}
				}
				if tc.wantOnce != "" {
					if count := strings.Count(collapseWhitespace(got), tc.wantOnce); count != 1 {
						t.Fatalf("Model.View %q count = %d, want 1:\n%s", tc.wantOnce, count, got)
					}
				}
				for _, forbidden := range tc.forbidden {
					if strings.Contains(got, forbidden) {
						t.Fatalf("Model.View leaked forbidden %q:\n%s", forbidden, got)
					}
				}
			}
		})
	}
}
