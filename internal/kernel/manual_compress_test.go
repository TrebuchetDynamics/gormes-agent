package kernel

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/testfixtures"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

func TestKernelManualCompressRewritesHistoryAndRecordsBoundary(t *testing.T) {
	engine := &testfixtures.ContextEngine{CompressResult: []llm.Message{{Role: "user", Content: llm.ContextPruningSummaryPrefix + "\nsummary"}, {Role: "user", Content: "latest"}}}
	k := New(Config{Model: "test-model", ContextEngine: engine}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	history := []llm.Message{{Role: "user", Content: "old"}, {Role: "assistant", Content: "answer"}, {Role: "user", Content: "latest"}}
	if err := k.ResumeSession("sess-compress", history); err != nil {
		t.Fatalf("ResumeSession: %v", err)
	}
	if err := k.ManualCompress("billing context"); err != nil {
		t.Fatalf("ManualCompress: %v", err)
	}

	frame := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.StatusText == "session compressed"
	}, time.Second)
	if len(frame.History) != 2 || !strings.Contains(frame.History[0].Content, "summary") {
		t.Fatalf("compressed history = %#v", frame.History)
	}
	if engine.CompressCalls != 1 || engine.CompressRequest.FocusTopic != "billing context" {
		t.Fatalf("compress calls=%d request=%+v", engine.CompressCalls, engine.CompressRequest)
	}
	if len(engine.BoundaryCalls) != 1 || engine.BoundaryCalls[0].OldSessionID != "sess-compress" || engine.BoundaryCalls[0].Reason != "manual_compress" {
		t.Fatalf("boundary calls = %#v", engine.BoundaryCalls)
	}
}

func TestKernelManualCompressRequiresContextEngine(t *testing.T) {
	k := New(Config{Model: "test-model"}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	<-k.Render()

	if err := k.ManualCompress(""); !errors.Is(err, ErrCompressionUnavailable) {
		t.Fatalf("ManualCompress error = %v, want %v", err, ErrCompressionUnavailable)
	}
}
