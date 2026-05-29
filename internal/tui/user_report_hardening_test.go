package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestUserReportHardening_RenderConvPathologicalFrames(t *testing.T) {
	longToken := strings.Repeat("x", 180)
	final := "final answer must render exactly once"
	cases := []struct {
		name  string
		frame kernel.RenderFrame
	}{
		{
			name: "long unbroken user and assistant content",
			frame: kernel.RenderFrame{History: []llm.Message{
				{Role: "user", Content: "please inspect " + longToken},
				{Role: "assistant", Content: "result " + longToken},
			}},
		},
		{
			name: "ansi and control-like transcript text is bounded",
			frame: kernel.RenderFrame{History: []llm.Message{
				{Role: "user", Content: "show colors"},
				{Role: "assistant", Content: "\x1b[31mred\x1b[0m\n\x1b]52;c;secret\x07"},
			}},
		},
		{
			name: "active quiet turn keeps latest prompt visible",
			frame: kernel.RenderFrame{
				Phase:   kernel.PhaseStreaming,
				History: []llm.Message{{Role: "user", Content: "do not leave me wondering"}},
			},
		},
		{
			name: "tool output and progress are bounded",
			frame: kernel.RenderFrame{
				Phase: kernel.PhaseStreaming,
				History: []llm.Message{
					{Role: "user", Content: "run verbose tool"},
					{Role: "tool", Name: "terminal", Content: strings.Repeat("tool line with lots of output\n", 40)},
				},
				SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: `tool: terminal: "` + longToken + `"`}},
			},
		},
		{
			name: "final answer never duplicates with stale draft and tool progress",
			frame: kernel.RenderFrame{
				Phase:     kernel.PhaseIdle,
				DraftText: final,
				History: []llm.Message{
					{Role: "user", Content: "summarize"},
					{Role: "assistant", Content: final},
				},
				SoulEvents: []kernel.SoulEntry{{At: time.Now(), Text: `tool: terminal: "grep secret"`}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, size := range []struct{ width, height int }{{20, 10}, {40, 12}, {80, 18}, {120, 30}} {
				t.Run(fmt.Sprintf("%dx%d", size.width, size.height), func(t *testing.T) {
					got := renderConv(tc.frame, size.width, size.height)
					if strings.TrimSpace(got) == "" {
						t.Fatalf("renderConv returned blank output for %+v", size)
					}
					for _, line := range strings.Split(got, "\n") {
						if w := lipgloss.Width(line); w > size.width {
							t.Fatalf("line width %d exceeds terminal width %d:\n%q\n\nfull output:\n%s", w, size.width, line, got)
						}
					}
					if strings.Contains(got, "tool iteration limit exceeded") {
						t.Fatalf("leaked internal tool budget text:\n%s", got)
					}
					if tc.name == "final answer never duplicates with stale draft and tool progress" {
						if count := strings.Count(collapseWhitespace(got), final); count != 1 {
							t.Fatalf("final answer count = %d, want 1:\n%s", count, got)
						}
						if strings.Contains(got, "grep secret") || strings.Contains(got, "terminal") {
							t.Fatalf("final idle frame leaked stale tool progress:\n%s", got)
						}
					}
				})
			}
		})
	}
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
