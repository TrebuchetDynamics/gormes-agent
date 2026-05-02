package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func TestProviderHTTPClient_UsesCodexOAuthCredentialPoolWhenEndpointEmpty(t *testing.T) {
	gormesHome := t.TempDir()
	t.Setenv("GORMES_HOME", gormesHome)

	var sawResponsesPath bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("request path = %q, want /v1/responses", r.URL.Path)
		}
		sawResponsesPath = true
		if r.Header.Get("Authorization") != "Bearer pool-access-token" {
			t.Fatalf("Authorization = %q, want credential pool access token", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"completed","output_text":"ok from codex pool"}`))
	}))
	defer server.Close()

	store := config.NewCodexOAuthStateStore(config.CodexOAuthStateStoreOptions{HermesHome: gormesHome})
	if _, err := store.SaveTokens(config.CodexOAuthTokens{
		AccountID:    "acct-pool",
		Label:        "Pool Account",
		AccessToken:  "pool-access-token",
		RefreshToken: "pool-refresh-token",
		BaseURL:      server.URL,
	}); err != nil {
		t.Fatalf("SaveTokens: %v", err)
	}

	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Model:    "gpt-5.5",
		Provider: "openai-codex",
	}}, "openai-codex")
	if err != nil {
		t.Fatalf("newProviderHTTPClient: %v", err)
	}

	stream, err := client.OpenStream(context.Background(), hermes.ChatRequest{
		Model:    "gpt-5.5",
		Messages: []hermes.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream error = %v", err)
	}
	defer stream.Close()
	event, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatalf("Recv token error = %v", err)
	}
	if event.Kind != hermes.EventToken || event.Token != "ok from codex pool" {
		t.Fatalf("event = %+v, want codex pool token event", event)
	}
	if !sawResponsesPath {
		t.Fatal("credential-pool backed responses endpoint was not called")
	}
}

func TestProviderHTTPClient_CodexMissingCredentialFailsBeforeRelativeURL(t *testing.T) {
	t.Setenv("GORMES_HOME", t.TempDir())
	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Model:    "gpt-5.5",
		Provider: "openai-codex",
	}}, "openai-codex")
	if err == nil {
		t.Fatalf("error = nil, client=%T; want credential setup failure", client)
	}
	if client != nil {
		t.Fatalf("client = %T, want nil on missing credential", client)
	}
	if got := err.Error(); got == "" || got == `Post "/v1/responses": unsupported protocol scheme ""` {
		t.Fatalf("error = %q, want setup evidence before relative /v1/responses URL", got)
	}
}

// TestProviderHTTPClient_EmptyEndpointGenericProviderFailsBeforeRelativeURL
// reproduces the live-incident bug where Telegram surfaced
// `Post "/v1/responses": unsupported protocol scheme ""` after the localhost
// default was removed. The bug was: a configured non-Codex provider with an
// empty Hermes.Endpoint silently built an HTTP client whose baseURL is "",
// so the first request to `/v1/responses` had no scheme/host. The fix must
// surface a setup error at construction time naming the missing endpoint.
func TestProviderHTTPClient_EmptyEndpointGenericProviderFailsBeforeRelativeURL(t *testing.T) {
	for _, provider := range []string{"openai", "anthropic"} {
		t.Run(provider, func(t *testing.T) {
			client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
				Model:    "gpt-5.5",
				APIKey:   "k-xyz",
				Provider: provider,
			}}, provider)
			if err == nil {
				t.Fatalf("error = nil, client=%T; want endpoint-missing failure", client)
			}
			if client != nil {
				t.Fatalf("client = %T, want nil when endpoint is unconfigured", client)
			}
			got := err.Error()
			if got == "" {
				t.Fatalf("error message empty; want setup evidence")
			}
			if got == `Post "/v1/responses": unsupported protocol scheme ""` {
				t.Fatalf("error = %q, want setup evidence BEFORE the relative /v1/responses URL is dialed", got)
			}
			if !strings.Contains(strings.ToLower(got), "endpoint") {
				t.Fatalf("error = %q, want message mentioning the missing endpoint so operators know what to set", got)
			}
		})
	}
}

func TestProviderHTTPClient_AnthropicProviderUsesMessagesAPI(t *testing.T) {
	var sawMessagesPath bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("request path = %q, want /v1/messages", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-ant-api-test" {
			t.Fatalf("x-api-key = %q, want sk-ant-api-test", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatalf("anthropic-version header missing")
		}
		sawMessagesPath = true
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Endpoint: server.URL + "/v1",
		Model:    "claude-sonnet-4",
		APIKey:   "sk-ant-api-test",
		Provider: "anthropic",
	}}, "anthropic")
	if err != nil {
		t.Fatalf("newProviderHTTPClient: %v", err)
	}
	stream, err := client.OpenStream(context.Background(), hermes.ChatRequest{
		Model:     "claude-sonnet-4",
		MaxTokens: 512,
		Messages:  []hermes.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream error = %v", err)
	}
	defer stream.Close()
	if !sawMessagesPath {
		t.Fatal("Anthropic Messages endpoint was not called")
	}
}

func TestProviderHTTPClient_OpenRouterDefaultRuntimeUsesOpenRouterKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "or-test-key")
	t.Setenv("OPENAI_API_KEY", "openai-fallback-key")

	endpoint, apiKey, err := resolveProviderHTTPClientCredentials(config.Config{Hermes: config.HermesCfg{
		Model:    "anthropic/claude-sonnet-4",
		Provider: "openrouter",
	}}, "openrouter")
	if err != nil {
		t.Fatalf("resolveProviderHTTPClientCredentials: %v", err)
	}
	if endpoint != hermes.OpenRouterDefaultBaseURL {
		t.Fatalf("endpoint = %q, want OpenRouter default", endpoint)
	}
	if apiKey != "or-test-key" {
		t.Fatalf("apiKey = %q, want OPENROUTER_API_KEY before OPENAI_API_KEY", apiKey)
	}
}

func TestProviderHTTPClient_OpenRouterMissingKeyFailsBeforeRelativeURL(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Model:    "anthropic/claude-sonnet-4",
		Provider: "openrouter",
	}}, "openrouter")
	if err == nil {
		t.Fatalf("error = nil, client=%T; want OpenRouter credential setup failure", client)
	}
	if client != nil {
		t.Fatalf("client = %T, want nil on missing OpenRouter key", client)
	}
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "openrouter") || !strings.Contains(got, "openrouter_api_key") {
		t.Fatalf("error = %q, want actionable OpenRouter key setup evidence", err)
	}
	if got == `post "/v1/chat/completions": unsupported protocol scheme ""` {
		t.Fatalf("error = %q, want setup evidence before relative URL", err)
	}
}

func TestProviderHTTPClient_CustomOpenRouterBaseUsesOpenRouterKey(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "or-test-key")

	endpoint, apiKey, err := resolveProviderHTTPClientCredentials(config.Config{Hermes: config.HermesCfg{
		Endpoint: "https://openrouter.ai/api/v1",
		Model:    "anthropic/claude-sonnet-4",
		Provider: "custom",
	}}, "custom")
	if err != nil {
		t.Fatalf("resolveProviderHTTPClientCredentials: %v", err)
	}
	if endpoint != hermes.OpenRouterDefaultBaseURL {
		t.Fatalf("endpoint = %q, want OpenRouter endpoint preserved", endpoint)
	}
	if apiKey != "or-test-key" {
		t.Fatalf("apiKey = %q, want OPENROUTER_API_KEY for custom OpenRouter base", apiKey)
	}
}

func TestProviderHTTPClient_OpenRouterRequestsCarryAttribution(t *testing.T) {
	var sawAttribution bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %q, want /v1/chat/completions", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer or-test-key" {
			t.Fatalf("Authorization = %q, want OpenRouter key", r.Header.Get("Authorization"))
		}
		if r.Header.Get("HTTP-Referer") == "" || r.Header.Get("X-OpenRouter-Title") == "" || r.Header.Get("X-OpenRouter-Categories") == "" {
			t.Fatalf("OpenRouter attribution headers missing: %#v", r.Header)
		}
		sawAttribution = true
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Endpoint: server.URL,
		APIKey:   "or-test-key",
		Model:    "anthropic/claude-sonnet-4",
		Provider: "openrouter",
	}}, "openrouter")
	if err != nil {
		t.Fatalf("newProviderHTTPClient: %v", err)
	}
	stream, err := client.OpenStream(context.Background(), hermes.ChatRequest{
		Model:    "anthropic/claude-sonnet-4",
		Messages: []hermes.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer stream.Close()
	if !sawAttribution {
		t.Fatal("OpenRouter request was not observed")
	}
}

// TestProviderHTTPClient_EmptyEndpointEmptyProviderFailsClearly covers the
// dead-config path: no endpoint configured and no provider declared. The
// helper must not silently dial an empty URL; it must error with both
// missing identifiers named so operators can see what to set.
func TestProviderHTTPClient_EmptyEndpointEmptyProviderFailsClearly(t *testing.T) {
	client, err := newProviderHTTPClient(config.Config{Hermes: config.HermesCfg{
		Model: "gpt-5.5",
	}}, "")
	if err == nil {
		t.Fatalf("error = nil, client=%T; want endpoint-and-provider missing failure", client)
	}
	if client != nil {
		t.Fatalf("client = %T, want nil when endpoint and provider both unset", client)
	}
	got := strings.ToLower(err.Error())
	if !strings.Contains(got, "endpoint") {
		t.Fatalf("error = %q, want mention of missing endpoint", got)
	}
}
