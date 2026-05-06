package tools

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

type fakeMCPClock struct {
	now time.Time
}

func (c *fakeMCPClock) Now() time.Time {
	return c.now
}

func (c *fakeMCPClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func newTestMCPCircuitBreaker(clock *fakeMCPClock) *MCPCircuitBreaker {
	return NewMCPCircuitBreaker(MCPCircuitBreakerOptions{
		Threshold: 2,
		Cooldown:  time.Minute,
		Now:       clock.Now,
	})
}

func TestMCPCircuitBreakerShortCircuitsBeforeCooldown(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestMCPCircuitBreaker(clock)
	calls := 0
	call := func(context.Context) (MCPCallResult, error) {
		calls++
		return MCPCallResult{}, errors.New("server still broken")
	}

	for i := 0; i < 2; i++ {
		_, evidence, err := CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
		if err == nil {
			t.Fatalf("call %d err = nil, want failure", i+1)
		}
		if evidence != MCPCircuitEvidenceServerUnreachable {
			t.Fatalf("call %d evidence = %q, want %q", i+1, evidence, MCPCircuitEvidenceServerUnreachable)
		}
	}

	_, evidence, err := CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
	if !errors.Is(err, ErrMCPBreakerOpen) {
		t.Fatalf("short-circuit err = %v, want ErrMCPBreakerOpen", err)
	}
	if evidence != MCPCircuitEvidenceBreakerOpen {
		t.Fatalf("short-circuit evidence = %q, want %q", evidence, MCPCircuitEvidenceBreakerOpen)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2; breaker must not touch the session before cooldown", calls)
	}
}

func TestMCPCircuitBreakerHalfOpenSuccessCloses(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestMCPCircuitBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	clock.Advance(time.Minute + time.Second)
	calls := 0

	_, evidence, err := CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", func(context.Context) (MCPCallResult, error) {
		calls++
		return MCPCallResult{Content: []StructuredContent{{Kind: "text", Text: "ok"}}}, nil
	})
	if err != nil {
		t.Fatalf("half-open call err = %v, want nil", err)
	}
	if evidence != MCPCircuitEvidenceOK {
		t.Fatalf("half-open success evidence = %q, want %q", evidence, MCPCircuitEvidenceOK)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want one half-open probe", calls)
	}
	if got := breaker.ErrorCount("srv"); got != 0 {
		t.Fatalf("breaker count = %d, want 0 after successful probe", got)
	}
}

func TestMCPCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestMCPCircuitBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	clock.Advance(time.Minute + time.Second)
	calls := 0
	call := func(context.Context) (MCPCallResult, error) {
		calls++
		return MCPCallResult{}, errors.New("probe failed")
	}

	_, evidence, err := CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
	if err == nil {
		t.Fatal("half-open failure err = nil, want failure")
	}
	if evidence != MCPCircuitEvidenceHalfOpenFailed {
		t.Fatalf("half-open failure evidence = %q, want %q", evidence, MCPCircuitEvidenceHalfOpenFailed)
	}

	_, evidence, err = CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
	if !errors.Is(err, ErrMCPBreakerOpen) {
		t.Fatalf("post-failure err = %v, want ErrMCPBreakerOpen", err)
	}
	if evidence != MCPCircuitEvidenceBreakerOpen {
		t.Fatalf("post-failure evidence = %q, want %q", evidence, MCPCircuitEvidenceBreakerOpen)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want only the half-open probe to touch the session", calls)
	}
}

func TestMCPReconnectResetClearsBreakerOnRecoveredOAuth(t *testing.T) {
	clock := &fakeMCPClock{now: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)}
	breaker := newTestMCPCircuitBreaker(clock)
	breaker.RecordFailure("srv", errors.New("first"))
	breaker.RecordFailure("srv", errors.New("second"))
	breaker.RecordFailure("srv", errors.New("third"))

	if evidence := breaker.ResetAfterReconnect("srv"); evidence != MCPCircuitEvidenceReconnectReset {
		t.Fatalf("ResetAfterReconnect evidence = %q, want %q", evidence, MCPCircuitEvidenceReconnectReset)
	}

	calls := 0
	call := func(context.Context) (MCPCallResult, error) {
		calls++
		return MCPCallResult{}, errors.New("retry still failed")
	}
	_, evidence, err := CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
	if err == nil {
		t.Fatal("post-reconnect retry err = nil, want failure")
	}
	if evidence != MCPCircuitEvidenceServerUnreachable {
		t.Fatalf("post-reconnect evidence = %q, want %q", evidence, MCPCircuitEvidenceServerUnreachable)
	}
	if got := breaker.ErrorCount("srv"); got != 1 {
		t.Fatalf("post-reconnect count = %d, want fresh count 1", got)
	}

	_, _, _ = CallMCPWithCircuitBreaker(context.Background(), breaker, "srv", call)
	if calls != 2 {
		t.Fatalf("calls = %d, want retry failure below threshold to leave the next call open", calls)
	}
}

func TestMCPCircuitBreakerManagedGatewayEvidence(t *testing.T) {
	toolCalls := 0
	srv := newFakeManagedGatewayServer(t, func(t *testing.T, w http.ResponseWriter, r *http.Request, req fakeManagedGatewayRequest) {
		switch req.Method {
		case "initialize":
			writeManagedJSONResult(w, req.ID, `{"protocolVersion":"2024-11-05","capabilities":{}}`)
		case "tools/call":
			toolCalls++
			http.Error(w, "down", http.StatusServiceUnavailable)
		default:
			t.Errorf("unexpected method %q", req.Method)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	})
	bridge := newTestManagedGatewayBridge(t, "firecrawl", srv, "token")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := bridge.Initialize(ctx); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	for i := 0; i < defaultMCPCircuitBreakerThreshold; i++ {
		_, evidence, err := bridge.CallTool(ctx, "web_search", map[string]any{"q": "x"})
		if err == nil {
			t.Fatalf("CallTool %d err = nil, want failure", i+1)
		}
		if evidence != ManagedGatewayEvidenceToolCallFailed {
			t.Fatalf("CallTool %d evidence = %q, want %q", i+1, evidence, ManagedGatewayEvidenceToolCallFailed)
		}
	}

	_, evidence, err := bridge.CallTool(ctx, "web_search", map[string]any{"q": "x"})
	if !errors.Is(err, ErrMCPBreakerOpen) {
		t.Fatalf("short-circuit err = %v, want ErrMCPBreakerOpen", err)
	}
	if evidence != ManagedGatewayEvidenceMCPBreakerOpen {
		t.Fatalf("short-circuit evidence = %q, want %q", evidence, ManagedGatewayEvidenceMCPBreakerOpen)
	}
	if toolCalls != defaultMCPCircuitBreakerThreshold {
		t.Fatalf("toolCalls = %d, want %d; breaker should stop touching the gateway", toolCalls, defaultMCPCircuitBreakerThreshold)
	}
}
