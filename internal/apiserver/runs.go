package apiserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultRunStreamTTL = 5 * time.Minute

// runEventsBacklogCap caps how many lifecycle/progress events the
// registry retains per run. A long-running run emitting tool progress
// or token deltas would otherwise grow memory unboundedly. Late SSE
// subscribers see at most the most recent runEventsBacklogCap events;
// older events are dropped FIFO.
const runEventsBacklogCap = 1024

type runRegistry struct {
	mu             sync.Mutex
	ttl            time.Duration
	now            func() time.Time
	runs           map[string]*runRecord
	swept          int
	requestTotal   int
	completedTotal int
	failedTotal    int
	stoppedTotal   int
}

type runRecord struct {
	id           string
	sessionID    string
	createdAt    time.Time
	terminatedAt time.Time
	events       []runEvent
	subscribers  []chan runEvent
	cancel       context.CancelFunc
	done         bool
	failed       bool
	stopped      bool
	consumed     bool
	errMsg       string
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
	r.requestTotal++
}

// stop cancels the in-flight context for the run and marks it stopped.
// Idempotent: calling stop on an already-terminal run is a no-op for
// the lifecycle but still returns true so the handler can respond 200.
// Publishes a typed `run.stopped` lifecycle event into the backlog
// before closing subscribers so SSE consumers see a terminal event
// symmetrical with `run.completed` and `run.failed`.
func (r *runRegistry) stop(id string) bool {
	r.mu.Lock()
	rec := r.runs[id]
	if rec == nil {
		r.mu.Unlock()
		return false
	}
	cancel := rec.cancel
	wasTerminal := rec.done
	var subs []chan runEvent
	if !wasTerminal {
		rec.stopped = true
		rec.done = true
		rec.terminatedAt = r.now()
		r.stoppedTotal++
		stoppedEvent := runEvent{Event: "run.stopped", RunID: id, Timestamp: r.now().Unix()}
		rec.events = append(rec.events, stoppedEvent)
		subs = append([]chan runEvent(nil), rec.subscribers...)
		rec.subscribers = nil
		r.mu.Unlock()
		for _, ch := range subs {
			select {
			case ch <- stoppedEvent:
			default:
			}
			close(ch)
		}
	} else {
		r.mu.Unlock()
	}
	if !wasTerminal && cancel != nil {
		cancel()
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
	if len(rec.events) > runEventsBacklogCap {
		rec.events = rec.events[len(rec.events)-runEventsBacklogCap:]
	}
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
	r.finishWith(id, false, "")
}

func (r *runRegistry) fail(id string, errMsg string) {
	r.finishWith(id, true, errMsg)
}

func (r *runRegistry) finishWith(id string, failed bool, errMsg string) {
	r.mu.Lock()
	rec := r.runs[id]
	if rec == nil {
		r.mu.Unlock()
		return
	}
	if !rec.done {
		if failed {
			r.failedTotal++
		} else {
			r.completedTotal++
		}
		rec.terminatedAt = r.now()
	}
	rec.done = true
	if failed {
		rec.failed = true
		if errMsg != "" {
			rec.errMsg = errMsg
		}
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
// reflects every lifecycle event published so far. LastEventType is
// the type of the most recent event in the backlog (e.g. "run.started",
// "message.delta", "run.completed") for stalled-run detection. Error
// carries the loop's failure message when Status == "failed"; empty
// otherwise.
type runStatusSnapshot struct {
	Status        string
	CreatedAt     int64
	TerminatedAt  int64
	EventsCount   int
	LastEventType string
	LastEventAt   int64
	Error         string
}

// listSnapshots returns a stable-sorted list of all live runs in the
// registry. Finished runs that have been swept or consumed are not
// present. Sort is by run_id ascending so callers see deterministic
// ordering across polls.
func (r *runRegistry) listSnapshots() []namedRunSnapshot {
	r.mu.Lock()
	ids := make([]string, 0, len(r.runs))
	for id := range r.runs {
		ids = append(ids, id)
	}
	r.mu.Unlock()
	sort.Strings(ids)
	out := make([]namedRunSnapshot, 0, len(ids))
	for _, id := range ids {
		if snap, ok := r.snapshot(id); ok {
			out = append(out, namedRunSnapshot{RunID: id, runStatusSnapshot: snap})
		}
	}
	return out
}

type namedRunSnapshot struct {
	RunID string
	runStatusSnapshot
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
		Error:       rec.errMsg,
	}
	if !rec.terminatedAt.IsZero() {
		snap.TerminatedAt = rec.terminatedAt.Unix()
	}
	if n := len(rec.events); n > 0 {
		snap.LastEventType = rec.events[n-1].Event
		snap.LastEventAt = rec.events[n-1].Timestamp
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
		"active":          len(r.runs),
		"orphaned_swept":  r.swept,
		"ttl_seconds":     int(r.ttl.Seconds()),
		"request_total":   r.requestTotal,
		"completed_total": r.completedTotal,
		"failed_total":    r.failedTotal,
		"stopped_total":   r.stoppedTotal,
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if !s.authorized(r) {
			writeOpenAIError(w, http.StatusUnauthorized, "Invalid API key", "invalid_request_error", "", "invalid_api_key")
			return
		}
		s.handleListRuns(w, r)
		return
	}
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

var validRunStatusFilters = []string{"in_progress", "completed", "failed", "stopped"}

func isValidRunStatusFilter(s string) bool {
	for _, v := range validRunStatusFilters {
		if s == v {
			return true
		}
	}
	return false
}

// parseStatusFilters splits a `?status=` value on commas and trims
// whitespace, returning the non-empty filter list. Empty input yields
// nil so callers can short-circuit "no filter".
func parseStatusFilters(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	statusFilters := parseStatusFilters(r.URL.Query().Get("status"))
	for _, f := range statusFilters {
		if !isValidRunStatusFilter(f) {
			writeOpenAIError(w, http.StatusBadRequest,
				"Invalid status filter "+f+"; valid values: "+strings.Join(validRunStatusFilters, ", "),
				"invalid_request_error", "status", "invalid_status_filter")
			return
		}
	}
	var sinceUnix int64
	if since := strings.TrimSpace(r.URL.Query().Get("since")); since != "" {
		parsed, err := strconv.ParseInt(since, 10, 64)
		if err != nil || parsed < 0 {
			writeOpenAIError(w, http.StatusBadRequest,
				"Invalid since timestamp "+since+"; expected non-negative unix seconds",
				"invalid_request_error", "since", "invalid_since_filter")
			return
		}
		sinceUnix = parsed
	}
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeOpenAIError(w, http.StatusBadRequest,
				"Invalid limit "+raw+"; expected positive integer",
				"invalid_request_error", "limit", "invalid_limit")
			return
		}
		limit = parsed
	}
	order := strings.TrimSpace(r.URL.Query().Get("order"))
	if order != "" && order != "asc" && order != "desc" {
		writeOpenAIError(w, http.StatusBadRequest,
			"Invalid order "+order+"; valid values: asc, desc",
			"invalid_request_error", "order", "invalid_order")
		return
	}
	snaps := s.runs.listSnapshots()
	if order == "desc" {
		sort.SliceStable(snaps, func(i, j int) bool {
			return snaps[i].CreatedAt > snaps[j].CreatedAt
		})
	}
	entries := make([]map[string]any, 0, len(snaps))
	total := 0
	for _, snap := range snaps {
		if len(statusFilters) > 0 {
			match := false
			for _, f := range statusFilters {
				if snap.Status == f {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}
		if sinceUnix > 0 && snap.CreatedAt < sinceUnix {
			continue
		}
		total++
		if limit > 0 && len(entries) >= limit {
			continue
		}
		entry := map[string]any{
			"run_id":       snap.RunID,
			"status":       snap.Status,
			"created_at":   snap.CreatedAt,
			"events_count": snap.EventsCount,
		}
		if snap.LastEventType != "" {
			entry["last_event_type"] = snap.LastEventType
		}
		if snap.LastEventAt > 0 {
			entry["last_event_at"] = snap.LastEventAt
		}
		if snap.TerminatedAt > 0 {
			entry["terminated_at"] = snap.TerminatedAt
			if snap.TerminatedAt > snap.CreatedAt {
				entry["duration_seconds"] = snap.TerminatedAt - snap.CreatedAt
			}
		}
		if snap.Error != "" {
			entry["error"] = snap.Error
		}
		entries = append(entries, entry)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"build": s.buildInfo,
		"total": total,
		"runs":  entries,
	})
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
		s.runs.fail(runID, err.Error())
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
		snap, _ := s.runs.snapshot(runID)
		body := map[string]any{
			"build":        s.buildInfo,
			"run_id":       runID,
			"status":       snap.Status,
			"created_at":   snap.CreatedAt,
			"events_count": snap.EventsCount,
		}
		if snap.LastEventType != "" {
			body["last_event_type"] = snap.LastEventType
		}
		if snap.TerminatedAt > 0 {
			body["terminated_at"] = snap.TerminatedAt
		}
		if snap.Error != "" {
			body["error"] = snap.Error
		}
		writeJSON(w, http.StatusOK, body)
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
	body := map[string]any{
		"build":        s.buildInfo,
		"run_id":       suffix,
		"status":       snap.Status,
		"created_at":   snap.CreatedAt,
		"events_count": snap.EventsCount,
	}
	if snap.LastEventType != "" {
		body["last_event_type"] = snap.LastEventType
	}
	if snap.LastEventAt > 0 {
		body["last_event_at"] = snap.LastEventAt
	}
	if snap.TerminatedAt > 0 {
		body["terminated_at"] = snap.TerminatedAt
		if snap.TerminatedAt > snap.CreatedAt {
			body["duration_seconds"] = snap.TerminatedAt - snap.CreatedAt
		}
	}
	if snap.Error != "" {
		body["error"] = snap.Error
	}
	writeJSON(w, http.StatusOK, body)
}

func (s *Server) streamRunEvents(w http.ResponseWriter, r *http.Request, runID string) {
	keep := strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("keep")), "true")
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
		if !keep {
			s.runs.remove(runID)
		}
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
				if !keep {
					s.runs.remove(runID)
				}
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
