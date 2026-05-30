package routing

import (
	"context"
	"testing"
)

func TestCapabilityRouter_SimpleQuery(t *testing.T) {
	r := NewCapabilityRouter([]string{"cheap-model"}, []string{"expensive-model"})
	result := r.Route("hello how are you")
	if result != "cheap-model" {
		t.Fatalf("simple query routed to %q, want cheap-model", result)
	}
}

func TestCapabilityRouter_ComplexQuery(t *testing.T) {
	r := NewCapabilityRouter([]string{"cheap-model"}, []string{"expensive-model"})
	result := r.Route("refactor the entire system architecture and rewrite multi-file modules")
	if result != "expensive-model" {
		t.Fatalf("complex query routed to %q, want expensive-model", result)
	}
}

func TestCapabilityRouter_LongQuery(t *testing.T) {
	r := NewCapabilityRouter([]string{"cheap-model"}, []string{"expensive-model"})
	longPrompt := ""
	for i := 0; i < 51; i++ {
		longPrompt += "word "
	}
	result := r.Route(longPrompt)
	if result != "expensive-model" {
		t.Fatalf("long query routed to %q, want expensive-model", result)
	}
}

func TestCapabilityRouter_ContextPropagation(t *testing.T) {
	r := NewCapabilityRouter([]string{"cheap"}, []string{"capable"})
	ctx := WithCapabilityRouter(context.Background(), r)
	got := GetCapabilityRouter(ctx)
	if got == nil {
		t.Fatal("capability router not retrieved from context")
	}
}
