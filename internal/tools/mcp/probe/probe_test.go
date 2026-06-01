package probe

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/mcp/descriptor"
)

type fakeSession struct {
	tools  []descriptor.RawTool
	err    error
	closed bool
}

func (s *fakeSession) ListTools(context.Context) ([]descriptor.RawTool, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tools, nil
}

func (s *fakeSession) Close() error {
	s.closed = true
	return nil
}

func TestServerToolsSkipsDisabledAndCleansUp(t *testing.T) {
	good := &fakeSession{tools: []descriptor.RawTool{{Name: "ok", Description: "works"}}}
	broken := &fakeSession{err: errors.New("boom")}
	var connected []string

	result := ServerTools(context.Background(), []config.MCPServerDefinition{
		{Name: "github", Enabled: true},
		{Name: "disabled", Enabled: false},
		{Name: "broken", Enabled: true},
	}, func(_ context.Context, def config.MCPServerDefinition) (Session, error) {
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

func TestServerToolsKeepsFirstDuplicateServerProvenance(t *testing.T) {
	var connected []string

	result := ServerTools(context.Background(), []config.MCPServerDefinition{
		{Name: "github", Enabled: true, Command: "first"},
		{Name: "github", Enabled: true, Command: "second"},
	}, func(_ context.Context, def config.MCPServerDefinition) (Session, error) {
		connected = append(connected, def.Command)
		return &fakeSession{tools: []descriptor.RawTool{{Name: def.Command}}}, nil
	})

	if len(connected) != 1 || connected[0] != "first" {
		t.Fatalf("connected = %+v, want only first duplicate server definition", connected)
	}
	tools := result["github"]
	if len(tools) != 1 || tools[0].Name != "first" {
		t.Fatalf("github tools = %+v, want first duplicate provenance", tools)
	}
}

func TestServerToolsCopiesRawToolSchemas(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	session := &fakeSession{tools: []descriptor.RawTool{{Name: "ok", InputSchema: schema}}}

	result := ServerTools(context.Background(), []config.MCPServerDefinition{{Name: "github", Enabled: true}}, func(context.Context, config.MCPServerDefinition) (Session, error) {
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
