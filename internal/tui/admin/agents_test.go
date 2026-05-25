package admin

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	goncho "github.com/TrebuchetDynamics/goncho/dynamicagents"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/memory"
)

const adminAgentsWait = 5 * time.Second

func TestAdminAgents_EmptyRegistryShowsSpawnCTA(t *testing.T) {
	isolateHealthHome(t)
	shell := New(NewAgentsScreen())
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))
	defer stopAdminAgentsTestModel(t, tm)

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("press n to spawn your first agent"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))
}

func TestAdminAgents_SpawnWizardCreatesRecord(t *testing.T) {
	isolateHealthHome(t)
	shell := New(NewAgentsScreen())
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))
	defer stopAdminAgentsTestModel(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Spawn agent"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Type("Research Bot")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Type("literature review")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Research Bot")) &&
			bytes.Contains(out, []byte("research-bot"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))

	registry, closeRegistry := openAdminAgentsRegistryForTest(t)
	defer closeRegistry()
	got, found, err := registry.Get(context.Background(), "research-bot")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || got.Name != "Research Bot" || got.Persona != "literature review" {
		t.Fatalf("spawned agent = %+v found=%v, want Research Bot/literature review", got, found)
	}
}

func TestAdminAgents_BindWizardWritesBinding(t *testing.T) {
	isolateHealthHome(t)
	seedDynamicAgent(t, "Research Bot", "literature review")
	shell := New(NewAgentsScreen())
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))
	defer stopAdminAgentsTestModel(t, tm)

	driveAgentBindWizard(t, tm)

	registry, closeRegistry := openAdminAgentsRegistryForTest(t)
	defer closeRegistry()
	got, found, err := registry.Resolve(context.Background(), goncho.BindingMatch{
		Channel:  "telegram",
		PeerKind: "user",
		PeerID:   "user-42",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !found || got != "research-bot" {
		t.Fatalf("Resolve found=%v id=%q, want research-bot", found, got)
	}
}

func TestAdminAgents_UnbindRemovesBinding(t *testing.T) {
	isolateHealthHome(t)
	seedDynamicAgent(t, "Research Bot", "literature review")
	shell := New(NewAgentsScreen())
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))
	defer stopAdminAgentsTestModel(t, tm)
	driveAgentBindWizard(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("binding removed"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))

	registry, closeRegistry := openAdminAgentsRegistryForTest(t)
	defer closeRegistry()
	_, found, err := registry.Resolve(context.Background(), goncho.BindingMatch{
		Channel:  "telegram",
		PeerKind: "user",
		PeerID:   "user-42",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if found {
		t.Fatalf("binding still resolves after unbind")
	}
}

func driveAgentBindWizard(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Research Bot"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Bind agent"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // agent=Research Bot
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // channel=telegram
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // peer_kind=user
	tm.Type("user-42")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // empty thread
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter}) // confirm
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("last binding: research-bot -> telegram/user/user-42"))
	}, teatest.WithDuration(adminAgentsWait), teatest.WithCheckInterval(10*time.Millisecond))
}

func stopAdminAgentsTestModel(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(adminAgentsWait))
}

func openAdminAgentsRegistryForTest(t *testing.T) (*goncho.DynamicAgentRegistry, func()) {
	t.Helper()
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	registry, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		_ = store.Close(context.Background())
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	return registry, func() {
		_ = store.Close(context.Background())
	}
}
