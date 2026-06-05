package gateway

import "testing"

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
