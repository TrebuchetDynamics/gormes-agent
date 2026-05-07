package apiserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAPIServerCapabilitiesEndpoint_AdvertisesHermesCompatibleContract(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})

	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var got struct {
		Object   string `json:"object"`
		Platform string `json:"platform"`
		Model    string `json:"model"`
		Auth     struct {
			Type     string `json:"type"`
			Required bool   `json:"required"`
		} `json:"auth"`
		Features struct {
			ChatCompletions         bool   `json:"chat_completions"`
			ChatCompletionsStream   bool   `json:"chat_completions_streaming"`
			ResponsesAPI            bool   `json:"responses_api"`
			ResponsesStreaming      bool   `json:"responses_streaming"`
			RunSubmission           bool   `json:"run_submission"`
			RunStatus               bool   `json:"run_status"`
			RunEventsSSE            bool   `json:"run_events_sse"`
			RunStop                 bool   `json:"run_stop"`
			ToolProgressEvents      bool   `json:"tool_progress_events"`
			SessionContinuityHeader string `json:"session_continuity_header"`
			CORS                    bool   `json:"cors"`
		} `json:"features"`
		Endpoints map[string]struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode capabilities: %v; body=%s", err, rec.Body.String())
	}
	if got.Object != "hermes.api_server.capabilities" {
		t.Fatalf("object = %q, want hermes.api_server.capabilities", got.Object)
	}
	if got.Platform != "gormes-agent" || got.Model != "gormes-agent" {
		t.Fatalf("platform/model = %q/%q, want gormes-agent/gormes-agent", got.Platform, got.Model)
	}
	if got.Auth.Type != "bearer" || got.Auth.Required {
		t.Fatalf("auth = %+v, want bearer optional", got.Auth)
	}
	if !got.Features.ChatCompletions || !got.Features.ChatCompletionsStream ||
		!got.Features.ResponsesAPI || !got.Features.ResponsesStreaming ||
		!got.Features.RunSubmission || !got.Features.RunStatus ||
		!got.Features.RunEventsSSE || !got.Features.RunStop ||
		!got.Features.ToolProgressEvents ||
		got.Features.SessionContinuityHeader != "X-Hermes-Session-Id" {
		t.Fatalf("features = %+v, missing required Hermes API-server capabilities", got.Features)
	}
	for name, wantPath := range map[string]string{
		"health":           "/health",
		"health_detailed":  "/health/detailed",
		"models":           "/v1/models",
		"chat_completions": "/v1/chat/completions",
		"responses":        "/v1/responses",
		"runs":             "/v1/runs",
		"run_status":       "/v1/runs/{run_id}",
		"run_events":       "/v1/runs/{run_id}/events",
		"run_stop":         "/v1/runs/{run_id}/stop",
	} {
		endpoint, ok := got.Endpoints[name]
		if !ok {
			t.Fatalf("missing endpoint %q in %+v", name, got.Endpoints)
		}
		if endpoint.Path != wantPath {
			t.Fatalf("endpoint %s path = %q, want %q", name, endpoint.Path, wantPath)
		}
	}
}

func TestAPIServerCapabilitiesEndpoint_RequiresBearerAuthWhenConfigured(t *testing.T) {
	srv := NewServer(Config{APIKey: "sk-capability-secret", ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})

	unauthorized := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	authorized := getJSON(t, srv.Handler(), "/v1/capabilities", map[string]string{
		"Authorization": "Bearer sk-capability-secret",
	})
	if authorized.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", authorized.Code, authorized.Body.String())
	}
	var got struct {
		Auth struct {
			Required bool `json:"required"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(authorized.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode capabilities: %v; body=%s", err, authorized.Body.String())
	}
	if !got.Auth.Required {
		t.Fatalf("auth.required = false, want true")
	}
}
