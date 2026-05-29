package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientHealthSendsAuthorizationHeader(t *testing.T) {
	var sawAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != defaultHealthPath {
			t.Fatalf("path = %q, want %s", r.URL.Path, defaultHealthPath)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer health-access-token" {
			t.Fatalf("Authorization = %q, want bearer token", got)
		}
		sawAuth = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClientWithProvider(server.URL, "health-access-token", "openai-codex")
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !sawAuth {
		t.Fatal("health endpoint was not called")
	}
}

func TestHTTPClientHealthOmitsAuthorizationWhenTokenEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization = %q, want empty header", got)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "")
	if err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health: %v", err)
	}
}
