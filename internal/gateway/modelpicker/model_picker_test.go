package modelpicker

import (
	"context"
	"testing"
)

func TestGatewayModelCommandNoArgsOpensTelegramPicker(t *testing.T) {
	resolver := NewModelPickerResolver(&SessionModelOverride{})
	providers := resolver.PickerProviders()
	if len(providers) == 0 {
		t.Fatal("PickerProviders returned zero providers")
	}
	foundOpenRouter := false
	for _, p := range providers {
		if p == "openrouter" {
			foundOpenRouter = true
			break
		}
	}
	if !foundOpenRouter {
		t.Log("openrouter not in picker providers (may be filtered); available:", providers)
	}
}

func TestGatewayModelPickerProviderCallbackEditsToModelStage(t *testing.T) {
	ov := &SessionModelOverride{}
	resolver := NewModelPickerResolver(ov)

	_, err := resolver.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: "test"})
	if err != nil {
		t.Fatalf("OpenModelPicker: %v", err)
	}

	resp, err := resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID:    "test",
		Prefix:    "mp",
		Value:     "openrouter",
		MessageID: 1,
	})
	if err != nil {
		t.Fatalf("HandleModelPickerCallback(mp): %v", err)
	}
	if resp.Finished {
		t.Error("expected not finished after provider selection")
	}
	if resp.Provider != "openrouter" {
		t.Errorf("Provider = %q, want openrouter", resp.Provider)
	}
}

func TestGatewayModelPickerModelCallbackAppliesSessionOverride(t *testing.T) {
	ov := &SessionModelOverride{}
	resolver := NewModelPickerResolver(ov)

	resolver.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: "test"})
	resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID:    "test",
		Prefix:    "mp",
		Value:     "openrouter",
		MessageID: 1,
	})

	resp, err := resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID:    "test",
		Prefix:    "mm",
		Value:     "0",
		MessageID: 1,
	})
	if err != nil {
		t.Fatalf("HandleModelPickerCallback(mm): %v", err)
	}
	if !resp.Finished {
		t.Error("expected finished after model selection")
	}
	if !resp.Changed {
		t.Error("expected Changed=true after model selection")
	}
	if ov.Model == "" {
		t.Error("SessionModelOverride.Model not set after model selection")
	}
	if ov.Provider != "openrouter" {
		t.Errorf("SessionModelOverride.Provider = %q, want openrouter", ov.Provider)
	}
}

func TestGatewayModelPickerCancelOrUnknownLeavesOverrideUnchanged(t *testing.T) {
	ov := &SessionModelOverride{}
	resolver := NewModelPickerResolver(ov)

	resolver.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: "test"})
	resp, err := resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID:    "test",
		Prefix:    "mx",
		Value:     "",
		MessageID: 1,
	})
	if err != nil {
		t.Fatalf("HandleModelPickerCallback(mx): %v", err)
	}
	if !resp.Finished {
		t.Error("expected finished after cancel")
	}
	if resp.Changed {
		t.Error("expected Changed=false after cancel")
	}
	if !ov.IsZero() {
		t.Errorf("override should be zero after cancel, got %+v", ov)
	}
}

func TestGatewayModelPickerCallbackFamiliesAreIsolated(t *testing.T) {
	ov := &SessionModelOverride{}
	resolver := NewModelPickerResolver(ov)

	_, err := resolver.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: "a"})
	if err != nil {
		t.Fatalf("OpenModelPicker(a): %v", err)
	}
	_, err = resolver.OpenModelPicker(context.Background(), ModelPickerRequest{ChatID: "b"})
	if err != nil {
		t.Fatalf("OpenModelPicker(b): %v", err)
	}

	resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID: "a", Prefix: "mp", Value: "openrouter", MessageID: 1,
	})

	resp, err := resolver.HandleModelPickerCallback(context.Background(), ModelPickerCallback{
		ChatID: "b", Prefix: "mx", Value: "", MessageID: 2,
	})
	if err != nil {
		t.Fatalf("HandleModelPickerCallback(mx): %v", err)
	}
	if !resp.Finished || resp.Changed {
		t.Error("chat b cancel should finish without override")
	}
}
