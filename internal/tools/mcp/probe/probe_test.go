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
	tools    []descriptor.RawTool
	err      error
	closeErr error
	closed   bool
}

func (s *fakeSession) ListTools(context.Context) ([]descriptor.RawTool, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.tools, nil
}

func (s *fakeSession) Close() error {
	s.closed = true
	return s.closeErr
}

func TestOnePreservesErrorsClosesAndCopiesResult(t *testing.T) {
	schema := json.RawMessage(`{"type":"object"}`)
	session := &fakeSession{tools: []descriptor.RawTool{{Name: "ok", InputSchema: schema}}}
	original := config.MCPServerDefinition{Name: "srv", Enabled: true, Headers: map[string]string{"Authorization": "secret"}}
	tools, err := One(context.Background(), original, func(_ context.Context, def config.MCPServerDefinition) (Session, error) {
		def.Headers["Authorization"] = "mutated"
		return session, nil
	})
	if err != nil {
		t.Fatalf("One: %v", err)
	}
	if !session.closed || original.Headers["Authorization"] != "secret" || len(tools) != 1 {
		t.Fatalf("closed=%v original=%+v tools=%+v", session.closed, original, tools)
	}
	schema[0] = '['
	if string(tools[0].InputSchema) != `{"type":"object"}` {
		t.Fatalf("schema aliases session: %s", tools[0].InputSchema)
	}

	wantClose := errors.New("close failed")
	_, err = One(context.Background(), original, func(context.Context, config.MCPServerDefinition) (Session, error) {
		return &fakeSession{closeErr: wantClose}, nil
	})
	if !errors.Is(err, wantClose) {
		t.Fatalf("close error = %v", err)
	}
	if _, err := One(context.Background(), original, nil); !errors.Is(err, ErrConnectorUnavailable) {
		t.Fatalf("nil connector error = %v", err)
	}
	wantConnect := errors.New("connect failed")
	if _, err := One(context.Background(), original, func(context.Context, config.MCPServerDefinition) (Session, error) {
		return nil, wantConnect
	}); !errors.Is(err, wantConnect) {
		t.Fatalf("connect error = %v", err)
	}
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

func TestServerToolsPassesConnectorIndependentServerDefinition(t *testing.T) {
	servers := []config.MCPServerDefinition{{
		Name:    "github",
		Enabled: true,
		Args:    []string{"serve"},
		Env:     map[string]string{"TOKEN": "secret"},
		Headers: map[string]string{"Authorization": "Bearer secret"},
		Sampling: config.MCPSamplingConfig{
			AllowedModels: []string{"gpt-4o"},
		},
	}}

	_ = ServerTools(context.Background(), servers, func(_ context.Context, def config.MCPServerDefinition) (Session, error) {
		def.Args[0] = "tampered"
		def.Env["TOKEN"] = "tampered"
		def.Headers["Authorization"] = "tampered"
		def.Sampling.AllowedModels[0] = "tampered"
		return &fakeSession{tools: []descriptor.RawTool{{Name: "ok"}}}, nil
	})

	if servers[0].Args[0] != "serve" {
		t.Fatalf("server args mutated by connector: %+v", servers[0].Args)
	}
	if servers[0].Env["TOKEN"] != "secret" {
		t.Fatalf("server env mutated by connector: %+v", servers[0].Env)
	}
	if servers[0].Headers["Authorization"] != "Bearer secret" {
		t.Fatalf("server headers mutated by connector: %+v", servers[0].Headers)
	}
	if servers[0].Sampling.AllowedModels[0] != "gpt-4o" {
		t.Fatalf("server sampling models mutated by connector: %+v", servers[0].Sampling.AllowedModels)
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
