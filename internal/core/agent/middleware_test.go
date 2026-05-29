package agent

import (
	"errors"
	"fmt"
	"testing"
)

// mockMiddleware is a test double that records its lifecycle calls.
type mockMiddleware struct {
	name      string
	beforeErr error
	afterErr  error
	calls     []string
}

func (m *mockMiddleware) Name() string { return m.name }
func (m *mockMiddleware) Before(ctx *MiddlewareContext) error {
	m.calls = append(m.calls, fmt.Sprintf("before:%s", m.name))
	return m.beforeErr
}
func (m *mockMiddleware) After(ctx *MiddlewareContext) error {
	m.calls = append(m.calls, fmt.Sprintf("after:%s", m.name))
	return m.afterErr
}

func TestMiddlewareChain_Empty(t *testing.T) {
	chain := NewMiddlewareChain()
	if len(chain.Names()) != 0 {
		t.Fatalf("expected empty chain, got %d middlewares", len(chain.Names()))
	}
	if err := chain.Before(nil); err != nil {
		t.Fatalf("Before on empty chain: %v", err)
	}
	if err := chain.After(nil); err != nil {
		t.Fatalf("After on empty chain: %v", err)
	}
}

func TestMiddlewareChain_Ordering(t *testing.T) {
	m1 := &mockMiddleware{name: "first"}
	m2 := &mockMiddleware{name: "second"}
	m3 := &mockMiddleware{name: "third"}

	chain := NewMiddlewareChain(m1, m2, m3)
	names := chain.Names()
	if len(names) != 3 || names[0] != "first" || names[1] != "second" || names[2] != "third" {
		t.Fatalf("names out of order: got %v, want [first second third]", names)
	}
}

func TestMiddlewareChain_BeforeAndAfterOrder(t *testing.T) {
	var globalCalls []string
	record := func(s string) { globalCalls = append(globalCalls, s) }

	m1 := &callRecordingMiddleware{name: "a", record: record}
	m2 := &callRecordingMiddleware{name: "b", record: record}

	chain := NewMiddlewareChain(m1, m2)
	ctx := &MiddlewareContext{}

	if err := chain.Before(ctx); err != nil {
		t.Fatalf("Before: %v", err)
	}
	if err := chain.After(ctx); err != nil {
		t.Fatalf("After: %v", err)
	}

	// Before runs first-to-last, After runs last-to-first.
	want := []string{"before:a", "before:b", "after:b", "after:a"}
	for i, call := range globalCalls {
		if call != want[i] {
			t.Fatalf("call %d: got %q, want %q", i, call, want[i])
		}
	}
}

// callRecordingMiddleware records lifecycle calls on a shared slice.
type callRecordingMiddleware struct {
	name   string
	record func(string)
}

func (m *callRecordingMiddleware) Name() string { return m.name }
func (m *callRecordingMiddleware) Before(ctx *MiddlewareContext) error {
	m.record("before:" + m.name)
	return nil
}
func (m *callRecordingMiddleware) After(ctx *MiddlewareContext) error {
	m.record("after:" + m.name)
	return nil
}

func TestMiddlewareChain_BeforeErrorAborts(t *testing.T) {
	m1 := &mockMiddleware{name: "ok"}
	m2 := &mockMiddleware{name: "fail", beforeErr: errors.New("abort")}
	m3 := &mockMiddleware{name: "never"}

	chain := NewMiddlewareChain(m1, m2, m3)
	err := chain.Before(&MiddlewareContext{})
	if err == nil || err.Error() != "abort" {
		t.Fatalf("expected abort error, got %v", err)
	}

	// m3.Before should NOT have been called
	for _, call := range m3.calls {
		t.Errorf("m3 should not have been called, got %s", call)
	}
}

func TestMiddlewareChain_Add(t *testing.T) {
	chain := NewMiddlewareChain()
	chain.Add(&mockMiddleware{name: "first"})
	chain.Add(&mockMiddleware{name: "second"})

	names := chain.Names()
	if len(names) != 2 || names[0] != "first" || names[1] != "second" {
		t.Fatalf("expected [first second], got %v", names)
	}
}

func TestLoopDetectorAdapter(t *testing.T) {
	adapter := &loopDetectAdapter{inner: NewLoopDetector()}
	var _ Middleware = adapter

	ctx := &MiddlewareContext{}
	// Before on a fresh detector should pass
	if err := adapter.Before(ctx); err != nil {
		t.Fatalf("loopDetectAdapter.Before (clean): %v", err)
	}
	// After should also pass
	if err := adapter.After(ctx); err != nil {
		t.Fatalf("loopDetectAdapter.After: %v", err)
	}
}

func TestLoopDetectorMiddleware_RecordsTurn(t *testing.T) {
	ld := NewLoopDetector()
	ld.Record(TurnRecord{Index: 0, ToolCalls: []string{"bash"}, Response: "ok", HadError: false})
	result := ld.Check()
	if result.Detected {
		t.Fatalf("single turn should not trigger loop detection: %s", result.Evidence)
	}
}

func TestRuntimeFeatures_Assembly(t *testing.T) {
	features := RuntimeFeatures{
		LoopDetect: FeatureEnabled,
		ToolError:  FeatureEnabled,
		ThreadData: FeatureEnabled,
	}

	chain := AssembleFromFeatures(features)
	names := chain.Names()

	// Expected order: ThreadData → ToolError → LoopDetect
	if len(names) < 3 {
		t.Fatalf("expected at least 3 middlewares, got %d: %v", len(names), names)
	}
	if names[0] != "thread_data" {
		t.Errorf("first middleware: got %q, want %q", names[0], "thread_data")
	}
	if names[2] != "loop_detector" {
		t.Errorf("third middleware: got %q, want %q", names[2], "loop_detector")
	}
}

func TestRuntimeFeatures_Disabled(t *testing.T) {
	features := RuntimeFeatures{
		LoopDetect: FeatureDisabled,
		ToolError:  FeatureEnabled,
		ThreadData: FeatureDisabled,
	}

	chain := AssembleFromFeatures(features)
	names := chain.Names()

	// Only ToolError should be active
	if len(names) != 1 || names[0] != "tool_error" {
		t.Fatalf("expected only [tool_error], got %v", names)
	}
}

func TestRuntimeFeatures_CustomMiddleware(t *testing.T) {
	custom := &mockMiddleware{name: "custom_loop"}
	features := RuntimeFeatures{
		LoopDetect: FeatureEnabled,
		CustomMiddleware: map[string]Middleware{
			"loop_detector": custom,
		},
	}

	chain := AssembleFromFeatures(features)
	names := chain.Names()

	found := false
	for _, n := range names {
		if n == "custom_loop" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("custom middleware not found in chain: %v", names)
	}
}

func TestMiddlewareChain_BeforeAfterContextData(t *testing.T) {
	m := &mockMiddleware{name: "ctx_test"}
	chain := NewMiddlewareChain(m)
	ctx := &MiddlewareContext{Data: map[string]any{"key": "value"}}

	chain.Before(ctx)
	chain.After(ctx)

	if m.calls[0] != "before:ctx_test" || m.calls[1] != "after:ctx_test" {
		t.Fatalf("unexpected calls: %v", m.calls)
	}
}

func TestMiddlewareChain_Dump(t *testing.T) {
	chain := NewMiddlewareChain(
		&mockMiddleware{name: "first"},
		&mockMiddleware{name: "second"},
		&mockMiddleware{name: "third"},
	)
	dump := chain.Dump()
	expected := "middleware chain: [first second third]"
	if dump != expected {
		t.Fatalf("dump: got %q, want %q", dump, expected)
	}
}
