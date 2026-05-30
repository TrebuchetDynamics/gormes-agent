package lifecycle

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestMemoryProviderLifecycle_InitializesPrefetchesAndSyncsInOrder(t *testing.T) {
	ctx := context.Background()
	builtin := &recordingLifecycleProvider{name: "builtin", prefetch: "builtin memory"}
	goncho := &recordingLifecycleProvider{name: "goncho", prefetch: "goncho memory", precompress: "goncho compression"}

	adapter := NewMemoryProviderLifecycle()
	adapter.Register(builtin)
	adapter.Register(goncho)

	if evidence := adapter.Initialize(ctx, MemoryProviderSession{SessionID: "session-1", Platform: "telegram", Model: "gpt-5-codex"}); len(evidence) != 0 {
		t.Fatalf("Initialize evidence = %#v, want none", evidence)
	}
	// Idempotent per provider/session: repeated initialization does not call providers again.
	adapter.Initialize(ctx, MemoryProviderSession{SessionID: "session-1", Platform: "telegram", Model: "gpt-5-codex"})

	block, evidence := adapter.Prefetch(ctx, MemoryPrefetchRequest{Query: "remember Juan", SessionID: "session-1"})
	if len(evidence) != 0 {
		t.Fatalf("Prefetch evidence = %#v, want none", evidence)
	}
	wantBlock := "builtin memory\n\ngoncho memory"
	if block != wantBlock {
		t.Fatalf("Prefetch block = %q, want %q", block, wantBlock)
	}

	if evidence := adapter.SyncTurn(ctx, MemoryTurn{SessionID: "session-1", User: "hi", Assistant: "hello"}); len(evidence) != 0 {
		t.Fatalf("SyncTurn evidence = %#v, want none", evidence)
	}

	contribution, evidence := adapter.PreCompress(ctx, []MemoryProviderMessage{{Role: "user", Content: "keep this"}})
	if len(evidence) != 0 {
		t.Fatalf("PreCompress evidence = %#v, want none", evidence)
	}
	if contribution != "goncho compression" {
		t.Fatalf("PreCompress contribution = %q, want goncho compression", contribution)
	}

	adapter.MemoryWrite(ctx, MemoryWriteEvent{Action: "add", Target: "memory", Content: "Juan prefers direct evidence"})
	adapter.Delegation(ctx, MemoryDelegationEvent{Task: "subtask", Result: "done", ChildSessionID: "child-1"})
	adapter.Shutdown(ctx)

	wantBuiltin := []string{"initialize:session-1", "prefetch:remember Juan", "sync:hi->hello", "precompress:1", "memory_write:add:memory", "delegation:subtask:child-1", "shutdown"}
	if !reflect.DeepEqual(builtin.calls, wantBuiltin) {
		t.Fatalf("builtin calls = %#v, want %#v", builtin.calls, wantBuiltin)
	}
	wantGoncho := []string{"initialize:session-1", "prefetch:remember Juan", "sync:hi->hello", "precompress:1", "memory_write:add:memory", "delegation:subtask:child-1", "shutdown"}
	if !reflect.DeepEqual(goncho.calls, wantGoncho) {
		t.Fatalf("goncho calls = %#v, want %#v", goncho.calls, wantGoncho)
	}
}

func TestMemoryProviderLifecycle_DegradesProviderFailuresWithoutBlocking(t *testing.T) {
	ctx := context.Background()
	bad := &recordingLifecycleProvider{name: "goncho", prefetchErr: errors.New("db password plain-token-123 unavailable")}
	good := &recordingLifecycleProvider{name: "builtin", prefetch: "builtin survives"}
	adapter := NewMemoryProviderLifecycle()
	adapter.Register(bad)
	adapter.Register(good)

	block, evidence := adapter.Prefetch(ctx, MemoryPrefetchRequest{Query: "q", SessionID: "s"})
	if block != "builtin survives" {
		t.Fatalf("Prefetch block = %q, want builtin survives", block)
	}
	if len(evidence) != 1 {
		t.Fatalf("evidence count = %d, want 1 (%#v)", len(evidence), evidence)
	}
	got := evidence[0]
	if got.Provider != "goncho" || got.Operation != "prefetch" || got.Code != "memory_provider_unavailable" || !got.Degraded {
		t.Fatalf("unexpected evidence: %#v", got)
	}
	if strings.Contains(got.Message, "plain-token-123") || strings.Contains(got.Message, "password") {
		t.Fatalf("evidence leaked sensitive error text: %#v", got)
	}
}

type recordingLifecycleProvider struct {
	name        string
	prefetch    string
	precompress string
	prefetchErr error
	calls       []string
}

func (p *recordingLifecycleProvider) Name() string { return p.name }
func (p *recordingLifecycleProvider) Initialize(_ context.Context, session MemoryProviderSession) error {
	p.calls = append(p.calls, "initialize:"+session.SessionID)
	return nil
}
func (p *recordingLifecycleProvider) Prefetch(_ context.Context, req MemoryPrefetchRequest) (string, error) {
	p.calls = append(p.calls, "prefetch:"+req.Query)
	if p.prefetchErr != nil {
		return "", p.prefetchErr
	}
	return p.prefetch, nil
}
func (p *recordingLifecycleProvider) SyncTurn(_ context.Context, turn MemoryTurn) error {
	p.calls = append(p.calls, "sync:"+turn.User+"->"+turn.Assistant)
	return nil
}
func (p *recordingLifecycleProvider) PreCompress(_ context.Context, messages []MemoryProviderMessage) (string, error) {
	p.calls = append(p.calls, "precompress:"+string(rune('0'+len(messages))))
	return p.precompress, nil
}
func (p *recordingLifecycleProvider) MemoryWrite(_ context.Context, event MemoryWriteEvent) error {
	p.calls = append(p.calls, "memory_write:"+event.Action+":"+event.Target)
	return nil
}
func (p *recordingLifecycleProvider) Delegation(_ context.Context, event MemoryDelegationEvent) error {
	p.calls = append(p.calls, "delegation:"+event.Task+":"+event.ChildSessionID)
	return nil
}
func (p *recordingLifecycleProvider) Shutdown(_ context.Context) error {
	p.calls = append(p.calls, "shutdown")
	return nil
}
