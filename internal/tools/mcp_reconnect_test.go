package tools

import (
	"context"
	"errors"
	"testing"
)

func TestMCPServerLifecycleEventShutdownWins(t *testing.T) {
	lifecycle := NewMCPServerLifecycle()
	lifecycle.SignalReconnect()

	if got := lifecycle.NextEvent(); got != MCPLifecycleEventReconnect {
		t.Fatalf("NextEvent = %q, want %q", got, MCPLifecycleEventReconnect)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after reconnect event was consumed")
	}

	lifecycle.SignalReconnect()
	lifecycle.SignalShutdown()
	if got := lifecycle.NextEvent(); got != MCPLifecycleEventShutdown {
		t.Fatalf("NextEvent with both events = %q, want %q", got, MCPLifecycleEventShutdown)
	}
	if lifecycle.ReconnectPending() {
		t.Fatal("ReconnectPending = true after shutdown won over reconnect")
	}
}

type fakeMCPProbeSession struct {
	tools  []MCPRawTool
	err    error
	closed bool
}

func (s *fakeMCPProbeSession) ListTools(context.Context) ([]MCPRawTool, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tools, nil
}

func (s *fakeMCPProbeSession) Close() error {
	s.closed = true
	return nil
}

func TestMCPProbeSkipsDisabledAndCleansUp(t *testing.T) {
	good := &fakeMCPProbeSession{tools: []MCPRawTool{{Name: "ok", Description: "works"}}}
	broken := &fakeMCPProbeSession{err: errors.New("boom")}
	var connected []string

	result := ProbeMCPServerTools(context.Background(), []MCPServerDefinition{
		{Name: "github", Enabled: true},
		{Name: "disabled", Enabled: false},
		{Name: "broken", Enabled: true},
	}, func(_ context.Context, def MCPServerDefinition) (MCPProbeSession, error) {
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
