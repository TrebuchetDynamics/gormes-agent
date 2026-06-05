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
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/doctor"
)

func TestAdminChat_GlobalKeyJumpsFromOtherScreen(t *testing.T) {
	isolateHealthHome(t)
	shell := New(
		NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
			return []HealthItem{{ID: "provider", Status: doctor.StatusPass, Title: "provider configured", Detail: "test"}}, nil
		}))),
		NewChatScreen(),
	)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))

	if got := shell.ActiveIndex(); got != 0 {
		t.Fatalf("initial ActiveIndex = %d, want 0", got)
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	waitForActive(t, shell, 1)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("[2 Chat]")) &&
			bytes.Contains(out, []byte("Chat scroll"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminChat_AgentSwapKeyOpensPicker(t *testing.T) {
	isolateHealthHome(t)
	seedDynamicAgent(t, "Research Bot", "literature review")
	chat := NewChatScreen()
	shell := New(chat)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Agent picker")) &&
			bytes.Contains(out, []byte("Research Bot"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Agent: research-bot"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestAdminChat_EscapeReturnsToPreviousScreen(t *testing.T) {
	isolateHealthHome(t)
	shell := New(
		NewSetupHealthScreen(WithHealthSource(healthSourceFunc(func(context.Context) ([]HealthItem, error) {
			return []HealthItem{{ID: "provider", Status: doctor.StatusPass, Title: "provider configured", Detail: "test"}}, nil
		}))),
		NewChatScreen(),
	)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	waitForActive(t, shell, 1)
	tm.Send(tea.KeyMsg{Type: tea.KeyEscape})
	waitForActive(t, shell, 0)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("[1 Setup]"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func seedDynamicAgent(t *testing.T, name, persona string) {
	t.Helper()
	store, err := memory.OpenSqlite(config.MemoryDBPath(), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	registry, err := goncho.NewDynamicAgentRegistry(store.DB())
	if err != nil {
		t.Fatalf("NewDynamicAgentRegistry: %v", err)
	}
	if _, err := registry.Create(context.Background(), goncho.CreateAgentOptions{Name: name, Persona: persona}); err != nil {
		t.Fatalf("Create dynamic agent: %v", err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("close memory store: %v", err)
	}
}

func TestAdminChat_SendsMessageThroughFakeProvider(t *testing.T) {
	isolateHealthHome(t)
	var got ChatRequest
	chat := NewChatScreen(WithChatResponder(chatResponderFunc(func(_ context.Context, req ChatRequest) (string, error) {
		got = req
		return "canned response", nil
	})))
	shell := New(chat)
	tm := teatest.NewTestModel(t, shell, teatest.WithInitialTermSize(96, 32))

	tm.Type("check chat")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("you: check chat")) &&
			bytes.Contains(out, []byte("assistant: canned response"))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))

	if got.Prompt != "check chat" {
		t.Fatalf("responder prompt = %q, want check chat", got.Prompt)
	}
	if len(got.Messages) == 0 || got.Messages[len(got.Messages)-1].Role != "user" || got.Messages[len(got.Messages)-1].Content != "check chat" {
		t.Fatalf("responder messages = %+v, want trailing user check chat", got.Messages)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
