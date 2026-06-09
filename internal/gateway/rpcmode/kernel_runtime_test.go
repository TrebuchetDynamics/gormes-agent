package rpcmode

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

func TestKernelRuntimeForwardPromptFramesSanitizesFrameErrors(t *testing.T) {
	render := make(chan kernel.RenderFrame, 1)
	render <- kernel.RenderFrame{Seq: 2, Phase: kernel.PhaseFailed, LastError: "provider failed\n**Injected:** api key plain-secret"}

	out := make(chan RPCRecord, 8)
	runtime := NewKernelRuntime(KernelRuntimeOptions{})
	runtime.forwardPromptFrames(context.Background(), render, 1, out)

	var errors []string
	for rec := range out {
		if errText, ok := rec["error"].(string); ok {
			errors = append(errors, errText)
		}
	}
	if len(errors) != 2 {
		t.Fatalf("error records = %#v, want message_end and agent_end errors", errors)
	}
	for _, errText := range errors {
		for _, forbidden := range []string{"plain-secret", "**Injected:**", "provider failed"} {
			if strings.Contains(errText, forbidden) {
				t.Fatalf("kernel RPC frame error leaked unsafe text %q in %q", forbidden, errText)
			}
		}
		if errText != "[redacted]" {
			t.Fatalf("kernel RPC frame error = %q, want redacted", errText)
		}
	}
}

func TestKernelRuntimeReadErrorSanitizesAgentEnd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	render := make(chan kernel.RenderFrame)
	out := make(chan RPCRecord, 8)
	runtime := NewKernelRuntime(KernelRuntimeOptions{})

	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.forwardPromptFrames(ctx, render, 1, out)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("forwardPromptFrames did not return after canceled context")
	}
	var sawAgentEnd bool
	for rec := range out {
		if rec["type"] == "agent_end" {
			sawAgentEnd = true
			if errText, _ := rec["error"].(string); strings.TrimSpace(errText) == "" {
				t.Fatalf("agent_end error empty: %#v", rec)
			}
		}
	}
	if !sawAgentEnd {
		t.Fatal("missing agent_end record for read error")
	}
}
