package llm

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

type fakeStream struct{}

func (f *fakeStream) Recv(ctx context.Context) (Event, error) { return Event{}, io.EOF }
func (f *fakeStream) SessionID() string                       { return "test-session" }
func (f *fakeStream) Close() error                            { return nil }

type fakeClient struct {
	stream Stream
	err    error
}

func (f *fakeClient) OpenStream(ctx context.Context, req ChatRequest) (Stream, error) {
	return f.stream, f.err
}

func (f *fakeClient) OpenRunEvents(ctx context.Context, runID string) (RunEventStream, error) {
	return nil, nil
}

func (f *fakeClient) Health(ctx context.Context) error {
	return nil
}

type fakeFactory struct {
	clients map[string]*fakeClient
}

func (f *fakeFactory) NewClient(providerName string) (Client, error) {
	if c, ok := f.clients[providerName]; ok {
		return c, nil
	}
	return nil, errors.New("unknown provider: " + providerName)
}

func TestProviderChain_FirstProviderSucceeds(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek": {stream: &fakeStream{}},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	stream, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if stream == nil {
		t.Fatal("expected non-nil stream")
	}
	if !result.Success {
		t.Fatal("expected result.Success = true")
	}
	if result.UsedProvider != "deepseek" {
		t.Fatalf("expected used_provider=deepseek, got %s", result.UsedProvider)
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(result.Attempts))
	}
	if result.Attempts[0].Decision != ChainDecisionSuccess {
		t.Fatalf("expected decision=success, got %s", result.Attempts[0].Decision)
	}
}

func TestProviderChain_FallbackToSecondProvider(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek": {err: &HTTPError{Status: 429, Body: "rate limited"}},
			"openai":   {stream: &fakeStream{}},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	stream, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if stream == nil {
		t.Fatal("expected non-nil stream")
	}
	_ = stream
	if result.UsedProvider != "openai" {
		t.Fatalf("expected used_provider=openai, got %s", result.UsedProvider)
	}
	if len(result.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(result.Attempts))
	}
	if result.Attempts[0].Decision != ChainDecisionFallback {
		t.Fatalf("expected first decision=fallback, got %s", result.Attempts[0].Decision)
	}
	if result.Attempts[1].Decision != ChainDecisionSuccess {
		t.Fatalf("expected second decision=success, got %s", result.Attempts[1].Decision)
	}
	if len(result.FailedProviders) != 1 || result.FailedProviders[0] != "deepseek" {
		t.Fatalf("expected failed_providers=[deepseek], got %v", result.FailedProviders)
	}
}

func TestProviderChain_AbortOnAuthError(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek": {err: &HTTPError{Status: 401, Body: "invalid key"}},
			"openai":   {stream: &fakeStream{}},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	_, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error, got success")
	}
	if result.Success {
		t.Fatal("expected result.Success = false")
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(result.Attempts))
	}
	if result.Attempts[0].Decision != ChainDecisionAbort {
		t.Fatalf("expected decision=abort, got %s", result.Attempts[0].Decision)
	}
	if len(result.FailedProviders) != 1 || result.FailedProviders[0] != "deepseek" {
		t.Fatalf("expected failed_providers=[deepseek], got %v", result.FailedProviders)
	}
}

func TestProviderChain_RetryThenFallback(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek": {err: &HTTPError{Status: 503, Body: "service unavailable"}},
			"openai":   {stream: &fakeStream{}},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second
	chain.Classifier = &DefaultChainErrorClassifier{MaxRetriesPerProvider: 2}

	stream, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatalf("expected success after retry+fallback, got error: %v", err)
	}
	_ = stream
	if result.UsedProvider != "openai" {
		t.Fatalf("expected used_provider=openai, got %s", result.UsedProvider)
	}
	if len(result.Attempts) != 3 {
		t.Fatalf("expected 3 attempts (2 deepseek + 1 openai), got %d", len(result.Attempts))
	}
	if result.Attempts[0].Decision != ChainDecisionRetry {
		t.Fatalf("expected first decision=retry, got %s", result.Attempts[0].Decision)
	}
	if result.Attempts[1].Decision != ChainDecisionFallback {
		t.Fatalf("expected second decision=fallback, got %s", result.Attempts[1].Decision)
	}
	if result.Attempts[2].Decision != ChainDecisionSuccess {
		t.Fatalf("expected third decision=success, got %s", result.Attempts[2].Decision)
	}
}

func TestProviderChain_AllProvidersFail(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek":  {err: &HTTPError{Status: 500, Body: "error"}},
			"openai":    {err: &HTTPError{Status: 502, Body: "error"}},
			"anthropic": {err: &HTTPError{Status: 503, Body: "error"}},
			"grok":      {err: &HTTPError{Status: 504, Body: "error"}},
			"ollama":    {err: &HTTPError{Status: 500, Body: "error"}},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	_, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error when all providers fail")
	}
	if result.Success {
		t.Fatal("expected result.Success = false")
	}
	if len(result.Attempts) != 5 {
		t.Fatalf("expected 5 attempts, got %d", len(result.Attempts))
	}
	if len(result.FailedProviders) != 5 {
		t.Fatalf("expected 5 failed providers, got %d", len(result.FailedProviders))
	}
	if result.FinalError == nil {
		t.Fatal("expected FinalError to be set")
	}
}

func TestProviderChain_FactoryError(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	_, result, err := chain.Dispatch(context.Background(), ChatRequest{})
	if err == nil {
		t.Fatal("expected error when factory fails")
	}
	if result.Success {
		t.Fatal("expected result.Success = false")
	}
	if len(result.Attempts) != 5 {
		t.Fatalf("expected 5 attempts (all factory errors), got %d", len(result.Attempts))
	}
	for _, ev := range result.Attempts {
		if ev.Decision != ChainDecisionFallback {
			t.Fatalf("expected fallback for factory error, got %s", ev.Decision)
		}
	}
}

func TestProviderChain_ContextCancel(t *testing.T) {
	factory := &fakeFactory{
		clients: map[string]*fakeClient{
			"deepseek": {err: context.Canceled},
		},
	}
	chain := NewProviderChain(factory)
	chain.PerAttemptTimeout = 5 * time.Second

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, result, err := chain.Dispatch(ctx, ChatRequest{})
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
	if result.Success {
		t.Fatal("expected result.Success = false")
	}
	if len(result.Attempts) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(result.Attempts))
	}
}
