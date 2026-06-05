package mouse

import (
	"bytes"
	"io"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	bubbleTeaEnableMouseAllMotionSeq = "\x1b[?1003h\x1b[?1006h"
	bubbleTeaDisableMouseSeq         = "\x1b[?1002l\x1b[?1003l\x1b[?1006l"
)

func TestHandleMouseSlash(t *testing.T) {
	if got := HandleMouseSlash("/mouse on", false); got != (MouseSlashDecision{Handled: true, Apply: true, Next: true, Status: "mouse tracking on"}) {
		t.Fatalf("HandleMouseSlash on = %#v", got)
	}
	if got := HandleMouseSlash("/mouse on", true); got != (MouseSlashDecision{Handled: true, Apply: false, Next: true, Status: "mouse tracking on"}) {
		t.Fatalf("HandleMouseSlash duplicate on = %#v", got)
	}
	if got := HandleMouseSlash("/mouse sideways", true); got != (MouseSlashDecision{Handled: true, Status: MouseSlashUsage}) {
		t.Fatalf("HandleMouseSlash invalid = %#v", got)
	}
}

func TestParseMouseTrackingSlash(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		current bool
		want    MouseSlashResult
	}{
		{name: "bare mouse toggles off", input: "/mouse", current: true, want: MouseSlashResult{Handled: true, Valid: true, Next: false}},
		{name: "toggle toggles on", input: "/mouse toggle", current: false, want: MouseSlashResult{Handled: true, Valid: true, Next: true}},
		{name: "on enables", input: "/mouse on", current: false, want: MouseSlashResult{Handled: true, Valid: true, Next: true}},
		{name: "off disables", input: "/mouse off", current: true, want: MouseSlashResult{Handled: true, Valid: true, Next: false}},
		{name: "scroll alias disables", input: "/scroll off", current: true, want: MouseSlashResult{Handled: true, Valid: true, Next: false}},
		{name: "invalid value is handled as usage error", input: "/mouse sideways", current: true, want: MouseSlashResult{Handled: true, Valid: false, Message: MouseSlashUsage}},
		{name: "other slash command is not handled", input: "/help", current: true, want: MouseSlashResult{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseMouseTrackingSlash(tt.input, tt.current); got != tt.want {
				t.Fatalf("ParseMouseTrackingSlash(%q, %v) = %#v, want %#v", tt.input, tt.current, got, tt.want)
			}
		})
	}
}

func TestDefaultMouseModeCmdEmitsBubbleTeaTerminalSequences(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		want    string
	}{
		{name: "enable", enabled: true, want: bubbleTeaEnableMouseAllMotionSeq},
		{name: "disable", enabled: false, want: bubbleTeaDisableMouseSeq},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := terminalOutputForCmd(t, DefaultMouseModeCmd(tt.enabled))
			if !strings.Contains(out, tt.want) {
				t.Fatalf("terminal output missing %q:\n%q", tt.want, out)
			}
		})
	}
}

type initCmdModel struct {
	cmd tea.Cmd
}

func (m initCmdModel) Init() tea.Cmd {
	return tea.Sequence(m.cmd, tea.Quit)
}

func (m initCmdModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m initCmdModel) View() string {
	return ""
}

func terminalOutputForCmd(t *testing.T, cmd tea.Cmd) string {
	t.Helper()
	var out bytes.Buffer
	p := tea.NewProgram(
		initCmdModel{cmd: cmd},
		tea.WithInput(bytes.NewBuffer(nil)),
		tea.WithOutput(&out),
		tea.WithoutSignals(),
	)
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(&out)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
