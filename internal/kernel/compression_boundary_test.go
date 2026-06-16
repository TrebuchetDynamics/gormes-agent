package kernel

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel/testfixtures"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestNotifyCompressionBoundary_RecordsAfterSuccessfulCompression(t *testing.T) {
	fake := &testfixtures.ContextEngine{}
	k := &Kernel{cfg: Config{ContextEngine: fake, Model: "test-model"}, sessionID: "sess-123"}

	err := k.NotifyCompressionBoundary(context.Background(), "threshold_exceeded")
	if err != nil {
		t.Fatalf("NotifyCompressionBoundary: %v", err)
	}
	if len(fake.BoundaryCalls) != 1 {
		t.Fatalf("boundary calls = %d, want 1", len(fake.BoundaryCalls))
	}
	b := fake.BoundaryCalls[0]
	if b.OldSessionID != "sess-123" || b.NewSessionID != "sess-123" || b.Reason != "threshold_exceeded" {
		t.Errorf("boundary = %+v, want sess-123/threshold_exceeded", b)
	}
}

func TestNotifyCompressionBoundary_SkipsWhenEngineIsNil(t *testing.T) {
	k := &Kernel{cfg: Config{ContextEngine: nil}}
	err := k.NotifyCompressionBoundary(context.Background(), "reason")
	if err != nil {
		t.Fatalf("expected nil error for nil engine, got %v", err)
	}
}

func TestCompressionBoundary_SkipOnFailedCompression(t *testing.T) {
	fake := &testfixtures.ContextEngine{CompressErr: errors.New("budget exhausted")}
	_, _, err := fake.Compress(context.Background(), nil, llm.CompressionRequest{})
	if err == nil {
		t.Fatal("expected compression error")
	}
	if len(fake.BoundaryCalls) != 0 {
		t.Fatalf("boundary calls = %d, want 0 for failed compression", len(fake.BoundaryCalls))
	}
}
