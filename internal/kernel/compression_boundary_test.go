package kernel

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

type fakeContextEngine struct {
	boundaryCalls []hermes.CompressionBoundary
	compressErr   error
	compressCalls int
}

func (e *fakeContextEngine) Name() string                                  { return "fake" }
func (e *fakeContextEngine) ToolDescriptors() []hermes.ToolDescriptor      { return nil }
func (e *fakeContextEngine) UpdateModelContext(hermes.ContextModelContext) {}
func (e *fakeContextEngine) OnSessionStart(_ context.Context, _ string, _ hermes.ContextSessionMeta) error {
	return nil
}
func (e *fakeContextEngine) OnSessionEnd(_ context.Context, _ string, _ []hermes.Message) error {
	return nil
}
func (e *fakeContextEngine) OnSessionReset()                               {}
func (e *fakeContextEngine) UpdateFromResponse(hermes.ContextUsage)        {}
func (e *fakeContextEngine) Status() hermes.ContextStatus                  { return hermes.ContextStatus{} }
func (e *fakeContextEngine) ShouldCompress(int) bool                       { return false }
func (e *fakeContextEngine) ShouldCompressPreflight([]hermes.Message) bool { return false }
func (e *fakeContextEngine) HasContentToCompress([]hermes.Message) bool    { return false }
func (e *fakeContextEngine) GetHistoryTokenEstimate() int                  { return 0 }
func (e *fakeContextEngine) HandleToolCall(_ context.Context, _ string, _ json.RawMessage, _ hermes.ContextToolCallOptions) (json.RawMessage, error) {
	return nil, nil
}
func (e *fakeContextEngine) Compress(_ context.Context, msgs []hermes.Message, _ hermes.CompressionRequest) ([]hermes.Message, hermes.CompressionReport, error) {
	e.compressCalls++
	if e.compressErr != nil {
		return msgs, hermes.CompressionReport{}, e.compressErr
	}
	return msgs, hermes.CompressionReport{}, nil
}
func (e *fakeContextEngine) OnCompressionBoundary(_ context.Context, b hermes.CompressionBoundary) error {
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
	_, _, err := fake.Compress(context.Background(), nil, hermes.CompressionRequest{})
	if err == nil {
		t.Fatal("expected compression error")
	}
	if len(fake.boundaryCalls) != 0 {
		t.Fatalf("boundary calls = %d, want 0 for failed compression", len(fake.boundaryCalls))
	}
}
