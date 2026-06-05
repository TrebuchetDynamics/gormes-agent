package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type fakeContextEngine struct {
	boundaryCalls []llm.CompressionBoundary
	modelUpdates  []llm.ContextModelContext
	compressErr   error
	compressCalls int
}

func (e *fakeContextEngine) Name() string                          { return "fake" }
func (e *fakeContextEngine) ToolDescriptors() []llm.ToolDescriptor { return nil }
func (e *fakeContextEngine) UpdateModelContext(update llm.ContextModelContext) {
	e.modelUpdates = append(e.modelUpdates, update)
}
func (e *fakeContextEngine) OnSessionStart(_ context.Context, _ string, _ llm.ContextSessionMeta) error {
	return nil
}
func (e *fakeContextEngine) OnSessionEnd(_ context.Context, _ string, _ []llm.Message) error {
	return nil
}
func (e *fakeContextEngine) OnSessionReset()                            {}
func (e *fakeContextEngine) UpdateFromResponse(llm.ContextUsage)        {}
func (e *fakeContextEngine) Status() llm.ContextStatus                  { return llm.ContextStatus{} }
func (e *fakeContextEngine) ShouldCompress(int) bool                    { return false }
func (e *fakeContextEngine) ShouldCompressPreflight([]llm.Message) bool { return false }
func (e *fakeContextEngine) HasContentToCompress([]llm.Message) bool    { return false }
func (e *fakeContextEngine) GetHistoryTokenEstimate() int               { return 0 }
func (e *fakeContextEngine) HandleToolCall(_ context.Context, _ string, _ json.RawMessage, _ llm.ContextToolCallOptions) (json.RawMessage, error) {
	return nil, nil
}
func (e *fakeContextEngine) Compress(_ context.Context, msgs []llm.Message, _ llm.CompressionRequest) ([]llm.Message, llm.CompressionReport, error) {
	e.compressCalls++
	if e.compressErr != nil {
		return msgs, llm.CompressionReport{}, e.compressErr
	}
	return msgs, llm.CompressionReport{}, nil
}
func (e *fakeContextEngine) OnCompressionBoundary(_ context.Context, b llm.CompressionBoundary) error {
	e.boundaryCalls = append(e.boundaryCalls, b)
	return nil
}

func TestNotifyCompressionBoundary_RecordsAfterSuccessfulCompression(t *testing.T) {
	fake := &fakeContextEngine{}
	k := &Kernel{cfg: Config{ContextEngine: fake, Model: "test-model"}, sessionID: "sess-123"}

	err := k.NotifyCompressionBoundary(context.Background(), "threshold_exceeded")
	if err != nil {
		t.Fatalf("NotifyCompressionBoundary: %v", err)
	}
	if len(fake.boundaryCalls) != 1 {
		t.Fatalf("boundary calls = %d, want 1", len(fake.boundaryCalls))
	}
	b := fake.boundaryCalls[0]
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
	fake := &fakeContextEngine{compressErr: errors.New("budget exhausted")}
	_, _, err := fake.Compress(context.Background(), nil, llm.CompressionRequest{})
	if err == nil {
		t.Fatal("expected compression error")
	}
	if len(fake.boundaryCalls) != 0 {
		t.Fatalf("boundary calls = %d, want 0 for failed compression", len(fake.boundaryCalls))
	}
}
