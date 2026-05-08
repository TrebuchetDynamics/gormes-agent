package apiserver

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAPIServerCapabilities_BuildAttribution proves `/v1/capabilities`
// carries the configured BuildInfo at the top of the JSON response so
// fleet automation discovering Gormes capabilities across machines
// can attribute the contract advertisement to the binary version that
// emitted it. Same convention as the dashboard JSON arc — additive to
// the OpenAI-compatible contract since clients ignore unknown fields.
func TestAPIServerCapabilities_BuildAttribution(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-cap-attr",
			GitCommit: "feedfade",
		},
	})
	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		Object string `json:"object"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Build.Version != "test-cap-attr" {
		t.Errorf("build.version = %q, want test-cap-attr (body=%s)", got.Build.Version, rec.Body.String())
	}
	if got.Build.GitCommit != "feedfade" {
		t.Errorf("build.git_commit = %q, want feedfade", got.Build.GitCommit)
	}
	if got.Object != "hermes.api_server.capabilities" {
		t.Errorf("object = %q, want hermes.api_server.capabilities (still addressable)", got.Object)
	}
}

// TestAPIServerHealth_BuildAttribution proves `/health` and
// `/v1/health` carry BuildInfo so fleet health-monitoring across
// machines can attribute each probe to the binary version.
func TestAPIServerHealth_BuildAttribution(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-health-attr",
			GitCommit: "abad1dea",
		},
	})
	for _, path := range []string{"/health", "/v1/health"} {
		rec := getJSON(t, srv.Handler(), path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d; body=%s", path, rec.Code, rec.Body.String())
		}
		var got struct {
			Build struct {
				Version string `json:"version"`
			} `json:"build"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("%s decode: %v\nbody=%s", path, err, rec.Body.String())
		}
		if got.Build.Version != "test-health-attr" {
			t.Errorf("%s build.version = %q, want test-health-attr (body=%s)", path, got.Build.Version, rec.Body.String())
		}
		if got.Status != "ok" {
			t.Errorf("%s status = %q, want ok (still addressable)", path, got.Status)
		}
	}
}

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
