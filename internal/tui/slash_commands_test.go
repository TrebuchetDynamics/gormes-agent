package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSlashCommandAliasAndPrefixDispatch(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
	}{
		{name: "catalog alias", input: "/provider openrouter", wantStatus: "/provider -> /model"},
		{name: "unique prefix", input: "/platf --json", wantStatus: "/platf -> /platforms"},
		{name: "exact wins over prefix", input: "/status now", wantStatus: "/status is recognized"},
		{name: "ambiguous prefix", input: "/stat now", wantStatus: "ambiguous command: /status, /statusbar"},
		{name: "unknown command", input: "/no-such-command-xyzzy", wantStatus: "unknown command /no-such-command-xyzzy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub := &nopSubmitter{}
			m := newSlashDispatchBehaviorModel(sub)
			m.editor.SetValue(tt.input)
			next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			runTestCmd(t, cmd)
			updated := next.(Model)

			if sub.calls != 0 {
				t.Fatalf("%s reached Submitter %d time(s), want 0", tt.input, sub.calls)
			}
			if got := updated.editor.Value(); got != "" {
				t.Fatalf("editor value after %s = %q, want cleared", tt.input, got)
			}
			if !strings.Contains(updated.statusMessage, tt.wantStatus) {
				t.Fatalf("status after %s = %q, want it to contain %q", tt.input, updated.statusMessage, tt.wantStatus)
			}
		})
	}
}
