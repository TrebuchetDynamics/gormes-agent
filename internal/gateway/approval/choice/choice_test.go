package choice

import (
	"context"
	"testing"
)

func TestResolverFuncAllowsNilContextAndIsolatesEvidence(t *testing.T) {
	evidence := map[string]string{"callback": "telegram"}
	resolver := ResolverFunc(func(ctx context.Context, res Resolution) error {
		if ctx == nil {
			panic("nil context")
		}
		res.Evidence["callback"] = "mutated"
		res.Evidence["new"] = "value"
		return nil
	})

	if err := resolver.ResolveGatewayApproval(nil, Resolution{Evidence: evidence}); err != nil {
		t.Fatalf("ResolveGatewayApproval error = %v, want nil", err)
	}
	if got := evidence["callback"]; got != "telegram" {
		t.Fatalf("caller evidence callback = %q, want isolated telegram", got)
	}
	if _, ok := evidence["new"]; ok {
		t.Fatalf("resolver mutation leaked into caller evidence: %+v", evidence)
	}
}
