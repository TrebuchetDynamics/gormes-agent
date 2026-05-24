package trace

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
		`ACTION [repo] Inspecting repository status (×2)`,
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

func TestFormatBlock_OperatorSemanticProgressHidesShellMechanics(t *testing.T) {
	got := FormatBlock([]string{
		`tool: memory`,
		`tool: memory`,
		`tool: terminal: printf '%s\n' '--- ~/.gormes profile state ---'`,
		`tool: terminal: printf '%s\n' '--- oversized skill events ---'`,
		`tool: terminal: printf '%s\n' 'active_profile:'; sed -n '1p' ~/.gormes/active_profile`,
		`tool: terminal: printf '%s\n' 'profile config candidates'; find ~/.gormes -name config.toml`,
		`tool: terminal: wc -c /home/xel/.gormes/profiles/miner/state.json`,
		`tool: terminal: python3 - <<'PY'
from pathlib import Path
print(Path('state.json').stat().st_size)
PY`,
	})

	for _, want := range []string{
		`INFO   [memory] Loading session memory (×2)`,
		`ACTION [profile] Inspecting Gormes profile state`,
		`ACTION [skills] Checking oversized skill events`,
		`ACTION [profile] Inspecting active Gormes profile`,
		`ACTION [config] Verifying profile config candidates`,
		`ACTION [profile] Measuring profile state size`,
		`ACTION [profile] Parsing profile state`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatBlock missing semantic progress %q in:\n%s", want, got)
		}
	}
	for _, forbidden := range []string{
		"printf", "sed -n", "python3", "wc -c", "💻 terminal", "🧠 memory", "tool_progress",
		"/home/xel/.gormes/profiles/miner/state.json",
	} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("FormatBlock leaked low-level detail %q in:\n%s", forbidden, got)
		}
	}
}
