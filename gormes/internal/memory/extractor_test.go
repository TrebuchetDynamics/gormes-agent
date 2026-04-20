package memory

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/XelHaku/golang-hermes-agent/gormes/internal/hermes"
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
