package panels_test

import (
	"strings"
	"testing"
	"time"

	tui "github.com/TrebuchetDynamics/gormes-agent/internal/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestHermesPanels_ToolSpinnerShowsElapsed proves that tool.started style
// spinner text includes the tool name/preview and a live elapsed timer in the
// Hermes mm:ss / "x.ys" / "Nm Ms" formats. Tracks
// ../hermes-agent/cli.py:_render_spinner_text.

func forceLipglossTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

func TestHermesPanels_ToolSpinnerShowsElapsed(t *testing.T) {
	tests := []struct {
		name         string
		state        tui.ToolSpinnerState
		mustContain  []string
		mustNotEmpty bool
	}{
		{
			name: "sub-minute renders single-decimal seconds",
			state: tui.ToolSpinnerState{
				ToolName: "bash",
				Preview:  "ls -la",
				Elapsed:  3500 * time.Millisecond,
			},
			mustContain: []string{"ls -la", "3.5s"},
		},
		{
			name: "over-minute renders Xm Ys",
			state: tui.ToolSpinnerState{
				ToolName: "search",
				Preview:  "grep foo",
				Elapsed:  75 * time.Second,
			},
			mustContain: []string{"grep foo", "1m 15s"},
		},
		{
			name: "zero elapsed omits the timer entirely",
			state: tui.ToolSpinnerState{
				ToolName: "bash",
				Preview:  "pwd",
				Elapsed:  0,
			},
			mustContain: []string{"pwd"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tui.RenderToolSpinner(tc.state)
			if got == "" {
				t.Fatalf("tui.RenderToolSpinner(%+v) = empty, want non-empty spinner text", tc.state)
			}
			for _, want := range tc.mustContain {
				if !strings.Contains(got, want) {
					t.Fatalf("tui.RenderToolSpinner(%+v) = %q, want contains %q", tc.state, got, want)
				}
			}
			if tc.state.Elapsed == 0 && (strings.Contains(got, "(") && strings.Contains(got, "s)")) {
				t.Fatalf("tui.RenderToolSpinner(%+v) = %q, must not include timer when Elapsed==0", tc.state, got)
			}
		})
	}
}

// TestHermesPanels_ToolScrollbackAllNewModes proves stacked tool-completion
// lines render under both tui.ToolScrollAll and tui.ToolScrollNew, that "new" mode
// suppresses consecutive duplicates, and that errors are tagged with [error].
// Tracks ../hermes-agent/cli.py:_on_tool_progress (tool.completed branch).
func TestHermesPanels_ToolScrollbackAllNewModes(t *testing.T) {
	items := []tui.ToolCompletion{
		{ToolName: "bash", Detail: "ls -la (0.4s)", Error: false},
		{ToolName: "bash", Detail: "ls (0.2s)", Error: false},
		{ToolName: "search", Detail: "grep foo (1.2s)", Error: true},
		{ToolName: "bash", Detail: "pwd (0.1s)", Error: false},
	}

	t.Run("all mode renders every completion stacked", func(t *testing.T) {
		got := tui.RenderToolScrollback(items, tui.ToolScrollAll)
		if len(got) != len(items) {
			t.Fatalf("tui.RenderToolScrollback all mode produced %d lines, want %d", len(got), len(items))
		}
		joined := strings.Join(got, "\n")
		for _, want := range []string{"ls -la", "ls (", "grep foo", "pwd"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("all-mode scrollback missing %q in %q", want, joined)
			}
		}
		if !strings.Contains(got[2], "[error]") {
			t.Fatalf("error completion not tagged: %q", got[2])
		}
		if strings.Contains(got[0], "[error]") || strings.Contains(got[1], "[error]") || strings.Contains(got[3], "[error]") {
			t.Fatalf("non-error completions wrongly tagged: %q", got)
		}
	})

	t.Run("new mode suppresses consecutive duplicates of the same tool", func(t *testing.T) {
		got := tui.RenderToolScrollback(items, tui.ToolScrollNew)
		// items[0]=bash, items[1]=bash (suppressed), items[2]=search,
		// items[3]=bash (different from previous => kept).
		if len(got) != 3 {
			t.Fatalf("new-mode scrollback len = %d, want 3 (consecutive bash dedup), got %q", len(got), got)
		}
		joined := strings.Join(got, "\n")
		if strings.Contains(joined, "ls (0.2s)") {
			t.Fatalf("new-mode scrollback should suppress consecutive duplicate bash entry: %q", joined)
		}
		if !strings.Contains(joined, "ls -la") || !strings.Contains(joined, "grep foo") || !strings.Contains(joined, "pwd") {
			t.Fatalf("new-mode scrollback missing expected entries: %q", joined)
		}
		if !strings.Contains(got[1], "[error]") {
			t.Fatalf("new-mode error completion not tagged: %q", got[1])
		}
	})
}

// TestHermesPanels_ApprovalPanelPreservesCommandAndChoices proves the
// dangerous-command approval panel keeps the command and every choice visible
// even with long descriptions, that the selected choice carries the Hermes
// "❯" prefix, and that tui.ApprovalView (when present and toggled via
// ViewExpanded) expands the command in place rather than truncating it.
// Tracks ../hermes-agent/cli.py:_get_approval_display_fragments.
func TestHermesPanels_ApprovalPanelPreservesCommandAndChoices(t *testing.T) {
	longCmd := strings.Repeat("rm -rf /tmp/foo-bar-baz/", 6) // > 70 chars
	longDesc := strings.Repeat("This command will affect many files. ", 12)

	state := tui.ApprovalPanelState{
		Description:  longDesc,
		Command:      longCmd,
		Choices:      []tui.ApprovalChoice{tui.ApprovalOnce, tui.ApprovalSession, tui.ApprovalAlways, tui.ApprovalDeny, tui.ApprovalView},
		Selected:     tui.ApprovalSession,
		ViewExpanded: false,
		Width:        80,
		Height:       20,
	}

	got := tui.RenderApprovalPanel(state)
	if got == "" {
		t.Fatalf("tui.RenderApprovalPanel returned empty string")
	}

	// Command must always be visible (truncated form is acceptable when
	// ViewExpanded is false). The first 30 chars of the command must
	// always be readable so the user can identify it.
	cmdHead := longCmd[:30]
	if !strings.Contains(got, cmdHead) {
		t.Fatalf("approval panel must include command head %q in %q", cmdHead, got)
	}
	// Every choice label must be present; "❯" must mark the selected
	// option (Allow for this session).
	for _, want := range []string{"Allow once", "Allow for this session", "permanent allowlist", "Deny"} {
		if !strings.Contains(got, want) {
			t.Fatalf("approval panel missing choice label %q in %q", want, got)
		}
	}
	if !strings.Contains(got, "❯") {
		t.Fatalf("approval panel missing selected-choice marker '❯' in %q", got)
	}
	// The selected line must be the session line. Find it by scanning
	// each line for both '❯' and the session label.
	foundSelectedSession := false
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "❯") && strings.Contains(line, "Allow for this session") {
			foundSelectedSession = true
			break
		}
	}
	if !foundSelectedSession {
		t.Fatalf("approval panel did not mark Allow-for-this-session as selected in %q", got)
	}

	// When ViewExpanded is true, the full command must render verbatim
	// (no truncation marker) and the "view" choice should be removed.
	expanded := state
	expanded.ViewExpanded = true
	expanded.Selected = tui.ApprovalOnce
	expandedRender := tui.RenderApprovalPanel(expanded)
	if !strings.Contains(expandedRender, longCmd) {
		t.Fatalf("ViewExpanded approval panel must include the full command verbatim; got %q", expandedRender)
	}
	if strings.Contains(expandedRender, "Show full command") {
		t.Fatalf("ViewExpanded approval panel must drop the 'view' choice; got %q", expandedRender)
	}
}

func TestHermesPanels_WithSkinUseSharedChromeStyles(t *testing.T) {
	forceLipglossTrueColor(t)
	skin := tui.BuiltinSkins()["poseidon"]
	state := tui.ApprovalPanelState{
		Description: "Review shell command before running it.",
		Command:     "rm -rf /tmp/example",
		Choices:     []tui.ApprovalChoice{tui.ApprovalOnce, tui.ApprovalDeny},
		Selected:    tui.ApprovalDeny,
		Width:       54,
		Height:      12,
	}

	got := tui.RenderApprovalPanelWithSkin(state, skin)
	for _, want := range []string{"Dangerous Command", "rm -rf", "Deny", "❯"} {
		if !strings.Contains(got, want) {
			t.Fatalf("styled approval panel missing %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("styled approval panel should use shared skin styles; got no ANSI styling:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if w := lipgloss.Width(line); w > 54 {
			t.Fatalf("styled approval panel line width %d exceeds 54: %q\n\n%s", w, line, got)
		}
	}
}

// TestHermesPanels_ClarifyPanelRendersNumberedChoicesAndOther proves the
// clarify panel renders 1..N numbered choices, marks the selected choice with
// "❯", appends an "Other" option, and includes the timeout hint. Tracks
// ../hermes-agent/cli.py:_get_clarify_display.
func TestHermesPanels_ClarifyPanelRendersNumberedChoicesAndOther(t *testing.T) {
	state := tui.ClarifyPanelState{
		Question:    "Which database should I use for this slice?",
		Choices:     []string{"Postgres", "SQLite", "DuckDB"},
		Selected:    1, // SQLite
		TimeoutHint: "(45s)",
		Width:       80,
		Height:      24,
	}

	got := tui.RenderClarifyPanel(state)
	if got == "" {
		t.Fatalf("tui.RenderClarifyPanel returned empty string")
	}
	if !strings.Contains(got, "Which database should I use") {
		t.Fatalf("clarify panel missing question text in %q", got)
	}
	for i, choice := range state.Choices {
		num := i + 1
		// Numbered prefix "1." / "2." / "3." must be present.
		marker := strings.Join([]string{string(rune('0' + num)), ". ", choice}, "")
		if !strings.Contains(got, marker) {
			t.Fatalf("clarify panel missing numbered choice %q in %q", marker, got)
		}
	}
	// Other option appears as the next number after the choices (4. Other).
	otherNum := len(state.Choices) + 1
	otherMarker := string(rune('0'+otherNum)) + ". Other"
	if !strings.Contains(got, otherMarker) {
		t.Fatalf("clarify panel missing Other option %q in %q", otherMarker, got)
	}
	// Selected choice (index 1 / "SQLite") must carry the ❯ marker.
	foundSelected := false
	for _, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "❯") && strings.Contains(line, "SQLite") {
			foundSelected = true
			break
		}
	}
	if !foundSelected {
		t.Fatalf("clarify panel did not mark SQLite as selected in %q", got)
	}
	if !strings.Contains(got, "(45s)") {
		t.Fatalf("clarify panel missing timeout hint in %q", got)
	}
}

// TestHermesPanels_SecretAndSudoPanelsNeverRenderSecretValue proves the secret
// and sudo input panels surface the prompt, hint, and countdown text but do
// not echo any secret bytes — the panel state itself only carries the input
// length, never the value. Tracks
// ../hermes-agent/cli.py:_sudo_password_callback and
// ../hermes-agent/cli.py:_get_secret_display.
func TestHermesPanels_SecretAndSudoPanelsNeverRenderSecretValue(t *testing.T) {
	const secretValue = "hunter2-do-not-leak-this-into-the-tui"

	tests := []struct {
		name         string
		state        tui.SecretPanelState
		mustContain  []string
		minSecretLen int
	}{
		{
			name: "sudo panel shows hint + countdown + masked length",
			state: tui.SecretPanelState{
				Mode:       tui.SecretPanelSudo,
				PromptText: "[sudo] password:",
				Countdown:  45 * time.Second,
				SecretLen:  len(secretValue),
				Hint:       "Enter to submit, ESC to skip",
			},
			mustContain:  []string{"[sudo] password", "Enter to submit", "(45s)"},
			minSecretLen: len(secretValue),
		},
		{
			name: "secret panel with help/hint shows prompt and hint",
			state: tui.SecretPanelState{
				Mode:       tui.SecretPanelArbitrary,
				PromptText: "Enter API token",
				Countdown:  0,
				SecretLen:  len(secretValue),
				Hint:       "type secret (hidden), Enter to submit · ESC to skip",
			},
			mustContain:  []string{"Enter API token", "type secret"},
			minSecretLen: len(secretValue),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tui.RenderSecretPanel(tc.state)
			if got == "" {
				t.Fatalf("tui.RenderSecretPanel returned empty for %+v", tc.state)
			}
			for _, want := range tc.mustContain {
				if !strings.Contains(got, want) {
					t.Fatalf("tui.RenderSecretPanel = %q, want contains %q", got, want)
				}
			}
			// Substring-scan for the literal secret value.
			if strings.Contains(got, secretValue) {
				t.Fatalf("tui.RenderSecretPanel must never echo the secret value; got %q", got)
			}
			// Prefixes/suffixes of the secret must not leak either.
			if strings.Contains(got, "hunter2") {
				t.Fatalf("tui.RenderSecretPanel leaked secret prefix; got %q", got)
			}
			// If a placeholder is shown for typed length, it must not be the
			// raw value — only mask characters or the length number itself.
			if strings.Contains(got, secretValue[:5]) {
				t.Fatalf("tui.RenderSecretPanel leaked secret head; got %q", got)
			}
		})
	}
}
