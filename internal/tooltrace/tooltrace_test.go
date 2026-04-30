package tooltrace

import (
	"strings"
	"testing"
)

func TestFormatBlock_HermesTranscriptShapeAndDedup(t *testing.T) {
	got := FormatBlock([]string{
		`tool: skill_view: plan`,
		`tool: todo: planning 5 task(s)`,
		`tool: search_files: chrono|cron`,
		`tool: terminal: git status --short`,
		`tool: terminal: git status --short`,
		`tool: read_file: /tmp/example.go`,
		`tool done: execute_code`,
	})

	for _, want := range []string{
		`📚 skill_view: "plan"`,
		`📋 todo: "planning 5 task(s)"`,
		`🔎 search_files: "chrono|cron"`,
		`💻 terminal: "git status --short" (×2)`,
		`📖 read_file: "/tmp/example.go"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatBlock missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "tool done") || strings.Contains(got, `🔧 tool done`) {
		t.Fatalf("FormatBlock leaked completion noise:\n%s", got)
	}
}

func TestFormatBlockModeNewSuppressesConsecutiveSameTool(t *testing.T) {
	got := FormatBlockMode([]string{
		`tool: read_file: a.go`,
		`tool: read_file: b.go`,
		`tool: search_files: needle`,
		`tool: read_file: c.go`,
		`tool done: read_file`,
		`tool: read_file: d.go`,
	}, "new")

	if strings.Contains(got, "b.go") || strings.Contains(got, "d.go") {
		t.Fatalf("new mode should suppress consecutive read_file entries:\n%s", got)
	}
	for _, want := range []string{"a.go", "search_files", "c.go"} {
		if !strings.Contains(got, want) {
			t.Fatalf("new mode missing %q in:\n%s", want, got)
		}
	}
}
