package gateway

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestReloadableHermesClientForwardsProviderStatus(t *testing.T) {
	current := llm.NewMockClient()
	current.SetProviderStatus(llm.ProviderStatus{Provider: "openrouter", Runtime: "chat_completions"})
	client := NewReloadableHermesClient(current)

	status := llm.ProviderStatusOf(client)
	if status.Provider != "openrouter" || status.Runtime != "chat_completions" {
		t.Fatalf("ProviderStatusOf(reloadable) = %+v, want current provider status", status)
	}

	next := llm.NewMockClient()
	next.SetProviderStatus(llm.ProviderStatus{Provider: "anthropic", Runtime: "anthropic_messages"})
	client.Set(next)
	status = llm.ProviderStatusOf(client)
	if status.Provider != "anthropic" || status.Runtime != "anthropic_messages" {
		t.Fatalf("ProviderStatusOf(reloadable after Set) = %+v, want updated provider status", status)
	}
}
