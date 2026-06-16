package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedTestClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *lockedTestClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *lockedTestClock) Set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
}

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
		t.Errorf("object = %q, want llm.api_server.capabilities (still addressable)", got.Object)
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

// TestAPIServerDetailedHealth_AutoPopulatesRunEventsFromRegistry
// proves that `/health/detailed` surfaces run lifecycle counters
// automatically from the runRegistry — without requiring callers to
// thread DetailedHealthRunEventsInput through their DetailedHealth
// callback. Single source of truth: the runRegistry. Same arc as
// `/v1/health` runs telemetry, but exposed through the structured
// /health/detailed model.
func TestAPIServerDetailedHealth_AutoPopulatesRunEventsFromRegistry(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode: %v", err)
	}
	loop.waitStarted(t)

	if rec := postJSON(t, h, "/v1/runs/"+started.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d", rec.Code)
	}
	loop.release(TurnResult{SessionID: started.RunID})

	rec := getJSON(t, h, "/health/detailed", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		RunEvents struct {
			Available    bool `json:"available"`
			StoppedTotal int  `json:"stopped_total"`
		} `json:"run_events"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if !got.RunEvents.Available {
		t.Errorf("run_events.available = false, want true (registry is wired)")
	}
	if got.RunEvents.StoppedTotal != 1 {
		t.Errorf("stopped_total = %d, want 1", got.RunEvents.StoppedTotal)
	}
}

// TestAPIServerHealth_ReportsOldestActiveAgeSeconds proves
// `/v1/health` runs telemetry exposes `oldest_active_age_seconds` —
// the age (in seconds) of the longest-running active (non-terminal)
// run. Operators alerting on stalled runs at the fleet level can
// trigger when the oldest active run exceeds an SLA without polling
// every individual run snapshot.
func TestAPIServerHealth_ReportsOldestActiveAgeSeconds(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	now := time.Unix(1_700_012_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	// Submit run at t=12000.
	if rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil); rec.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", rec.Code)
	}
	loop.waitStarted(t)

	// Advance clock by 90s.
	now = time.Unix(1_700_012_090, 0)

	rec := getJSON(t, h, "/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health = %d", rec.Code)
	}
	var got struct {
		Runs struct {
			OldestActiveAgeSeconds int64 `json:"oldest_active_age_seconds"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Runs.OldestActiveAgeSeconds != 90 {
		t.Errorf("oldest_active_age_seconds = %d, want 90", got.Runs.OldestActiveAgeSeconds)
	}
}

// TestAPIServerHealth_ReportsPeakActiveRunsCounter proves
// `/v1/health` runs telemetry exposes `peak_active` — the high-water
// mark of active concurrent runs across process lifetime — so
// operators sizing API capacity can detect whether their fleet ever
// hits the practical concurrency ceiling without scraping live
// metrics over time.
func TestAPIServerHealth_ReportsPeakActiveRunsCounter(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	h := srv.Handler()

	// Submit 3 runs (all in_progress because loop blocks).
	for i := 0; i < 3; i++ {
		if rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil); rec.Code != http.StatusAccepted {
			t.Fatalf("POST %d = %d", i, rec.Code)
		}
	}

	rec := getJSON(t, h, "/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health = %d", rec.Code)
	}
	var got struct {
		Runs struct {
			Active     int `json:"active"`
			PeakActive int `json:"peak_active"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Runs.Active != 3 {
		t.Errorf("active = %d, want 3", got.Runs.Active)
	}
	if got.Runs.PeakActive != 3 {
		t.Errorf("peak_active = %d, want 3 (high-water mark)", got.Runs.PeakActive)
	}
}

// TestAPIServerHealth_ReportsRequestTotalRunsCounter proves the
// `/v1/health` runs telemetry exposes a `request_total` counter
// incremented every time a run is submitted via POST /v1/runs.
// Operators measuring run throughput need this independent of
// terminal-state counters (completed/failed/stopped) so they can
// derive in-flight + abandoned counts without snapshot drift.
func TestAPIServerHealth_ReportsRequestTotalRunsCounter(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-throughput"}}
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	for i := 0; i < 3; i++ {
		rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("POST %d status = %d", i, rec.Code)
		}
	}

	rec := getJSON(t, h, "/v1/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health = %d", rec.Code)
	}
	var got struct {
		Runs struct {
			RequestTotal int `json:"request_total"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Runs.RequestTotal != 3 {
		t.Errorf("request_total = %d, want 3", got.Runs.RequestTotal)
	}
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

// TestAPIServerRunStop_EnvelopeMirrorsStatusEnvelope proves the
// stop response envelope mirrors the GET /v1/runs/{run_id} status
// envelope shape — same `created_at` and `events_count` so callers
// using `POST /stop` followed by `GET /v1/runs/{id}` see consistent
// metadata or can cache the stop response as the canonical run snap.
func TestAPIServerRunStop_EnvelopeMirrorsStatusEnvelope(t *testing.T) {
	loop := newBlockingRunLoop()
	now := time.Unix(1_700_001_000, 0)
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	srv.now = func() time.Time { return now }

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode: %v", err)
	}
	loop.waitStarted(t)

	rec := postJSON(t, srv.Handler(), "/v1/runs/"+started.RunID+"/stop", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		CreatedAt   int64 `json:"created_at"`
		EventsCount int   `json:"events_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.CreatedAt != now.Unix() {
		t.Errorf("created_at = %d, want %d", got.CreatedAt, now.Unix())
	}
	if got.EventsCount < 2 {
		t.Errorf("events_count = %d, want >= 2 (run.started + run.stopped)", got.EventsCount)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestRunRegistry_BoundsEventsBacklogToCap proves the per-record
// events backlog is capped so a long-running run that emits many
// tool.progress / message.delta events doesn't grow registry memory
// unboundedly. When the cap is reached, the oldest events are
// dropped — the most recent events stay visible to late subscribers.
func TestRunRegistry_BoundsEventsBacklogToCap(t *testing.T) {
	r := newRunRegistry(time.Hour, time.Now)
	r.create("run_cap", nil)
	for i := 0; i < runEventsBacklogCap+50; i++ {
		r.publish("run_cap", runEvent{Event: "message.delta", RunID: "run_cap"})
	}
	snap, ok := r.snapshot("run_cap")
	if !ok {
		t.Fatal("snapshot returned false")
	}
	if snap.EventsCount > runEventsBacklogCap {
		t.Errorf("events_count = %d, want <= %d (capped)", snap.EventsCount, runEventsBacklogCap)
	}
	if snap.LastEventType != "message.delta" {
		t.Errorf("last_event_type = %q, want message.delta", snap.LastEventType)
	}
}

// TestAPIServerRunEvents_EmitsSnapshotPreludeComment proves
// `GET /v1/runs/{run_id}/events` writes an SSE comment containing
// the run snapshot summary before any backlog event so consumers
// see context (status, events_count, etc.) immediately even when
// the backlog is empty. SSE comments start with `:` per spec and
// are ignored by EventSource clients while remaining visible to
// raw HTTP debuggers.
func TestAPIServerRunEvents_EmitsSnapshotPreludeComment(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	// Stop so SSE drain returns immediately.
	if rec := postJSON(t, h, "/v1/runs/"+started.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d", rec.Code)
	}

	rec := getJSON(t, h, "/v1/runs/"+started.RunID+"/events", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("events status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ": snapshot ") {
		t.Errorf("SSE body missing snapshot prelude:\n%s", body)
	}
	if !strings.Contains(body, started.RunID) {
		t.Errorf("snapshot prelude missing run_id %q:\n%s", started.RunID, body)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunEvents_PreludeIncludesTimestamps proves the SSE
// prelude carries `created_at` and `last_event_at` so reconnecting
// consumers can compute lag (now - last_event_at) and run age
// (now - created_at) before any backlog event arrives.
func TestAPIServerRunEvents_PreludeIncludesTimestamps(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	now := time.Unix(1_700_015_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID+"/events?backlog_only=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"created_at":1700015000`) {
		t.Errorf("SSE prelude missing created_at:\n%s", body)
	}
	if !strings.Contains(body, `"last_event_at":1700015000`) {
		t.Errorf("SSE prelude missing last_event_at:\n%s", body)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunEvents_PreludeIncludesSessionID proves the SSE
// prelude comment carries `session_id` when one is set, so consumers
// can route or filter at stream open without a round-trip to the
// status endpoint.
func TestAPIServerRunEvents_PreludeIncludesSessionID(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-prelude"}}
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "sess-prelude-123"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID+"/events", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"session_id":"sess-prelude-123"`) {
		t.Errorf("SSE prelude missing session_id:\n%s", body)
	}
}

// TestAPIServerRunEvents_BacklogOnlyClosesAfterDrain proves
// `GET /v1/runs/{run_id}/events?backlog_only=true` writes the
// existing backlog then closes the stream — even for in-progress
// runs that would otherwise keep the connection open. Operators
// reading historical events without holding a live connection use
// this for snapshot inspection.
func TestAPIServerRunEvents_BacklogOnlyClosesAfterDrain(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	// Run is still in_progress (loop blocked). Without backlog_only this
	// would block. With backlog_only, drain then close.
	rec := getJSON(t, h, "/v1/runs/"+started.RunID+"/events?backlog_only=true", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"event":"run.started"`) {
		t.Errorf("backlog drain missing run.started:\n%s", body)
	}
	if !strings.Contains(body, ": stream closed") {
		t.Errorf("backlog_only stream did not close cleanly:\n%s", body)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunEvents_KeepFlagPreservesRunForStatusQuery proves
// that `GET /v1/runs/{run_id}/events?keep=true` does NOT remove the
// run from the registry after the SSE stream closes, so callers can
// subsequently fetch terminal status, duration, and error via
// `GET /v1/runs/{run_id}`. Without `keep`, the run is removed on
// stream end (current default to bound memory).
func TestAPIServerRunEvents_KeepFlagPreservesRunForStatusQuery(t *testing.T) {
	loop := &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-keep"}}
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)

	// Drain events with keep=true.
	events := getJSON(t, h, "/v1/runs/"+started.RunID+"/events?keep=true", nil)
	if events.Code != http.StatusOK {
		t.Fatalf("events status = %d", events.Code)
	}

	// Status should still be reachable after stream close.
	status := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if status.Code != http.StatusOK {
		t.Fatalf("post-events status code = %d (want 200, run should be retained); body=%s", status.Code, status.Body.String())
	}

	// Confirm default removes the run: drain again WITHOUT keep, expect 404 on subsequent status.
	loop2 := &fakeTurnLoop{result: TurnResult{Content: "ok", SessionID: "sess-no-keep"}}
	srv2 := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop2,
		ResponseStore: NewResponseStore(10),
	})
	h2 := srv2.Handler()
	r2 := postJSON(t, h2, "/v1/runs", map[string]any{"input": "ping"}, nil)
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)
	getJSON(t, h2, "/v1/runs/"+s2.RunID+"/events", nil)
	gone := getJSON(t, h2, "/v1/runs/"+s2.RunID, nil)
	if gone.Code != http.StatusNotFound {
		t.Errorf("default events drain followed by status = %d, want 404", gone.Code)
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

// TestAPIServerRunStatus_IncludesDurationOnTerminal proves the run
// status response carries `duration_seconds` (terminated_at -
// created_at) when the run reaches terminal status. Latency
// dashboards consume this directly without arithmetic on two unix
// timestamps in every render. Non-terminal runs omit the field.
func TestAPIServerRunStatus_IncludesDurationOnTerminal(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	now := time.Unix(1_700_008_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	// Pre-stop: duration_seconds omitted.
	pre := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	var preBody struct {
		DurationSeconds int64 `json:"duration_seconds"`
	}
	json.Unmarshal(pre.Body.Bytes(), &preBody)
	if preBody.DurationSeconds != 0 {
		t.Errorf("pre-stop duration_seconds = %d, want 0 (omitempty for non-terminal)", preBody.DurationSeconds)
	}

	now = time.Unix(1_700_008_042, 0)
	if rec := postJSON(t, h, "/v1/runs/"+started.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d", rec.Code)
	}

	post := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if post.Code != http.StatusOK {
		t.Fatalf("post-stop status = %d", post.Code)
	}
	var postBody struct {
		DurationSeconds int64 `json:"duration_seconds"`
	}
	if err := json.Unmarshal(post.Body.Bytes(), &postBody); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, post.Body.String())
	}
	if postBody.DurationSeconds != 42 {
		t.Errorf("duration_seconds = %d, want 42", postBody.DurationSeconds)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_IncludesTerminatedAtWhenTerminal proves the
// run status response carries `terminated_at` (unix seconds) when
// the run reaches a terminal status (completed/failed/stopped) so
// dashboards can compute run duration as terminated_at - created_at
// without scraping events. Non-terminal runs omit the field.
func TestAPIServerRunStatus_IncludesTerminatedAtWhenTerminal(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	now := time.Unix(1_700_007_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	// Pre-stop: terminated_at omitted.
	pre := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	var preBody struct {
		TerminatedAt int64 `json:"terminated_at"`
	}
	json.Unmarshal(pre.Body.Bytes(), &preBody)
	if preBody.TerminatedAt != 0 {
		t.Errorf("pre-stop terminated_at = %d, want 0 (omitempty for non-terminal)", preBody.TerminatedAt)
	}

	// Advance clock and stop the run.
	now = time.Unix(1_700_007_045, 0)
	if rec := postJSON(t, h, "/v1/runs/"+started.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d", rec.Code)
	}

	post := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if post.Code != http.StatusOK {
		t.Fatalf("post-stop status = %d; body=%s", post.Code, post.Body.String())
	}
	var postBody struct {
		Status       string `json:"status"`
		TerminatedAt int64  `json:"terminated_at"`
	}
	if err := json.Unmarshal(post.Body.Bytes(), &postBody); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, post.Body.String())
	}
	if postBody.Status != "stopped" {
		t.Errorf("status = %q, want stopped", postBody.Status)
	}
	if postBody.TerminatedAt != now.Unix() {
		t.Errorf("terminated_at = %d, want %d (clock at stop)", postBody.TerminatedAt, now.Unix())
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_IncludesSessionID proves the run status
// response carries `session_id` so dashboards grouping runs by
// session can do so without re-deriving from the run_id alone.
// Hermes' Responses API supports explicit session_id; runs created
// without one fall back to using run_id as session_id.
func TestAPIServerRunStatus_IncludesSessionID(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "sess-abc-123"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.SessionID != "sess-abc-123" {
		t.Errorf("session_id = %q, want sess-abc-123", got.SessionID)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_IncludesIdleSecondsForActiveRuns proves
// non-terminal runs carry `idle_seconds` (server clock - last
// event timestamp). Operators detecting stalled runs see it
// directly without subtracting two unix timestamps in their
// dashboard renderer. Terminal runs omit the field — duration
// already covers their lifetime.
func TestAPIServerRunStatus_IncludesIdleSecondsForActiveRuns(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	now := time.Unix(1_700_014_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	// run.started published at t=14000. Advance 25s.
	now = time.Unix(1_700_014_025, 0)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Status      string `json:"status"`
		IdleSeconds int64  `json:"idle_seconds"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Status != "in_progress" {
		t.Fatalf("status = %q, want in_progress", got.Status)
	}
	if got.IdleSeconds != 25 {
		t.Errorf("idle_seconds = %d, want 25", got.IdleSeconds)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_IncludesLastEventAt proves the run status
// response carries `last_event_at` (unix seconds of most recent
// event timestamp) so dashboards can detect silent runs by computing
// `now - last_event_at`. Complements `last_event_type` to reveal
// "stuck on tool.progress for 5 minutes" without scraping events.
func TestAPIServerRunStatus_IncludesLastEventAt(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	now := time.Unix(1_700_009_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		LastEventAt int64 `json:"last_event_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.LastEventAt != now.Unix() {
		t.Errorf("last_event_at = %d, want %d", got.LastEventAt, now.Unix())
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_IncludesToolCallsCount proves the run
// snapshot exposes `tool_calls_count` — the number of tool.progress
// events emitted so far. Dashboards showing tool activity per run
// (e.g. "Read File 3x, Bash 1x") read this directly without
// scanning the events backlog.
func TestAPIServerRunStatus_IncludesToolCallsCount(t *testing.T) {
	loop := newBlockingRunLoopWithToolProgress(t, 3)
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitToolProgress(t)

	rec := getJSON(t, h, "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		ToolCallsCount int `json:"tool_calls_count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.ToolCallsCount != 3 {
		t.Errorf("tool_calls_count = %d, want 3", got.ToolCallsCount)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// toolProgressLoop emits the requested number of tool.progress
// events, then blocks awaiting release.
type toolProgressLoop struct {
	*blockingRunLoop
	progressCount int
	progressDone  chan struct{}
}

func newBlockingRunLoopWithToolProgress(t *testing.T, count int) *toolProgressLoop {
	return &toolProgressLoop{
		blockingRunLoop: newBlockingRunLoop(),
		progressCount:   count,
		progressDone:    make(chan struct{}),
	}
}

func (l *toolProgressLoop) StreamTurn(ctx context.Context, req TurnRequest, cb StreamCallbacks) (TurnResult, error) {
	for i := 0; i < l.progressCount; i++ {
		_ = cb.OnToolProgress(ToolProgressEvent{Name: "fake_tool", Status: "completed"})
	}
	close(l.progressDone)
	return l.blockingRunLoop.StreamTurn(ctx, req, cb)
}

func (l *toolProgressLoop) waitToolProgress(t *testing.T) {
	t.Helper()
	select {
	case <-l.progressDone:
	case <-time.After(time.Second):
		t.Fatal("tool progress events did not arrive")
	}
}

// TestAPIServerRunStatus_IncludesLastEventType proves the run status
// response carries `last_event_type` reflecting the most recent
// lifecycle event type published. Operators detecting stalled runs
// stuck in the middle of streaming can spot a run with status
// "in_progress" but `last_event_type=message.delta` from minutes ago,
// distinguishing a stuck loop from one that hasn't started yet.
func TestAPIServerRunStatus_IncludesLastEventType(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(start.Body.Bytes(), &started)
	loop.waitStarted(t)

	rec := getJSON(t, srv.Handler(), "/v1/runs/"+started.RunID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		LastEventType string `json:"last_event_type"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.LastEventType != "run.started" {
		t.Errorf("last_event_type = %q, want run.started", got.LastEventType)
	}

	loop.release(TurnResult{SessionID: started.RunID})
}

// TestAPIServerRunStatus_FailedRunIncludesErrorMessage proves the
// run status endpoint surfaces the `error` field carrying the loop's
// failure message when status is `failed`. Fleet automation polling
// failed runs needs the error string to alert/route without scraping
// the run.failed SSE event from the backlog. Successful runs omit the
// `error` field via `omitempty` to keep the success envelope tight.
func TestAPIServerRunStatus_FailedRunIncludesErrorMessage(t *testing.T) {
	loop := &fakeTurnLoop{err: errors.New("upstream provider 503")}
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})

	start := postJSON(t, srv.Handler(), "/v1/runs", map[string]any{"input": "ping"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST status = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var got struct {
		Status string `json:"status"`
		Error  string `json:"error"`
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
		t.Fatalf("status = %q, want failed", got.Status)
	}
	if got.Error != "upstream provider 503" {
		t.Errorf("error = %q, want \"upstream provider 503\"", got.Error)
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

// TestAPIServerRunsList_FiltersBySinceTimestamp proves
// `GET /v1/runs?since=<unix>` returns only runs whose created_at is
// >= the given timestamp. Fleet automation polling for new runs
// since the last poll uses this to avoid re-scanning runs it has
// already seen, supporting incremental dashboards.
func TestAPIServerRunsList_FiltersBySinceTimestamp(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour, // keep both runs alive past the orphan sweep
	})
	now := time.Unix(1_700_002_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	// Submit run at t=2000.
	r1 := postJSON(t, h, "/v1/runs", map[string]any{"input": "old"}, nil)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("POST 1 = %d", r1.Code)
	}
	loop.waitStarted(t)
	var s1 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r1.Body.Bytes(), &s1)

	// Advance clock by 30s (well within RunTTL) then submit run.
	now = time.Unix(1_700_002_030, 0)
	r2 := postJSON(t, h, "/v1/runs", map[string]any{"input": "new"}, nil)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("POST 2 = %d", r2.Code)
	}
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)

	// Sanity: both runs are tracked.
	all := getJSON(t, h, "/v1/runs", nil)
	var allBody struct {
		Runs []map[string]any `json:"runs"`
	}
	json.Unmarshal(all.Body.Bytes(), &allBody)
	if len(allBody.Runs) != 2 {
		t.Fatalf("baseline list count = %d, want 2; body=%s", len(allBody.Runs), all.Body.String())
	}

	// since=1700002015 should return only the second run.
	rec := getJSON(t, h, "/v1/runs?since=1700002015", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Runs []struct {
			RunID     string `json:"run_id"`
			CreatedAt int64  `json:"created_at"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Runs) != 1 || got.Runs[0].RunID != s2.RunID {
		t.Errorf("filtered = %+v, want only %s", got.Runs, s2.RunID)
	}

	loop.release(TurnResult{SessionID: s1.RunID})
	loop.release(TurnResult{SessionID: s2.RunID})
}

// TestAPIServerRunsList_OrderDescNewestFirst proves
// `GET /v1/runs?order=desc` returns runs newest-first by created_at,
// so dashboards showing the most recent runs first don't have to
// reverse the array client-side. Default order remains ascending
// (run_id alpha), matching deterministic enumeration. Invalid order
// values are rejected with 400.
func TestAPIServerRunsList_OrderDescNewestFirst(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	clock := &lockedTestClock{now: time.Unix(1_700_006_000, 0)}
	srv.now = clock.Now
	h := srv.Handler()

	for i := 0; i < 3; i++ {
		clock.Set(time.Unix(1_700_006_000+int64(i), 0))
		if rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil); rec.Code != http.StatusAccepted {
			t.Fatalf("POST %d = %d", i, rec.Code)
		}
	}

	rec := getJSON(t, h, "/v1/runs?order=desc", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Runs []struct {
			CreatedAt int64 `json:"created_at"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Runs) != 3 {
		t.Fatalf("runs count = %d, want 3", len(got.Runs))
	}
	if got.Runs[0].CreatedAt < got.Runs[1].CreatedAt || got.Runs[1].CreatedAt < got.Runs[2].CreatedAt {
		t.Errorf("expected desc order; got created_ats: %d %d %d",
			got.Runs[0].CreatedAt, got.Runs[1].CreatedAt, got.Runs[2].CreatedAt)
	}

	bad := getJSON(t, h, "/v1/runs?order=sideways", nil)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("invalid order status = %d, want 400", bad.Code)
	}
}

// TestAPIServerRunsList_IncludesTotalCount proves the response
// envelope carries `total` — the number of runs after filtering but
// before limit truncation. Pagination consumers need this to know
// how many more pages to fetch and to detect new arrivals between
// polls without separate API calls.
func TestAPIServerRunsList_IncludesTotalCount(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	clock := &lockedTestClock{now: time.Unix(1_700_005_000, 0)}
	srv.now = clock.Now
	h := srv.Handler()

	for i := 0; i < 4; i++ {
		clock.Set(time.Unix(1_700_005_000+int64(i), 0))
		if rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil); rec.Code != http.StatusAccepted {
			t.Fatalf("POST %d = %d", i, rec.Code)
		}
	}

	rec := getJSON(t, h, "/v1/runs?limit=2", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Total int              `json:"total"`
		Runs  []map[string]any `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 4 {
		t.Errorf("total = %d, want 4 (post-filter, pre-limit)", got.Total)
	}
	if len(got.Runs) != 2 {
		t.Errorf("runs = %d, want 2 (limited)", len(got.Runs))
	}
}

// TestAPIServerRunsList_LimitsResultCount proves
// `GET /v1/runs?limit=N` caps the response to N runs (oldest first
// after sort). Operators querying high-cardinality fleets need to
// bound response size; without this they'd have to filter
// client-side after downloading the full set.
func TestAPIServerRunsList_LimitsResultCount(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	clock := &lockedTestClock{now: time.Unix(1_700_004_000, 0)}
	srv.now = clock.Now
	h := srv.Handler()

	// Submit 3 runs with distinct nano-times.
	for i := 0; i < 3; i++ {
		clock.Set(time.Unix(1_700_004_000+int64(i), 0))
		rec := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("POST %d = %d", i, rec.Code)
		}
	}

	// Without limit, expect 3.
	all := getJSON(t, h, "/v1/runs", nil)
	var allBody struct {
		Runs []map[string]any `json:"runs"`
	}
	json.Unmarshal(all.Body.Bytes(), &allBody)
	if len(allBody.Runs) != 3 {
		t.Fatalf("baseline = %d, want 3", len(allBody.Runs))
	}

	// limit=2 returns 2.
	rec := getJSON(t, h, "/v1/runs?limit=2", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Runs []map[string]any `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Runs) != 2 {
		t.Errorf("limit=2 returned %d runs, want 2", len(got.Runs))
	}

	// limit=0 or non-numeric → 400.
	bad := getJSON(t, h, "/v1/runs?limit=banana", nil)
	if bad.Code != http.StatusBadRequest {
		t.Errorf("limit=banana status = %d, want 400", bad.Code)
	}
}

// TestAPIServerRunsList_RejectsUnknownStatusFilter proves an unknown
// `?status=` value returns 400 with the list of valid filters rather
// than silently producing an empty list — a footgun for fleet
// automation typo'ing a status value.
func TestAPIServerRunsList_RejectsUnknownStatusFilter(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}, ResponseStore: NewResponseStore(10)})

	rec := getJSON(t, srv.Handler(), "/v1/runs?status=banana", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"in_progress", "completed", "failed", "stopped"} {
		if !strings.Contains(body, want) {
			t.Errorf("error body missing valid status %q: %s", want, body)
		}
	}
}

// TestAPIServerRunsList_FiltersBySessionIDPrefix proves
// `GET /v1/runs?session_id_prefix=tg-` returns runs whose session_id
// starts with the prefix. Channel-scoped dashboards (e.g. all
// Telegram-prefixed sessions) need prefix-filter without enumerating
// every session_id explicitly.
func TestAPIServerRunsList_FiltersBySessionIDPrefix(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	now := time.Unix(1_700_013_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	// Run in tg- session.
	r1 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "tg-12345"}, nil)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("POST 1 = %d", r1.Code)
	}
	loop.waitStarted(t)
	var s1 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r1.Body.Bytes(), &s1)

	// Run in slack- session.
	now = time.Unix(1_700_013_010, 0)
	r2 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "slack-C123"}, nil)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("POST 2 = %d", r2.Code)
	}
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)

	rec := getJSON(t, h, "/v1/runs?session_id_prefix=tg-", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Total int `json:"total"`
		Runs  []struct {
			SessionID string `json:"session_id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 1 {
		t.Fatalf("total = %d, want 1", got.Total)
	}
	if got.Runs[0].SessionID != "tg-12345" {
		t.Errorf("got %+v, want session tg-12345", got.Runs)
	}

	loop.release(TurnResult{SessionID: s1.RunID})
}

// TestAPIServerRunsList_FiltersBySessionID proves
// `GET /v1/runs?session_id=...` returns only runs matching the
// session_id. Dashboards rendering "all runs in this session" need
// server-side filtering rather than downloading all runs and
// filtering client-side. Same JSON envelope as the other filters.
func TestAPIServerRunsList_FiltersBySessionID(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	now := time.Unix(1_700_011_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	// Run 1 in session A.
	r1 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "sess-A"}, nil)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("POST 1 = %d", r1.Code)
	}
	loop.waitStarted(t)
	var s1 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r1.Body.Bytes(), &s1)

	// Run 2 in session B.
	now = time.Unix(1_700_011_010, 0)
	r2 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping", "session_id": "sess-B"}, nil)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("POST 2 = %d", r2.Code)
	}
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)

	// Filter for session A only.
	rec := getJSON(t, h, "/v1/runs?session_id=sess-A", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Total int `json:"total"`
		Runs  []struct {
			RunID     string `json:"run_id"`
			SessionID string `json:"session_id"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 1 {
		t.Fatalf("total = %d, want 1", got.Total)
	}
	if got.Runs[0].RunID != s1.RunID || got.Runs[0].SessionID != "sess-A" {
		t.Errorf("filtered = %+v, want %s/sess-A", got.Runs, s1.RunID)
	}

	loop.release(TurnResult{SessionID: s1.RunID})
}

// TestAPIServerRunsList_StatusFilterAcceptsMultipleValues proves
// `GET /v1/runs?status=stopped,failed` returns runs matching ANY of
// the comma-separated values. Fleet automation surfacing terminal
// runs (failed OR stopped) needs this to avoid two round-trips.
// Whitespace around commas is tolerated.
func TestAPIServerRunsList_StatusFilterAcceptsMultipleValues(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
		RunTTL:        time.Hour,
	})
	now := time.Unix(1_700_010_000, 0)
	srv.now = func() time.Time { return now }
	h := srv.Handler()

	// Run 1: in_progress (start, never release).
	r1 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("POST 1 = %d", r1.Code)
	}
	loop.waitStarted(t)
	var s1 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r1.Body.Bytes(), &s1)

	// Run 2: stopped.
	now = time.Unix(1_700_010_010, 0)
	r2 := postJSON(t, h, "/v1/runs", map[string]any{"input": "ping"}, nil)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("POST 2 = %d", r2.Code)
	}
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)
	if rec := postJSON(t, h, "/v1/runs/"+s2.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop 2 = %d", rec.Code)
	}

	// Filter for stopped,in_progress should return both.
	rec := getJSON(t, h, "/v1/runs?status=stopped,in_progress", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Total int `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("multi-status total = %d, want 2", got.Total)
	}

	// Whitespace tolerated.
	rec2 := getJSON(t, h, "/v1/runs?status=stopped%2C%20in_progress", nil)
	if rec2.Code != http.StatusOK {
		t.Fatalf("whitespace status = %d; body=%s", rec2.Code, rec2.Body.String())
	}

	loop.release(TurnResult{SessionID: s1.RunID})
}

// TestAPIServerRunsList_FiltersByStatusQuery proves
// `GET /v1/runs?status=in_progress` returns only runs whose status
// matches the filter. Fleet automation polling for stalled in-flight
// runs needs status filtering server-side rather than client-side
// across machines with potentially many tracked runs.
func TestAPIServerRunsList_FiltersByStatusQuery(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	// Submit two runs, stop one, leave the other in-progress.
	r1 := postJSON(t, h, "/v1/runs", map[string]any{"input": "run1"}, nil)
	if r1.Code != http.StatusAccepted {
		t.Fatalf("POST 1 = %d", r1.Code)
	}
	var s1 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r1.Body.Bytes(), &s1)
	loop.waitStarted(t)

	if rec := postJSON(t, h, "/v1/runs/"+s1.RunID+"/stop", nil, nil); rec.Code != http.StatusOK {
		t.Fatalf("stop 1 = %d", rec.Code)
	}

	r2 := postJSON(t, h, "/v1/runs", map[string]any{"input": "run2"}, nil)
	if r2.Code != http.StatusAccepted {
		t.Fatalf("POST 2 = %d", r2.Code)
	}
	var s2 struct {
		RunID string `json:"run_id"`
	}
	json.Unmarshal(r2.Body.Bytes(), &s2)
	loop.waitStarted(t)

	// Filter for in_progress only.
	rec := getJSON(t, h, "/v1/runs?status=in_progress", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Runs []struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if len(got.Runs) != 1 {
		t.Fatalf("filtered runs = %d, want 1", len(got.Runs))
	}
	if got.Runs[0].RunID != s2.RunID || got.Runs[0].Status != "in_progress" {
		t.Errorf("got = %+v, want only run %q in_progress", got.Runs, s2.RunID)
	}

	loop.release(TurnResult{SessionID: s2.RunID})
}

// TestAPIServerRunsList_ReturnsActiveRunsSnapshot proves
// `GET /v1/runs` returns a list of currently-tracked runs with their
// status snapshots so fleet automation auditing in-flight work
// across machines can enumerate running runs without holding SSE
// streams. List is bounded to the registry's live entries; finished
// runs that have been swept or consumed are not present.
func TestAPIServerRunsList_ReturnsActiveRunsSnapshot(t *testing.T) {
	loop := newBlockingRunLoop()
	srv := NewServer(Config{
		ModelName:     "gormes-agent",
		BuildInfo:     BuildInfo{Version: "test-list-attr"},
		Loop:          loop,
		ResponseStore: NewResponseStore(10),
	})
	h := srv.Handler()

	start := postJSON(t, h, "/v1/runs", map[string]any{"input": "hold"}, nil)
	if start.Code != http.StatusAccepted {
		t.Fatalf("POST = %d", start.Code)
	}
	var started struct {
		RunID string `json:"run_id"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode: %v", err)
	}
	loop.waitStarted(t)

	rec := getJSON(t, h, "/v1/runs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var got struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
		Runs []struct {
			RunID  string `json:"run_id"`
			Status string `json:"status"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Build.Version != "test-list-attr" {
		t.Errorf("build.version = %q, want test-list-attr", got.Build.Version)
	}
	if len(got.Runs) != 1 || got.Runs[0].RunID != started.RunID {
		t.Errorf("runs = %+v, want one entry for %q", got.Runs, started.RunID)
	}
	if got.Runs[0].Status != "in_progress" {
		t.Errorf("status = %q, want in_progress", got.Runs[0].Status)
	}

	loop.release(TurnResult{SessionID: started.RunID})
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

// TestAPIServerCapabilities_AdvertisesRunEventsParams proves
// `/v1/capabilities` features advertise the supported query params
// for `GET /v1/runs/{run_id}/events`. SDKs discover whether to
// request backlog-only or keep-after-drain semantics from a single
// capability read.
func TestAPIServerCapabilities_AdvertisesRunEventsParams(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})
	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Features struct {
			RunEventsParams []string `json:"run_events_params"`
		} `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	want := map[string]bool{"backlog_only": false, "keep": false}
	for _, p := range got.Features.RunEventsParams {
		want[p] = true
	}
	for p, present := range want {
		if !present {
			t.Errorf("features.run_events_params missing %q (got %v)", p, got.Features.RunEventsParams)
		}
	}
}

// TestAPIServerCapabilities_AdvertisesRunEventsBacklogCap proves
// `/v1/capabilities` features advertise `run_events_backlog_cap` so
// SDKs subscribing to /v1/runs/{id}/events know how many events they
// might miss if they connect after the run has emitted more than the
// cap. Operators sizing dashboards know the worst-case backlog size.
func TestAPIServerCapabilities_AdvertisesRunEventsBacklogCap(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})
	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Features struct {
			RunEventsBacklogCap int `json:"run_events_backlog_cap"`
		} `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if got.Features.RunEventsBacklogCap != runEventsBacklogCap {
		t.Errorf("run_events_backlog_cap = %d, want %d", got.Features.RunEventsBacklogCap, runEventsBacklogCap)
	}
	if got.Features.RunEventsBacklogCap <= 0 {
		t.Errorf("run_events_backlog_cap must be positive, got %d", got.Features.RunEventsBacklogCap)
	}
}

// TestAPIServerCapabilities_AdvertisesRunLifecycleEvents proves
// `/v1/capabilities` features advertise the typed lifecycle event
// names emitted by the run SSE stream so SDKs and dashboards can
// build switch-statements without hard-coding strings. The list
// covers every event type the registry publishes.
func TestAPIServerCapabilities_AdvertisesRunLifecycleEvents(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})
	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Features struct {
			RunLifecycleEvents []string `json:"run_lifecycle_events"`
		} `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	want := map[string]bool{
		"run.started":   false,
		"run.completed": false,
		"run.failed":    false,
		"run.stopped":   false,
		"message.delta": false,
		"tool.progress": false,
	}
	for _, ev := range got.Features.RunLifecycleEvents {
		want[ev] = true
	}
	for ev, present := range want {
		if !present {
			t.Errorf("features.run_lifecycle_events missing %q (got %v)", ev, got.Features.RunLifecycleEvents)
		}
	}
}

// TestAPIServerCapabilities_AdvertisesRunsListFilters proves the
// `/v1/capabilities` features map advertises `runs_list` plus
// supported query-param filters, so SDKs and dashboards that wire
// up listing UI can discover available filters from a single read
// rather than guessing or hard-coding strings. Same convention as
// the rest of the capabilities advertisement — additive feature
// flags ignored by older clients.
func TestAPIServerCapabilities_AdvertisesRunsListFilters(t *testing.T) {
	srv := NewServer(Config{ModelName: "gormes-agent", Loop: &fakeTurnLoop{}})
	rec := getJSON(t, srv.Handler(), "/v1/capabilities", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Features struct {
			RunsList        bool     `json:"runs_list"`
			RunsListFilters []string `json:"runs_list_filters"`
		} `json:"features"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v\nbody=%s", err, rec.Body.String())
	}
	if !got.Features.RunsList {
		t.Errorf("features.runs_list = false, want true")
	}
	wantFilters := map[string]bool{"status": false, "since": false, "limit": false, "order": false}
	for _, f := range got.Features.RunsListFilters {
		wantFilters[f] = true
	}
	for f, present := range wantFilters {
		if !present {
			t.Errorf("features.runs_list_filters missing %q (got %v)", f, got.Features.RunsListFilters)
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
		t.Fatalf("object = %q, want llm.api_server.capabilities", got.Object)
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
