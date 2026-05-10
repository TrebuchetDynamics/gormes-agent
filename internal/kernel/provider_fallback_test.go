package kernel

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
)

func TestFallbackActivationAfterEmptyResponses(t *testing.T) {
	primary := hermes.NewMockClient()
	for i := 0; i < 4; i++ {
		primary.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "")
	}

	fallback := hermes.NewMockClient()
	fallback.SetProviderStatus(hermes.ProviderStatus{Provider: "anthropic", Runtime: "chat_completions"})
	fallback.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Fallback answer."},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensOut: 1},
	}, "sess-fallback")

	k := New(Config{
		Model:     "primary-model",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Fallback: hermes.NormalizeFallbackModelConfig(map[string]any{
			"provider": "anthropic",
			"model":    "claude-opus-4-20250514",
		}),
		FallbackClientFactory: func(_ context.Context, route hermes.ModelRoute) (hermes.Client, error) {
			if route.Provider != "anthropic" || route.Model != "claude-opus-4-20250514" {
				t.Fatalf("fallback route = %#v, want anthropic claude-opus-4-20250514", route)
			}
			return fallback, nil
		},
	}, primary, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	_, final := drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)
	if len(primary.Requests()) != 4 {
		t.Fatalf("primary OpenStream calls = %d, want 4 empty attempts", len(primary.Requests()))
	}
	if reqs := fallback.Requests(); len(reqs) != 1 || reqs[0].Model != "claude-opus-4-20250514" {
		t.Fatalf("fallback requests = %#v, want one request on fallback model", reqs)
	}
	if len(final.History) == 0 || final.History[len(final.History)-1].Content != "Fallback answer." {
		t.Fatalf("final history = %#v, want fallback answer", final.History)
	}
	if !hasSoulEvent(final.SoulEvents, "fallback_activated: anthropic/claude-opus-4-20250514") {
		t.Fatalf("SoulEvents = %#v, want fallback_activated evidence", final.SoulEvents)
	}
}

func TestFallbackActivationResolvesCredentialAlias(t *testing.T) {
	t.Setenv("MY_GOOGLE_KEY", "google-secret-from-env")
	primary := hermes.NewMockClient()
	for i := 0; i < 4; i++ {
		primary.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "")
	}

	fallback := hermes.NewMockClient()
	fallback.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "Fallback answer."},
		{Kind: hermes.EventDone, FinishReason: "stop", TokensOut: 1},
	}, "sess-fallback")

	var captured hermes.ModelRoute
	k := New(Config{
		Model:     "primary-model",
		Endpoint:  "http://mock",
		Admission: Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Fallback: hermes.NormalizeFallbackModelConfig(map[string]any{
			"provider":    "custom",
			"model":       "gemini-flash",
			"base_url":    "https://generativelanguage.googleapis.com/v1beta/openai",
			"api_key_env": "MY_GOOGLE_KEY",
		}),
		FallbackClientFactory: func(_ context.Context, route hermes.ModelRoute) (hermes.Client, error) {
			captured = route
			return fallback, nil
		},
	}, primary, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	_, final := drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	if captured.Provider != "custom" || captured.Model != "gemini-flash" {
		t.Fatalf("captured fallback route = %#v, want custom/gemini-flash", captured)
	}
	if captured.BaseURL != "https://generativelanguage.googleapis.com/v1beta/openai" {
		t.Fatalf("captured BaseURL = %q", captured.BaseURL)
	}
	if captured.ExplicitAPIKey != "google-secret-from-env" {
		t.Fatalf("captured ExplicitAPIKey = %q, want env secret", captured.ExplicitAPIKey)
	}
	if hasSoulEvent(final.SoulEvents, "google-secret-from-env") {
		t.Fatalf("SoulEvents leaked fallback API key: %#v", final.SoulEvents)
	}
}

func TestCompressorBudgetUpdatesAfterFallback(t *testing.T) {
	primary := hermes.NewMockClient()
	for i := 0; i < 4; i++ {
		primary.Script([]hermes.Event{{Kind: hermes.EventDone, FinishReason: "stop"}}, "")
	}
	fallback := hermes.NewMockClient()
	fallback.Script([]hermes.Event{
		{Kind: hermes.EventToken, Token: "ok"},
		{Kind: hermes.EventDone, FinishReason: "stop"},
	}, "sess-fallback")
	engine := &fakeContextEngine{}

	k := New(Config{
		Model:         "primary-model",
		Endpoint:      "http://mock",
		Admission:     Admission{MaxBytes: 200_000, MaxLines: 10_000},
		ContextEngine: engine,
		Fallback: hermes.FallbackModelPolicy{
			Enabled: true,
			Routes:  []hermes.ModelRoute{{Provider: "openai-codex", Model: "gpt-5.5"}},
		},
		FallbackClientFactory: func(context.Context, hermes.ModelRoute) (hermes.Client, error) {
			return fallback, nil
		},
	}, primary, store.NewNoop(), telemetry.New(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go k.Run(ctx)
	initial := <-k.Render()

	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	_, _ = drainUntilIdle(t, k.Render(), initial.Seq, 2*time.Second)

	if len(engine.modelUpdates) < 2 {
		t.Fatalf("model updates = %#v, want initial primary and fallback updates", engine.modelUpdates)
	}
	got := engine.modelUpdates[len(engine.modelUpdates)-1]
	if got.Provider != "openai-codex" || got.Model != "gpt-5.5" {
		t.Fatalf("last context model update = %#v, want fallback provider/model", got)
	}
	if got.ContextLength != 272000 {
		t.Fatalf("fallback ContextLength = %d, want provider cap 272000", got.ContextLength)
	}
}

func hasSoulEvent(events []SoulEntry, needle string) bool {
	for _, event := range events {
		if strings.Contains(event.Text, needle) {
			return true
		}
	}
	return false
}
