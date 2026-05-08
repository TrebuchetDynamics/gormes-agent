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

// TestAPIServer_BuildHeadersOnEveryResponse proves every apiserver
// response carries `X-Gormes-Build-Version` and `X-Gormes-Build-Commit`
// headers so fleet log aggregation can attribute responses to a binary
// version without JSON parsing. Same source of truth as the JSON
// build envelopes — driven by Config.BuildInfo. Set unconditionally
// in middleware so the OpenAI-compatible /v1/models endpoint (which
// keeps its body OpenAI-spec) still gets attribution.
func TestAPIServer_BuildHeadersOnEveryResponse(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-headers-attr",
			GitCommit: "1337c0de",
		},
	})
	for _, path := range []string{"/health", "/v1/health", "/v1/capabilities"} {
		rec := getJSON(t, srv.Handler(), path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d", path, rec.Code)
		}
		if got := rec.Header().Get("X-Gormes-Build-Version"); got != "test-headers-attr" {
			t.Errorf("%s X-Gormes-Build-Version = %q, want test-headers-attr", path, got)
		}
		if got := rec.Header().Get("X-Gormes-Build-Commit"); got != "1337c0de" {
			t.Errorf("%s X-Gormes-Build-Commit = %q, want 1337c0de", path, got)
		}
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

// TestAPIServerRuns_BuildAttribution proves `POST /v1/runs` carries the
// configured BuildInfo at the top of the start envelope so fleet
// automation triggering runs across machines can attribute each run to
// the binary version that minted it. Same convention as the dashboard
// JSON arc — additive to the Hermes-compat contract because clients
// ignore unknown fields.
func TestAPIServerRuns_BuildAttribution(t *testing.T) {
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-runs-attr",
			GitCommit: "ba5eba11",
		},
		Loop:          &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-attr"}},
		ResponseStore: NewResponseStore(10),
	})
	rec := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Build.Version != "test-runs-attr" {
		t.Errorf("build.version = %q, want test-runs-attr (body=%s)", got.Build.Version, rec.Body.String())
	}
	if got.Build.GitCommit != "ba5eba11" {
		t.Errorf("build.git_commit = %q, want ba5eba11", got.Build.GitCommit)
	}
	if got.RunID == "" || got.Status != "started" {
		t.Errorf("run start envelope = %+v, want run_id populated and status=started", got)
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
