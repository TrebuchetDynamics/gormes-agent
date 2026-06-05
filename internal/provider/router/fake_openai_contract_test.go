package router

import (
	"context"
	"reflect"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/support/testutil/fakeopenai"
)

func TestContractFakeOpenAIProviderSupportsProbeChatAndStream(t *testing.T) {
	fake := fakeopenai.New(t)
	provider := NewHTTPUpstreamProvider(HTTPUpstreamProviderOptions{
		Client: fake.Client,
		LookupEnv: func(key string) (string, bool) {
			if key == "FAKE_OPENAI_API_KEY" {
				return "test-key", true
			}
			return "", false
		},
	})
	route := Route{
		Name:      "fake-ci",
		Provider:  "custom",
		Model:     "fake-gormes-ci",
		BaseURL:   fake.URL,
		APIKeyEnv: "FAKE_OPENAI_API_KEY",
	}

	probe := provider.Probe(context.Background(), route)
	if !probe.Available || !reflect.DeepEqual(probe.Evidence, []string{"models_probe_available"}) {
		t.Fatalf("Probe() = %+v, want available fake OpenAI-compatible provider", probe)
	}

	result, err := provider.ChatCompletion(context.Background(), route, ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "prove fake provider chat"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if result.Content != "fake provider answer" || result.Usage == nil || result.Usage.TotalTokens != 10 {
		t.Fatalf("ChatCompletion() = %+v, want fake answer with usage", result)
	}

	stream, err := provider.StreamChatCompletion(context.Background(), route, ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "prove fake provider stream"}},
	})
	if err != nil {
		t.Fatalf("StreamChatCompletion() error = %v", err)
	}
	if got := stream.Chunks; !reflect.DeepEqual(got, []string{"fake ", "provider answer"}) {
		t.Fatalf("stream chunks = %#v, want fake provider chunks", got)
	}

	requests := fake.Requests()
	if len(requests) != 2 {
		t.Fatalf("fake provider recorded %d chat requests, want 2", len(requests))
	}
	if requests[0].Model != "fake-gormes-ci" || requests[1].Model != "fake-gormes-ci" || !requests[1].Stream {
		t.Fatalf("recorded requests = %+v, want routed model and stream flag", requests)
	}
}
