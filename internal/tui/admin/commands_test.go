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
		{Path: "gateway status", Use: "gormes gateway status", Short: "check gateway runtime state"},
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
