package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/provider"
)

// coldStartBudget is the maximum acceptable time for a cold-start provider
// selection benchmark on the developer baseline machine. It prevents
// regressions from eager construction of unselected providers.
//
// Budget: 5 ms — a lazy provider pool with 10 registered factories should
// construct only the selected provider in a few ms. Eagerly constructing
// all 10 would exceed this budget by several multiples (HTTP client init,
// TLS config, etc.).
const coldStartBudget = 5 * time.Millisecond

// slowFactory simulates a provider constructor that takes meaningful time
// (e.g. TLS config, credential resolution). We use a small sleep so the
// benchmark can distinguish lazy from eager construction.
func slowFactory(name string) provider.ClientFactory {
	return func() (llm.Client, error) {
		time.Sleep(50 * time.Microsecond)
		return &stubRuntimeClient{name: name}, nil
	}
}

type stubRuntimeClient struct {
	name string
}

func (s *stubRuntimeClient) OpenStream(_ context.Context, _ llm.ChatRequest) (llm.Stream, error) {
	return nil, nil
}
func (s *stubRuntimeClient) OpenRunEvents(_ context.Context, _ string) (llm.RunEventStream, error) {
	return nil, nil
}
func (s *stubRuntimeClient) Health(_ context.Context) error { return nil }

func BenchmarkGormesColdStart(b *testing.B) {
	providers := []string{
		"openai", "anthropic", "bedrock", "deepseek", "groq",
		"ollama", "firecrawl", "openrouter", "codex", "custom",
	}

	for i := 0; i < b.N; i++ {
		pool := provider.NewClientPool()
		for _, name := range providers {
			pool.Register(name, slowFactory(name))
		}

		start := time.Now()
		_, err := pool.Get("anthropic")
		elapsed := time.Since(start)

		if err != nil {
			b.Fatalf("Get(anthropic): %v", err)
		}

		if elapsed > coldStartBudget {
			b.Errorf("cold start %v exceeds budget %v; check for eager construction of unselected providers", elapsed, coldStartBudget)
		}

		// Reset for the next iteration so each iteration is a true cold start.
		pool.Reset()
	}
}
