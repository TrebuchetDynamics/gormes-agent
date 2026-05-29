package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

func TestXSearchDescriptor(t *testing.T) {
	tools := NewXSearchTools(XSearchConfig{})
	if len(tools) == 0 {
		t.Fatal("expected x_search tools")
	}

	var found bool
	for _, tool := range tools {
		if tool.Name() == "x_search" {
			found = true
			desc := tool.Description()
			if desc == "" {
				t.Error("x_search description should not be empty")
			}
			schema := tool.Schema()
			if len(schema) == 0 {
				t.Error("x_search schema should not be empty")
			}
			if tool.Timeout() == 0 {
				t.Error("x_search timeout should be non-zero")
			}
			break
		}
	}
	if !found {
		t.Error("x_search tool not found in registry")
	}
}

func TestXSearchAuthStatus_Missing(t *testing.T) {
	cfg := XSearchConfig{}
	status := cfg.AuthStatus()

	if status.Configured {
		t.Error("expected Configured=false when no auth is set")
	}
	if status.AuthMode != "" {
		t.Errorf("expected empty AuthMode, got %q", status.AuthMode)
	}
}

func TestXSearchAuthStatus_APIKey(t *testing.T) {
	cfg := XSearchConfig{
		AuthMode: "api_key",
		APIKey:   "test-key-123",
	}
	status := cfg.AuthStatus()

	if !status.Configured {
		t.Error("expected Configured=true with API key")
	}
	if status.AuthMode != "api_key" {
		t.Errorf("expected AuthMode=api_key, got %q", status.AuthMode)
	}
	if strings.Contains(status.RedactedKey, "test-key-123") {
		t.Error("redacted key should not contain full API key")
	}
}

func TestXSearchAuthStatus_OAuth(t *testing.T) {
	cfg := XSearchConfig{
		AuthMode:    "oauth",
		OAuthToken:  "oauth-token-abc",
		OAuthExpiry: time.Now().Add(24 * time.Hour),
	}
	status := cfg.AuthStatus()

	if !status.Configured {
		t.Error("expected Configured=true with OAuth token")
	}
	if status.AuthMode != "oauth" {
		t.Errorf("expected AuthMode=oauth, got %q", status.AuthMode)
	}
}

func TestXSearchAuthStatus_OAuthExpired(t *testing.T) {
	cfg := XSearchConfig{
		AuthMode:    "oauth",
		OAuthToken:  "expired-token",
		OAuthExpiry: time.Now().Add(-1 * time.Hour),
	}
	status := cfg.AuthStatus()

	if status.Configured {
		t.Error("expected Configured=false with expired OAuth token")
	}
	if !status.Expired {
		t.Error("expected Expired=true")
	}
}

func TestXSearchExecute_MissingAuth(t *testing.T) {
	tool := &XSearchTool{cfg: XSearchConfig{}}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	if err == nil {
		t.Fatal("expected error when auth is missing")
	}
	if !strings.Contains(err.Error(), "auth") && !strings.Contains(err.Error(), "credential") {
		t.Errorf("expected auth-related error, got: %v", err)
	}
}

func TestXSearchExecute_FakeResults(t *testing.T) {
	tool := &XSearchTool{
		cfg: XSearchConfig{
			AuthMode: "api_key",
			APIKey:   "test-key",
			Fake:     true,
		},
	}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"golang testing"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp XSearchResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Error("expected fake results")
	}
}

func TestXSearchExecute_RateLimitDegraded(t *testing.T) {
	tool := &XSearchTool{
		cfg: XSearchConfig{
			AuthMode:  "api_key",
			APIKey:    "test-key",
			RateLimit: true,
		},
	}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"test"}`))
	if err == nil {
		t.Fatal("expected rate-limit error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "rate") {
		t.Errorf("expected rate-limit error, got: %v", err)
	}
}

func TestXSearchToolRegistry(t *testing.T) {
	r := toolkit.NewRegistry()
	RegisterXSearchTools(r, XSearchConfig{
		AuthMode: "api_key",
		APIKey:   "test-key",
		Fake:     true,
	})

	descs := r.Descriptors()
	var found bool
	for _, d := range descs {
		if d.Name == "x_search" {
			found = true
			break
		}
	}
	if !found {
		t.Error("x_search not found in registry descriptors")
	}
}

func TestXSearchSchema_HasQueryField(t *testing.T) {
	tools := NewXSearchTools(XSearchConfig{})
	for _, tool := range tools {
		if tool.Name() == "x_search" {
			schema := tool.Schema()
			if !strings.Contains(string(schema), "query") {
				t.Error("x_search schema should contain 'query' field")
			}
			break
		}
	}
}
