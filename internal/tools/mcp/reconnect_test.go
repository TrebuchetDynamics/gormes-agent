package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestServerLifecycleEventShutdownWins(t *testing.T) {
	lifecycle := NewServerLifecycle()
	lifecycle.SignalReconnect()

	if got := lifecycle.NextEvent(); got != LifecycleEventReconnect {
		t.Fatalf("NextEvent = %q, want %q", got, LifecycleEventReconnect)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after reconnect event was consumed")
	}

	lifecycle.SignalReconnect()
	lifecycle.SignalShutdown()
	if got := lifecycle.NextEvent(); got != LifecycleEventShutdown {
		t.Fatalf("NextEvent with both events = %q, want %q", got, LifecycleEventShutdown)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after shutdown won over reconnect")
	}
}

type fakeProbeSession struct {
	tools  []RawTool
	err    error
	closed bool
}

func (s *fakeProbeSession) ListTools(context.Context) ([]RawTool, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tools, nil
}

func (s *fakeProbeSession) Close() error {
	s.closed = true
	return nil
}

func TestMCPProbeSkipsDisabledAndCleansUp(t *testing.T) {
	good := &fakeProbeSession{tools: []RawTool{{Name: "ok", Description: "works"}}}
	broken := &fakeProbeSession{err: errors.New("boom")}
	var connected []string

	result := ProbeServerTools(context.Background(), []MCPServerDefinition{
		{Name: "github", Enabled: true},
		{Name: "disabled", Enabled: false},
		{Name: "broken", Enabled: true},
	}, func(_ context.Context, def MCPServerDefinition) (ProbeSession, error) {
		connected = append(connected, def.Name)
		switch def.Name {
		case "github":
			return good, nil
		case "broken":
			return broken, nil
		default:
			t.Fatalf("unexpected connect to %q", def.Name)
			return nil, nil
		}
	})

	if _, ok := result["disabled"]; ok {
		t.Fatalf("disabled server was probed: %+v", result)
	}
	if _, ok := result["broken"]; ok {
		t.Fatalf("failed server appeared in result: %+v", result)
	}
	if got := len(result["github"]); got != 1 {
		t.Fatalf("github tools len = %d, want 1; result=%+v", got, result)
	}
	if len(connected) != 2 || connected[0] != "github" || connected[1] != "broken" {
		t.Fatalf("connected = %+v, want github and broken only", connected)
	}
	if !good.closed {
		t.Fatal("successful probe session was not closed")
	}
	if !broken.closed {
		t.Fatal("failed probe session was not closed")
	}
}

func TestMCPProbeKeepsFirstDuplicateServerProvenance(t *testing.T) {
	var connected []string

	result := ProbeServerTools(context.Background(), []MCPServerDefinition{
		{Name: "github", Enabled: true, Command: "first"},
		{Name: "github", Enabled: true, Command: "second"},
	}, func(_ context.Context, def MCPServerDefinition) (ProbeSession, error) {
		connected = append(connected, def.Command)
		return &fakeProbeSession{tools: []RawTool{{Name: def.Command}}}, nil
	})

	if len(connected) != 1 || connected[0] != "first" {
		t.Fatalf("connected = %+v, want only first duplicate server definition", connected)
	}
	tools := result["github"]
	if len(tools) != 1 || tools[0].Name != "first" {
		t.Fatalf("github tools = %+v, want first duplicate provenance", tools)
	}
}

func TestMCPProbeCopiesRawToolSchemas(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	session := &fakeProbeSession{tools: []RawTool{{Name: "ok", InputSchema: schema}}}

	result := ProbeServerTools(context.Background(), []MCPServerDefinition{{Name: "github", Enabled: true}}, func(context.Context, MCPServerDefinition) (ProbeSession, error) {
		return session, nil
	})
	schema[0] = '['
	session.tools[0].InputSchema[1] = 'X'

	tools := result["github"]
	if len(tools) != 1 {
		t.Fatalf("github tools len = %d, want 1", len(tools))
	}
	if string(tools[0].InputSchema) != `{"type":"object"}` {
		t.Fatalf("probe result schema aliases session buffer: %s", tools[0].InputSchema)
	}
}
