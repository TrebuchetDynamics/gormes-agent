package gateway

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type streamConsumerCompatibilitySink struct {
	target DeliveryTarget
	called bool
}

func (s *streamConsumerCompatibilitySink) DeliverFrame(_ context.Context, target DeliveryTarget, _ kernel.RenderFrame) error {
	s.target = target
	s.called = true
	return nil
}

func TestGatewayStreamConsumerCompatibilityWrapper(t *testing.T) {
	sink := &streamConsumerCompatibilitySink{}
	consumer := NewGatewayStreamConsumer(sink)
	target := DeliveryTarget{Platform: "telegram", ChatID: "42", IsExplicit: true}
	results := consumer.FanOut(context.Background(), kernel.RenderFrame{Phase: kernel.PhaseIdle}, []DeliveryTarget{target})
	if !sink.called || sink.target != target {
		t.Fatalf("wrapper sink call = called %v target %+v, want %+v", sink.called, sink.target, target)
	}
	if len(results) != 1 || results[0].Target != target || results[0].Err != nil {
		t.Fatalf("wrapper results = %+v, want successful target", results)
	}
}
