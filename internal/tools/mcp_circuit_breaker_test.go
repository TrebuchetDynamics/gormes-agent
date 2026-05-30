package tools

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

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
