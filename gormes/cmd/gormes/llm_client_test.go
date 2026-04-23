package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/config"
)

func TestNewLLMClient_AnthropicUsesModelsHealthEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("path = %s, want /v1/models", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Fatalf("x-api-key = %q, want test-key", got)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client, endpoint := newLLMClient(config.Config{
		Hermes: config.HermesCfg{
			Provider: "anthropic",
			Endpoint: srv.URL,
			APIKey:   "test-key",
		},
	})
	if endpoint != srv.URL {
		t.Fatalf("endpoint = %q, want %q", endpoint, srv.URL)
	}
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health() error = %v", err)
	}
}

func TestNewLLMClient_AnthropicRewritesLegacyDefaultEndpoint(t *testing.T) {
	_, endpoint := newLLMClient(config.Config{
		Hermes: config.HermesCfg{
			Provider: "anthropic",
			Endpoint: "http://127.0.0.1:8642",
		},
	})
	if endpoint != "https://api.anthropic.com" {
		t.Fatalf("endpoint = %q, want https://api.anthropic.com", endpoint)
	}
}
