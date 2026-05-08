package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const defaultRunStreamTTL = 5 * time.Minute

type runRegistry struct {
	mu    sync.Mutex
	ttl   time.Duration
	now   func() time.Time
	runs  map[string]*runRecord
	swept int
}

type runRecord struct {
	id          string
	createdAt   time.Time
	events      []runEvent
	subscribers []chan runEvent
	cancel      context.CancelFunc
	done        bool
	failed      bool
	stopped     bool
	consumed    bool
}

type runEvent struct {
	Event     string        `json:"event"`
	RunID     string        `json:"run_id"`
	Timestamp int64         `json:"timestamp"`
	Delta     string        `json:"delta,omitempty"`
	Name      string        `json:"name,omitempty"`
	Preview   string        `json:"preview,omitempty"`
	Status    string        `json:"status,omitempty"`
	Output    string        `json:"output,omitempty"`
	Usage     ResponseUsage `json:"usage,omitempty"`
	Error     string        `json:"error,omitempty"`
}

func newRunRegistry(ttl time.Duration, now func() time.Time) *runRegistry {
	if ttl <= 0 {
		ttl = defaultRunStreamTTL
	}
	if now == nil {
		now = time.Now
	}
	return &runRegistry{
		ttl:  ttl,
		now:  now,
		runs: make(map[string]*runRecord),
	}
}

func (r *runRegistry) setClock(now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	r.mu.Lock()
	r.now = now
	r.mu.Unlock()
}

func (r *runRegistry) create(id string, cancel context.CancelFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[id] = &runRecord{id: id, createdAt: r.now(), cancel: cancel}
}

// stop cancels the in-flight context for the run and marks it stopped.
// Idempotent: calling stop on an already-terminal run is a no-op for
// the lifecycle but still returns true so the handler can respond 200.
func (r *runRegistry) stop(id string) bool {
	r.mu.Lock()
	rec := r.runs[id]
	if rec == nil {
		r.mu.Unlock()
		return false
	}
	cancel := rec.cancel
	wasTerminal := rec.done
	if !wasTerminal {
		rec.stopped = true
		rec.done = true
	}
	subs := append([]chan runEvent(nil), rec.subscribers...)
	rec.subscribers = nil
	r.mu.Unlock()
	if !wasTerminal && cancel != nil {
		cancel()
	}
	if !wasTerminal {
		for _, ch := range subs {
			close(ch)
		}
	}
	return true
}

func (r *runRegistry) publish(id string, ev runEvent) {
	r.mu.Lock()
	rec := r.runs[id]
	if rec == nil {
		r.mu.Unlock()
		return
	}
	rec.events = append(rec.events, ev)
	subs := append([]chan runEvent(nil), rec.subscribers...)
	r.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (r *runRegistry) finish(id string) {
	r.finishWith(id, false)
}

func (r *runRegistry) fail(id string) {
	r.finishWith(id, true)
}

func (r *runRegistry) finishWith(id string, failed bool) {
	r.mu.Lock()
	rec := r.runs[id]
	if rec == nil {
		r.mu.Unlock()
		return
	}
	rec.done = true
	if failed {
		rec.failed = true
	}
	subs := append([]chan runEvent(nil), rec.subscribers...)
	rec.subscribers = nil
	r.mu.Unlock()
	for _, ch := range subs {
		close(ch)
	}
}

func (r *runRegistry) subscribe(id string) ([]runEvent, <-chan runEvent, bool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := r.runs[id]
	if rec == nil {
		return nil, nil, false, false
	}
	rec.consumed = true
	backlog := append([]runEvent(nil), rec.events...)
	if rec.done {
		return backlog, nil, true, true
	}
	ch := make(chan runEvent, 32)
	rec.subscribers = append(rec.subscribers, ch)
	return backlog, ch, true, false
}

// status reports whether the run is known and, if so, whether the
// async turn has finished. Mirrors the lifecycle SSE callers see: a
// run is `in_progress` until publish+finish, then `completed` until
// it is removed (either after stream consumption or orphan sweep).
func (r *runRegistry) status(id string) (string, bool) {
	snapshot, ok := r.snapshot(id)
	if !ok {
		return "", false
	}
	return snapshot.Status, true
}

// runStatusSnapshot is the read-model returned to fleet automation
// polling `/v1/runs/{run_id}`. CreatedAt is unix seconds; EventsCount
// reflects every lifecycle event published so far.
type runStatusSnapshot struct {
	Status      string
	CreatedAt   int64
	EventsCount int
}

func (r *runRegistry) snapshot(id string) (runStatusSnapshot, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec := r.runs[id]
	if rec == nil {
		return runStatusSnapshot{}, false
	}
	snap := runStatusSnapshot{
		CreatedAt:   rec.createdAt.Unix(),
		EventsCount: len(rec.events),
	}
	switch {
	case rec.stopped:
		snap.Status = "stopped"
	case rec.failed:
		snap.Status = "failed"
	case rec.done:
		snap.Status = "completed"
	default:
		snap.Status = "in_progress"
	}
	return snap, true
}

func (r *runRegistry) remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.runs, id)
}

func (r *runRegistry) sweepOrphans() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	var swept int
	for id, rec := range r.runs {
		if rec.consumed || len(rec.subscribers) > 0 {
			continue
		}
		if now.Sub(rec.createdAt) > r.ttl {
			delete(r.runs, id)
			swept++
		}
	}
	r.swept += swept
	return swept
}

func (r *runRegistry) stats() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	return map[string]any{
		"active":         len(r.runs),
		"orphaned_swept": r.swept,
		"ttl_seconds":    int(r.ttl.Seconds()),
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	if s.loop == nil {
		writeOpenAIError(w, http.StatusServiceUnavailable, "Native turn loop is not configured", "server_error", "", "turn_loop_unavailable")
		return
	}
	body, err := readLimitedBody(w, r, s.maxBodyBytes)
	if err != nil {
		writeOpenAIError(w, http.StatusRequestEntityTooLarge, "Request body too large.", "invalid_request_error", "", "body_too_large")
		return
	}
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		writeOpenAIError(w, http.StatusBadRequest, "Invalid JSON in request body", "invalid_request_error", "", "invalid_json")
		return
	}
	runID := "run_" + randomHexFromTime(s.now())
	turnReq, _, errResp := s.buildResponseTurnRequest(req)
	if errResp != nil {
		writeOpenAIError(w, errResp.status, errResp.message, "invalid_request_error", errResp.param, errResp.code)
		return
	}
	if explicit := stringField(req, "session_id"); explicit != "" {
		turnReq.SessionID = explicit
	} else if turnReq.SessionID == "" || strings.HasPrefix(turnReq.SessionID, "api-") {
		turnReq.SessionID = runID
	}
	s.runs.setClock(s.now)
	s.runs.sweepOrphans()
	turnCtx, cancel := context.WithCancel(context.Background())
	s.runs.create(runID, cancel)
	go s.runAsyncTurn(turnCtx, runID, turnReq)
	writeJSON(w, http.StatusAccepted, map[string]any{"build": s.buildInfo, "run_id": runID, "status": "started"})
}

func (s *Server) runAsyncTurn(ctx context.Context, runID string, turnReq TurnRequest) {
	now := s.now().Unix()
	s.runs.publish(runID, runEvent{Event: "run.started", RunID: runID, Timestamp: now})
	result, err := s.loop.StreamTurn(ctx, turnReq, StreamCallbacks{
		OnToken: func(token string) error {
			s.runs.publish(runID, runEvent{Event: "message.delta", RunID: runID, Timestamp: s.now().Unix(), Delta: token})
			return nil
		},
		OnToolProgress: func(progress ToolProgressEvent) error {
			s.runs.publish(runID, runEvent{
				Event:     "tool.progress",
				RunID:     runID,
				Timestamp: s.now().Unix(),
				Name:      progress.Name,
				Preview:   progress.Preview,
				Status:    progress.Status,
			})
			return nil
		},
	})
	if err != nil {
		s.runs.publish(runID, runEvent{Event: "run.failed", RunID: runID, Timestamp: s.now().Unix(), Error: err.Error()})
		s.runs.fail(runID)
		return
	}
	s.runs.publish(runID, runEvent{
		Event:     "run.completed",
		RunID:     runID,
		Timestamp: s.now().Unix(),
		Output:    result.Content,
		Usage: ResponseUsage{
			InputTokens:  result.Usage.PromptTokens,
			OutputTokens: result.Usage.CompletionTokens,
			TotalTokens:  result.Usage.TotalTokens,
		},
	})
	s.runs.finish(runID)
}

func (s *Server) handleRunEvents(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
		return
	}
	suffix := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	if runID, ok := strings.CutSuffix(suffix, "/events"); ok {
		if runID == "" || strings.Contains(runID, "/") {
			writeOpenAIError(w, http.StatusNotFound, "Run not found", "invalid_request_error", "", "run_not_found")
			return
		}
		if r.Method != http.MethodGet {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		s.streamRunEvents(w, r, runID)
		return
	}
	if runID, ok := strings.CutSuffix(suffix, "/stop"); ok {
		if runID == "" || strings.Contains(runID, "/") {
			writeOpenAIError(w, http.StatusNotFound, "Run not found", "invalid_request_error", "", "run_not_found")
			return
		}
		if r.Method != http.MethodPost {
			writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
			return
		}
		if !s.runs.stop(runID) {
			writeOpenAIError(w, http.StatusNotFound, "Run not found: "+runID, "invalid_request_error", "", "run_not_found")
			return
		}
		status, _ := s.runs.status(runID)
		writeJSON(w, http.StatusOK, map[string]any{
			"build":  s.buildInfo,
			"run_id": runID,
			"status": status,
		})
		return
	}
	if suffix == "" || strings.Contains(suffix, "/") {
		writeOpenAIError(w, http.StatusNotFound, "Run not found", "invalid_request_error", "", "run_not_found")
		return
	}
	if r.Method != http.MethodGet {
		writeOpenAIError(w, http.StatusMethodNotAllowed, "Method not allowed", "invalid_request_error", "", "method_not_allowed")
		return
	}
	snap, ok := s.runs.snapshot(suffix)
	if !ok {
		writeOpenAIError(w, http.StatusNotFound, "Run not found: "+suffix, "invalid_request_error", "", "run_not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build":        s.buildInfo,
		"run_id":       suffix,
		"status":       snap.Status,
		"created_at":   snap.CreatedAt,
		"events_count": snap.EventsCount,
	})
}

func (s *Server) streamRunEvents(w http.ResponseWriter, r *http.Request, runID string) {
	backlog, ch, exists, done := s.runs.subscribe(runID)
	if !exists {
		writeOpenAIError(w, http.StatusNotFound, "Run not found: "+runID, "invalid_request_error", "", "run_not_found")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	for _, ev := range backlog {
		writeSSEData(w, ev)
	}
	flush(w)
	if done {
		writeSSEComment(w, "stream closed")
		flush(w)
		s.runs.remove(runID)
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				writeSSEComment(w, "stream closed")
				flush(w)
				s.runs.remove(runID)
				return
			}
			writeSSEData(w, ev)
			flush(w)
		}
	}
}

func (s *Server) sweepOrphanedRuns() int {
	s.runs.setClock(s.now)
	return s.runs.sweepOrphans()
}

func (s *Server) runHealthStatus() map[string]any {
	return s.runs.stats()
}

func writeSSEComment(w http.ResponseWriter, text string) error {
	_, err := w.Write([]byte(": " + text + "\n\n"))
	return err
}
