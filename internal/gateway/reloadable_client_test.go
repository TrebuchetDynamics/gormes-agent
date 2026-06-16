package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestReloadableHermesClientTypedNilIsUnavailable(t *testing.T) {
	var client *ReloadableHermesClient
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("typed nil reloadable client panicked: %v", r)
		}
	}()

	if _, err := client.OpenStream(context.Background(), llm.ChatRequest{}); err == nil {
		t.Fatal("OpenStream typed nil err = nil, want unavailable error")
	}
	if _, err := client.OpenRunEvents(context.Background(), "run-1"); err == nil {
		t.Fatal("OpenRunEvents typed nil err = nil, want unavailable error")
	}
	if err := client.Health(context.Background()); err == nil {
		t.Fatal("Health typed nil err = nil, want unavailable error")
	}
	status := llm.ProviderStatusOf(client)
	if status.Provider != "unknown" || status.Runtime != "unknown" {
		t.Fatalf("ProviderStatusOf typed nil = %+v, want unknown provider status", status)
	}
}

func TestReloadableHermesClientHonorsCanceledContextBeforeDelegating(t *testing.T) {
	current := llm.NewMockClient()
	client := NewReloadableHermesClient(current)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.OpenStream(ctx, llm.ChatRequest{SessionID: "sess-canceled"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenStream canceled err = %v, want context.Canceled", err)
	}
	if got := len(current.Requests()); got != 0 {
		t.Fatalf("OpenStream delegated after cancellation; requests=%d", got)
	}
	if _, err := client.OpenRunEvents(ctx, "run-canceled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenRunEvents canceled err = %v, want context.Canceled", err)
	}
	if err := client.Health(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Health canceled err = %v, want context.Canceled", err)
	}
}

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
