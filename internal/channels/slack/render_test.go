package slack

import (
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestFormatStream_RendersHermesToolProgressBlock(t *testing.T) {
	got := formatStream(kernel.RenderFrame{
		SoulEvents: []kernel.SoulEntry{
			{At: time.Now(), Text: `tool: skill_view: plan`},
			{At: time.Now(), Text: `tool: search_files: chrono|cron`},
			{At: time.Now(), Text: `tool: terminal: git status --short`},
			{At: time.Now(), Text: `tool: terminal: git status --short`},
			{At: time.Now(), Text: `tool done: execute_code`},
		},
	})

	for _, want := range []string{
		`📚 skill_view: "plan"`,
		`🔎 search_files: "chrono|cron"`,
		`ACTION [repo] Inspecting repository status (×2)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatStream missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tool done") || strings.Contains(got, `🔧 tool`) {
		t.Fatalf("formatStream leaked generic tool text:\n%s", got)
	}
}
