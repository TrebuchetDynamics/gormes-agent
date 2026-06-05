package usagehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestClientDoAccountUsageRequestSendsNonBlankHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization header = %q", got)
		}
		if got := r.Header.Get("X-Blank"); got != "" {
			t.Fatalf("blank header should be omitted, got %q", got)
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	resp, err := (Client{Client: server.Client()}).DoAccountUsageRequest(context.Background(), llm.AccountUsageHTTPRequest{
		URL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer token",
			"X-Blank":       "  ",
		},
	})
	if err != nil {
		t.Fatalf("DoAccountUsageRequest: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Fatalf("body = %s", string(resp.Body))
	}
}
