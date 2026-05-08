package apiserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
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

// TestAPIServerRunStatus_IncludesCreatedAtAndEventCount proves
// `GET /v1/runs/{run_id}` includes `created_at` (unix seconds) and
// `events_count` (count of lifecycle events published so far) so
// operators tracking long-running or stalled runs can compute age
// and detect silent runs without subscribing to the SSE stream. Same
// JSON envelope as the existing status endpoint — additive fields.
func TestAPIServerRunStatus_IncludesCreatedAtAndEventCount(t *testing.T) {
	loop := newBlockingRunLoop()
	now := time.Unix(1_700_000_000, 0)
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		BuildInfo:     BuildInfo{Version: "test-counts-attr"},
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	srv.now = func() time.Time { return now }

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	loop.waitStarted(t)

	rec := getJSON(t, srv.Handler(), "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		CreatedAt   int64  `json:"created_at"`
		EventsCount int    `json:"events_count"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.CreatedAt != now.Unix() {
		t.Errorf("created_at = %d, want %d", got.CreatedAt, now.Unix())
	}
	if got.EventsCount < 1 {
		t.Errorf("events_count = %d, want >= 1 (run.started published)", got.EventsCount)
	}
	if got.Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Status)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerHealth_ReportsRunLifecycleCounters proves the
// `/v1/health` runs telemetry exposes process-lifetime counters
// (`completed_total`, `failed_total`, `stopped_total`) so fleet
// monitoring can graph run error rates and stop velocity without
// scraping every SSE stream. Counters are additive — existing fields
// (`active`, `orphaned_swept`, `ttl_seconds`) stay intact.
func TestAPIServerHealth_ReportsRunLifecycleCounters(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	// Submit one run that we will stop, raising stopped_total.
	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode: %v", err)
	}
	loop.waitStarted(t)

	stop := postJSON(t, h, "/v1/runs/"+started.RunID+"/stop", nil, nil)
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d; body=%s", stop.Code, stop.Body.String())
	}
	loop.release(TurnResult{SessionID: started.RunID})

	rec := getJSON(t, h, "/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Runs struct {
			Active         int `json:"active"`
			OrphanedSwept  int `json:"orphaned_swept"`
			CompletedTotal int `json:"completed_total"`
			FailedTotal    int `json:"failed_total"`
			StoppedTotal   int `json:"stopped_total"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Runs.StoppedTotal != 1 {
		t.Errorf("stopped_total = %d, want 1", got.Runs.StoppedTotal)
	}
}

// TestAPIServerRunStop_PublishesRunStoppedEventToBacklog proves
// `POST /v1/runs/{run_id}/stop` records a typed `run.stopped`
// lifecycle event in the SSE backlog so subscribers reading the
// events stream see a terminal event symmetrical with `run.completed`
// and `run.failed`. Fleet automation surfacing run lifecycles needs
// this typed terminus rather than relying on stream-close inference.
func TestAPIServerRunStop_PublishesRunStoppedEventToBacklog(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	loop.waitStarted(t)

	stop := postJSON(t, srv.Handler(), "/v1/runs/"+started.RunID+"/stop", nil, nil)
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d; body=%s", stop.Code, stop.Body.String())
	}

	events := getJSON(t, srv.Handler(), "/v1/runs/"+started.RunID+"/events", nil)
	if events.Code != http.StatusOK {
		t.Fatalf("events status = %d; body=%s", events.Code, events.Body.String())
	}
	body := events.Body.String()
	if !strings.Contains(body, `"event":"run.stopped"`) {
		t.Errorf("events backlog missing run.stopped:\n%s", body)
	}

	// Release the loop so the goroutine exits.
	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStop_CancelsRunAndReportsStopped proves
// `POST /v1/runs/{run_id}/stop` cancels an in-progress run, the run
// status transitions to `stopped`, and the response envelope carries
// build attribution. Capabilities advertises this endpoint; this slice
// ships the handler. Fleet automation killing runaway runs across
// machines parses this response to confirm the stop landed.
func TestAPIServerRunStop_CancelsRunAndReportsStopped(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		BuildInfo:     BuildInfo{Version: "test-stop-attr", GitCommit: "5106dead"},
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	loop.waitStarted(t)

	stop := postJSON(t, srv.Handler(), "/v1/runs/"+started.RunID+"/stop", nil, nil)
	if stop.Code != http.StatusOK {
		t.Fatalf("stop status = %d; body=%s", stop.Code, stop.Body.String())
	}
	var stopBody struct {
		Build struct {
			Version   string `json:"version"`
			GitCommit string `json:"git_commit"`
		} `json:"build"`
		RunID  string `json:"run_id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(stop.Body.Bytes(), &stopBody); err != nil {
		t.Fatalf("decode stop: %v\nbody=%s", err, stop.Body.String())
	}
	if stopBody.Build.Version != "test-stop-attr" || stopBody.Build.GitCommit != "5106dead" {
		t.Errorf("build = %+v, want version=test-stop-attr commit=5106dead", stopBody.Build)
	}
	if stopBody.RunID != started.RunID {
		t.Errorf("run_id = %q, want %q", stopBody.RunID, started.RunID)
	}
	if stopBody.Status != "stopped" {
		t.Errorf("status = %q, want stopped", stopBody.Status)
	}

	// Release the blocking loop so the goroutine can exit cleanly.
	loop.release(TurnResult{SessionID: started.RunID})

	// A second stop on a terminal run is idempotent: still 200 with stopped.
	again := postJSON(t, srv.Handler(), "/v1/runs/"+started.RunID+"/stop", nil, nil)
	if again.Code != http.StatusOK {
		t.Fatalf("idempotent stop status = %d; body=%s", again.Code, again.Body.String())
	}

	missing := postJSON(t, srv.Handler(), "/v1/runs/run_does_not_exist/stop", nil, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unknown run stop status = %d, want 404; body=%s", missing.Code, missing.Body.String())
	}
}

// TestAPIServerRunStatus_ReportsFailedStatusOnLoopError proves
// `GET /v1/runs/{run_id}` surfaces a `failed` status when the async
// turn returned an error, distinct from `completed`. Fleet automation
// polling submitted runs needs this distinction to alert on failures
// without scraping the SSE error event. Same JSON envelope as the
// success path — build provenance leads.
func TestAPIServerRunStatus_ReportsFailedStatusOnLoopError(t *testing.T) {
	loop := &fakeTurnLoop{err: errors.New("upstream provider blew up")}
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		BuildInfo:     BuildInfo{Version: "test-runfail-attr"},
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start: %v", err)
	}

	// Poll briefly for the async turn to publish run.failed + finish.
	deadline := time.Now().Add(2 * time.Second)
	var got struct {
		Status string `json:"status"`
		Build  struct {
			Version string `json:"version"`
		} `json:"build"`
	}
	for time.Now().Before(deadline) {
		rec := getJSON(t, srv.Handler(), "/v1/runs/"+started.RunID, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status code = %d; body=%s", rec.Code, rec.Body.String())
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
		}
		if got.Status == "failed" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got.Status != "failed" {
		t.Errorf("status = %q, want failed (loop returned error)", got.Status)
	}
	if got.Build.Version != "test-runfail-attr" {
		t.Errorf("build.version = %q, want test-runfail-attr", got.Build.Version)
	}
}

// TestAPIServerRunStatus_ReportsLifecycleAndAttribution proves
// `GET /v1/runs/{run_id}` returns the run's current lifecycle status
// (`in_progress` while streaming, `completed` after the loop finishes)
// plus build provenance, so fleet automation polling submitted runs
// across machines can detect terminal state without holding an SSE
// connection. The capabilities endpoint already advertises this route
// — this slice ships the handler. Build provenance leads — same
// convention as the rest of the JSON arc.
func TestAPIServerRunStatus_ReportsLifecycleAndAttribution(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-status"}}
	srv := NewServer(Config{
		ModelName: "gormes-agent",
		BuildInfo: BuildInfo{
			Version:   "test-runstatus-attr",
			GitCommit: "deadc0de",
		},
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d; body=%s", start.Code, start.Body.String())
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	if started.RunID == "" {
		t.Fatal("run_id empty in start envelope")
	}

	rec := getJSON(t, srv.Handler(), "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status status = %d; body=%s", rec.Code, rec.Body.String())
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
	if got.Build.Version != "test-runstatus-attr" || got.Build.GitCommit != "deadc0de" {
		t.Errorf("build = %+v, want version=test-runstatus-attr commit=deadc0de", got.Build)
	}
	if got.RunID != started.RunID {
		t.Errorf("run_id = %q, want %q", got.RunID, started.RunID)
	}
	if got.Status != "in_progress" && got.Status != "completed" {
		t.Errorf("status = %q, want in_progress or completed", got.Status)
	}

	missing := getJSON(t, srv.Handler(), "/v1/runs/run_does_not_exist", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404; body=%s", missing.Code, missing.Body.String())
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
