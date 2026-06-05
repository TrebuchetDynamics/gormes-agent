package webcontent

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

type fakeClient struct{}

func (fakeClient) OpenStream(context.Context, llm.ChatRequest) (llm.Stream, error)   { return nil, nil }
func (fakeClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) { return nil, nil }
func (fakeClient) Health(context.Context) error                                      { return nil }

func TestNewProcessorReturnsProcessor(t *testing.T) {
	processor := NewProcessor(fakeClient{}, "fixture-model")
	if processor == nil {
		t.Fatal("NewProcessor returned nil")
	}
}
