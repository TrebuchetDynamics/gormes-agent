package admin

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestAdminCommands_RendersWholeCLICatalog(t *testing.T) {
	entries := []CommandEntry{
		{Path: "auth add", Use: "gormes auth add <provider>", Short: "Add a provider credential"},
		{Path: "gateway status", Use: "gormes gateway status", Short: "check gateway runtime state", Runnable: true},
		{Path: "kanban list", Use: "gormes kanban list", Short: "list tasks"},
	}
	shell := New(NewCommandsScreen(entries))
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("CLI commands")) &&
			bytes.Contains(out, []byte("3 commands")) &&
			bytes.Contains(out, []byte("gormes auth add <provider>")) &&
			bytes.Contains(out, []byte("Add a provider credential"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Selected: gormes gateway status")) &&
			bytes.Contains(out, []byte("check gateway runtime state"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminCommands_EnterRunsSafeCommandInline(t *testing.T) {
	entries := []CommandEntry{
		{Path: "gateway status", Use: "gormes gateway status", Short: "check gateway runtime state", Runnable: true, RunLabel: "gormes gateway status"},
	}
	ran := false
	screen := NewCommandsScreen(entries, WithCommandRunner(func(entry CommandEntry) CommandRunResult {
		ran = true
		if entry.Path != "gateway status" {
			t.Fatalf("runner entry path = %q, want gateway status", entry.Path)
		}
		return CommandRunResult{RunLabel: entry.RunLabel, Output: "gateway runtime: stopped\n"}
	}))
	tm := teatest.NewTestModel(t, New(screen), teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Enter: run gormes gateway status"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Command output: gormes gateway status")) &&
			bytes.Contains(out, []byte("gateway runtime: stopped"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	if !ran {
		t.Fatal("expected runner to be called")
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminCommands_SearchFiltersAndRunsSelectedMatch(t *testing.T) {
	entries := []CommandEntry{
		{Path: "auth status", Use: "gormes auth status <provider>", Short: "show redacted provider auth status", Runnable: true, RunLabel: "gormes auth status openai-codex"},
		{Path: "gateway status", Use: "gormes gateway status", Short: "check gateway runtime state", Runnable: true, RunLabel: "gormes gateway status"},
		{Path: "kanban list", Use: "gormes kanban list", Short: "list tasks", Runnable: true, RunLabel: "gormes kanban list"},
	}
	ran := false
	screen := NewCommandsScreen(entries, WithCommandRunner(func(entry CommandEntry) CommandRunResult {
		ran = true
		if entry.Path != "gateway status" {
			t.Fatalf("runner entry path = %q, want gateway status", entry.Path)
		}
		return CommandRunResult{RunLabel: entry.RunLabel, Output: "gateway runtime: stopped\n"}
	}))
	tm := teatest.NewTestModel(t, New(screen), teatest.WithInitialTermSize(100, 30))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("gate")

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Search: gate")) &&
			bytes.Contains(out, []byte("1 of 3 commands")) &&
			bytes.Contains(out, []byte("gormes gateway status")) &&
			bytes.Contains(out, []byte("Selected: gormes gateway status"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Command output: gormes gateway status")) &&
			bytes.Contains(out, []byte("gateway runtime: stopped"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	if !ran {
		t.Fatal("expected runner to be called")
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminCommands_SearchCapturesGlobalShortcutRunes(t *testing.T) {
	entries := []CommandEntry{
		{Path: "auth login", Use: "gormes auth login openai-codex", Short: "authenticate openai codex"},
	}
	shell := New(NewCommandsScreen(entries), &stubScreen{name: "Chat"})
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(100, 30))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("codex")

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Search: codex")) &&
			bytes.Contains(out, []byte("Selected: gormes auth login openai-codex"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	if got := shell.ActiveIndex(); got != 0 {
		t.Fatalf("active index after command search = %d, want Commands tab", got)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminCommands_EnterRejectsNonRunnableCommand(t *testing.T) {
	entries := []CommandEntry{
		{Path: "auth add", Use: "gormes auth add <provider>", Short: "Add a provider credential"},
	}
	screen := NewCommandsScreen(entries, WithCommandRunner(func(entry CommandEntry) CommandRunResult {
		t.Fatalf("runner should not be called for non-runnable command: %#v", entry)
		return CommandRunResult{}
	}))
	tm := teatest.NewTestModel(t, New(screen), teatest.WithInitialTermSize(100, 30))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Command is not runnable inside gormes admin")) &&
			bytes.Contains(out, []byte("mutating commands stay explicit"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminDefaultScreensIncludesCommandsTab(t *testing.T) {
	screens := NewDefaultScreens(WithCommandEntries([]CommandEntry{
		{Path: "doctor", Use: "gormes doctor", Short: "check readiness"},
	}))
	var titles []string
	for _, screen := range screens {
		titles = append(titles, screen.Title())
	}
	want := []string{"Setup", "Chat", "Agents", "Commands"}
	if len(titles) != len(want) {
		t.Fatalf("default screen titles = %#v, want %#v", titles, want)
	}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("default screen titles = %#v, want %#v", titles, want)
		}
	}
}
