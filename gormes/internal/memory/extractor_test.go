package memory

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/store"
)

// fakeLLM implements hermes.Client with scripted responses. Each
// OpenStream call returns the next scripted response (or a default
// empty-graph JSON when scripts are exhausted).
type fakeLLM struct {
	mu        sync.Mutex
	scripts   []fakeResp
	openCalls atomic.Int64
}

type fakeResp struct {
	body string
	err  error
}

func (f *fakeLLM) script(body string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scripts = append(f.scripts, fakeResp{body: body, err: err})
}

func (f *fakeLLM) Health(ctx context.Context) error { return nil }

func (f *fakeLLM) OpenStream(ctx context.Context, _ hermes.ChatRequest) (hermes.Stream, error) {
	f.openCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.scripts) == 0 {
		return &fakeStream{body: `{"entities":[],"relationships":[]}`}, nil
	}
	r := f.scripts[0]
	f.scripts = f.scripts[1:]
	if r.err != nil {
		return nil, r.err
	}
	return &fakeStream{body: r.body}, nil
}

func (f *fakeLLM) OpenRunEvents(ctx context.Context, _ string) (hermes.RunEventStream, error) {
	return nil, hermes.ErrRunEventsNotSupported
}

type fakeStream struct {
	body string
	emit bool
}

func (s *fakeStream) SessionID() string { return "" }
func (s *fakeStream) Close() error      { return nil }
func (s *fakeStream) Recv(ctx context.Context) (hermes.Event, error) {
	select {
	case <-ctx.Done():
		return hermes.Event{}, ctx.Err()
	default:
	}
	if !s.emit {
		s.emit = true
		return hermes.Event{Kind: hermes.EventToken, Token: s.body}, nil
	}
	return hermes.Event{Kind: hermes.EventDone, FinishReason: "stop"}, errStreamEOF
}

var errStreamEOF = errors.New("eof")

func openExtractor(t *testing.T, cfg ExtractorConfig) (*SqliteStore, *Extractor, *fakeLLM) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := OpenSqlite(path, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	llm := &fakeLLM{}
	e := NewExtractor(s, llm, cfg, nil)
	t.Cleanup(func() {
		_ = e.Close(context.Background())
		_ = s.Close(context.Background())
	})
	return s, e, llm
}

func TestExtractor_NewExtractorWithZeroConfigFillsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, _ := OpenSqlite(path, 0, nil)
	defer s.Close(context.Background())

	e := NewExtractor(s, &fakeLLM{}, ExtractorConfig{}, nil)
	if e.cfg.BatchSize != 5 {
		t.Errorf("BatchSize default = %d, want 5", e.cfg.BatchSize)
	}
	if e.cfg.PollInterval != 10*time.Second {
		t.Errorf("PollInterval default = %v, want 10s", e.cfg.PollInterval)
	}
	if e.cfg.MaxAttempts != 5 {
		t.Errorf("MaxAttempts default = %d, want 5", e.cfg.MaxAttempts)
	}
	if e.cfg.CallTimeout != 30*time.Second {
		t.Errorf("CallTimeout default = %v, want 30s", e.cfg.CallTimeout)
	}
}

func TestExtractor_CloseBeforeRunIsNoop(t *testing.T) {
	_, e, _ := openExtractor(t, ExtractorConfig{})
	if err := e.Close(context.Background()); err != nil {
		t.Errorf("Close before Run: %v", err)
	}
	if err := e.Close(context.Background()); err != nil {
		t.Errorf("double Close: %v", err)
	}
}

func TestExtractor_RunExitsOnCtxCancel(t *testing.T) {
	_, e, _ := openExtractor(t, ExtractorConfig{PollInterval: 20 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		e.Run(ctx)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond) // let the loop tick a few times
	cancel()

	select {
	case <-done:
		// Run returned after cancel.
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit within 2s of ctx cancel")
	}
}

// seedTurns inserts N user turns via the store's fast path and waits
// until the persistence worker has drained them all to disk.
func seedTurns(t *testing.T, s *SqliteStore, contents ...string) {
	t.Helper()
	for i, c := range contents {
		payload, _ := json.Marshal(map[string]any{
			"session_id": "sess-extractor-test",
			"content":    c,
			"ts_unix":    int64(1745000000 + i),
		})
		_, _ = s.Exec(context.Background(), store.Command{
			Kind: store.AppendUserTurn, Payload: payload,
		})
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		_ = s.db.QueryRow("SELECT COUNT(*) FROM turns WHERE session_id = 'sess-extractor-test'").Scan(&n)
		if n == len(contents) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("seedTurns timeout — persistence worker did not drain")
}

func TestExtractor_HappyPathPopulatesGraph(t *testing.T) {
	s, e, llm := openExtractor(t, ExtractorConfig{
		PollInterval: 30 * time.Millisecond,
		BatchSize:    5,
		CallTimeout:  2 * time.Second,
	})

	seedTurns(t, s, "I'm working on Arenaton")
	llm.script(`{"entities":[
		{"name":"Jose","type":"PERSON","description":""},
		{"name":"Arenaton","type":"PROJECT","description":""}
	],"relationships":[
		{"source":"Jose","target":"Arenaton","predicate":"WORKS_ON","weight":0.9}
	]}`, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go e.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var e1, nRel int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE extracted = 1`).Scan(&e1)
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM relationships`).Scan(&nRel)
		if e1 >= 1 && nRel >= 1 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	var extracted int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE extracted = 1`).Scan(&extracted)
	if extracted < 1 {
		t.Errorf("turns.extracted=1 count = %d, want >= 1", extracted)
	}
	var nEnt, nRel int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM entities`).Scan(&nEnt)
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM relationships`).Scan(&nRel)
	if nEnt != 2 || nRel != 1 {
		t.Errorf("entities=%d relationships=%d, want 2 and 1", nEnt, nRel)
	}
	if llm.openCalls.Load() != 1 {
		t.Errorf("openCalls = %d, want exactly 1", llm.openCalls.Load())
	}
}

func TestExtractor_EmptyResultStillMarksExtracted(t *testing.T) {
	s, e, llm := openExtractor(t, ExtractorConfig{PollInterval: 30 * time.Millisecond})
	seedTurns(t, s, "weather small talk")
	llm.script(`{"entities":[],"relationships":[]}`, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go e.Run(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var n int
		_ = s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE extracted = 1`).Scan(&n)
		if n >= 1 {
			break
		}
		time.Sleep(15 * time.Millisecond)
	}

	var n int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM turns WHERE extracted = 1`).Scan(&n)
	if n < 1 {
		t.Errorf("empty-result batch not marked extracted=1")
	}
}

func TestExtractor_DeadLettersAfterMaxAttempts(t *testing.T) {
	s, e, llm := openExtractor(t, ExtractorConfig{
		PollInterval: 20 * time.Millisecond,
		BatchSize:    1,
		MaxAttempts:  3,
		CallTimeout:  500 * time.Millisecond,
		// Short backoff so the 3-attempt dead-letter sequence finishes
		// within the test's 3s budget. Production uses 2s/60s defaults.
		BackoffBase: 10 * time.Millisecond,
		BackoffMax:  30 * time.Millisecond,
	})

	seedTurns(t, s, "doomed turn")
	// Always-malformed LLM output.
	for i := 0; i < 5; i++ {
		llm.script("not json", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go e.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var extracted int
		_ = s.db.QueryRow(`SELECT extracted FROM turns WHERE content = 'doomed turn'`).Scan(&extracted)
		if extracted == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	var extracted, attempts int
	_ = s.db.QueryRow(`SELECT extracted, extraction_attempts FROM turns WHERE content = 'doomed turn'`).
		Scan(&extracted, &attempts)
	if extracted != 2 {
		t.Errorf("extracted = %d, want 2 (dead-letter)", extracted)
	}
	if attempts < 3 {
		t.Errorf("attempts = %d, want >= 3", attempts)
	}
}

func TestExtractor_SkipsCronTurns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	store, _ := OpenSqlite(path, 0, nil)
	defer store.Close(context.Background())

	// Seed one normal turn + one cron turn directly via SQL.
	now := time.Now().Unix()
	_, err := store.db.Exec(
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, cron, cron_job_id)
		 VALUES('s', 'user', 'normal turn about Widgets', ?, 'c', 0, NULL),
		        ('cron:j:1', 'user', 'cron turn about Gizmos', ?, 'c', 1, 'j')`,
		now, now+1)
	if err != nil {
		t.Fatal(err)
	}

	ext := NewExtractor(store, &fakeLLM{}, ExtractorConfig{BatchSize: 10}, nil)
	rows, err := ext.pollBatch(context.Background())
	if err != nil {
		t.Fatalf("pollBatch: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("pollBatch returned %d rows, want 1 (normal only)", len(rows))
	}
	if !strings.Contains(rows[0].content, "Widgets") {
		t.Errorf("returned row = %q, want the normal turn (Widgets)", rows[0].content)
	}
}

func TestExtractor_BackoffSleepsBetweenFailures(t *testing.T) {
	s, e, llm := openExtractor(t, ExtractorConfig{
		PollInterval: 5 * time.Millisecond,
		BatchSize:    1,
		MaxAttempts:  10, // high so we don't dead-letter during the test
		CallTimeout:  100 * time.Millisecond,
		BackoffBase:  80 * time.Millisecond,
		BackoffMax:   200 * time.Millisecond,
	})
	seedTurns(t, s, "backoff test")
	for i := 0; i < 20; i++ {
		llm.script("not json", nil)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	start := time.Now()
	go e.Run(ctx)

	<-ctx.Done()
	elapsed := time.Since(start)

	calls := llm.openCalls.Load()
	// Without backoff, 5ms poll interval => ~300 calls in 1.5s.
	// With 80ms base doubling to 160ms to 200ms cap, expect < 15.
	if calls > 15 {
		t.Errorf("openCalls = %d in %v — backoff not applied (expected < 15)", calls, elapsed)
	}
}
