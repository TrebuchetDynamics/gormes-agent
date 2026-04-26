package hermes

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAzureAnthropicOpenStream_StripsTrailingV1AndSendsAPIVersionQueryWithStaticKey(t *testing.T) {
	var requestURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.URL.RequestURI()
		if r.URL.Path != "/anthropic/v1/messages" {
			t.Fatalf("path = %q, want /anthropic/v1/messages", r.URL.Path)
		}
		if got := r.URL.Query().Get("api-version"); got != "2025-04-15" {
			t.Fatalf("api-version query = %q, want 2025-04-15", got)
		}
		if strings.Contains(requestURI, "?api-version=2025-04-15/v1/messages") {
			t.Fatalf("request URI malformed with query before messages path: %q", requestURI)
		}
		if got := r.Header.Get("x-api-key"); got != "configured-azure-key" {
			t.Fatalf("x-api-key = %q, want configured Azure key", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want no bearer auth for Azure Anthropic", got)
		}
		if got := r.Header.Get("anthropic-version"); got != anthropicVersion {
			t.Fatalf("anthropic-version = %q, want %q", got, anthropicVersion)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	client := NewAzureAnthropicClient(AzureAnthropicClientConfig{
		BaseURL:    srv.URL + "/anthropic/v1",
		APIKey:     "configured-azure-key",
		APIVersion: "2025-04-15",
	})
	stored := client.(*anthropicClient)
	if stored.baseURL != srv.URL+"/anthropic" {
		t.Fatalf("stored baseURL = %q, want %q", stored.baseURL, srv.URL+"/anthropic")
	}

	stream, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream() error = %v", err)
	}
	defer stream.Close()
	if _, err := stream.Recv(context.Background()); err != io.EOF {
		t.Fatalf("Recv() err = %v, want EOF", err)
	}
	if requestURI != "/anthropic/v1/messages?api-version=2025-04-15" {
		t.Fatalf("request URI = %q, want clean api-version query after /v1/messages", requestURI)
	}
}

func TestAzureAnthropicOpenStream_UsesAzureKeyFromEnvironmentBeforeAnyOAuthPath(t *testing.T) {
	t.Setenv("AZURE_ANTHROPIC_KEY", "env-azure-key")
	t.Setenv("ANTHROPIC_TOKEN", "cc-oauth-token-that-must-not-be-used")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-api-key"); got != "env-azure-key" {
			t.Fatalf("x-api-key = %q, want AZURE_ANTHROPIC_KEY", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want Azure OAuth bypass", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	client := NewAzureAnthropicClient(AzureAnthropicClientConfig{
		BaseURL:    srv.URL + "/anthropic",
		APIVersion: "2025-04-15",
	})
	status := ProviderStatusOf(client)
	if !strings.Contains(status.Capabilities.PromptCache.Reason, "azure_oauth_bypassed") {
		t.Fatalf("PromptCache.Reason = %q, want azure_oauth_bypassed evidence", status.Capabilities.PromptCache.Reason)
	}

	stream, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream() error = %v", err)
	}
	defer stream.Close()
}

func TestAzureAnthropicOpenStream_ReportsMissingKeyBeforeLiveMessagesRequest(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer srv.Close()

	client := NewAzureAnthropicClient(AzureAnthropicClientConfig{
		BaseURL:    srv.URL + "/anthropic/v1",
		APIVersion: "2025-04-15",
		LookupEnv:  func(string) (string, bool) { return "", false },
	})
	status := ProviderStatusOf(client)
	if !strings.Contains(status.Capabilities.PromptCache.Reason, "azure_anthropic_key_missing") {
		t.Fatalf("PromptCache.Reason = %q, want azure_anthropic_key_missing evidence", status.Capabilities.PromptCache.Reason)
	}
	if !strings.Contains(status.Capabilities.PromptCache.Reason, "azure_api_version_query") {
		t.Fatalf("PromptCache.Reason = %q, want azure_api_version_query evidence", status.Capabilities.PromptCache.Reason)
	}

	_, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("OpenStream() err = nil, want missing Azure key error")
	}
	if !strings.Contains(err.Error(), "azure_anthropic_key_missing") {
		t.Fatalf("OpenStream() err = %v, want azure_anthropic_key_missing", err)
	}
	if requests != 0 {
		t.Fatalf("server requests = %d, want 0 before Azure key is configured", requests)
	}
}

func TestAnthropicOpenStream_KeepsBearerAuthForNonAzureToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer cc-oauth-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		if got := r.Header.Get("x-api-key"); got != "" {
			t.Fatalf("x-api-key = %q, want direct Anthropic bearer path to remain unchanged", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	}))
	defer srv.Close()

	client := NewAnthropicClient(srv.URL, "cc-oauth-token")
	stream, err := client.OpenStream(context.Background(), ChatRequest{
		Model:    "claude-sonnet-4-6",
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("OpenStream() error = %v", err)
	}
	defer stream.Close()
}
