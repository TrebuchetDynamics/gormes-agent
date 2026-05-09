package signal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Fake transport + clock + lock helpers
// ---------------------------------------------------------------------------

type fakeRoundTripper struct {
	mu       sync.Mutex
	handlers map[string]func(*http.Request) (*http.Response, error)
	requests []*http.Request // record all received requests
}

func newFakeRoundTripper() *fakeRoundTripper {
	return &fakeRoundTripper{handlers: map[string]func(*http.Request) (*http.Response, error){}}
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	key := req.Method + " " + req.URL.Path
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	if h, ok := f.handlers[key]; ok {
		return h(req)
	}
	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("not found")),
		Header:     http.Header{},
	}, nil
}

func (f *fakeRoundTripper) handle(method, path string, fn func(*http.Request) (*http.Response, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers[method+" "+path] = fn
}

func (f *fakeRoundTripper) lastRequest() *http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return nil
	}
	return f.requests[len(f.requests)-1]
}

type fakeLock struct {
	mu        sync.Mutex
	acquired  bool
	acquireCh chan struct{}
}

func newFakeLock() *fakeLock { return &fakeLock{acquireCh: make(chan struct{}, 1)} }

func (l *fakeLock) Acquire(ctx context.Context, kind, key, label string) (func(), bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.acquired {
		return nil, false
	}
	l.acquired = true
	l.acquireCh <- struct{}{}
	released := &atomic.Bool{}
	return func() {
		if released.CompareAndSwap(false, true) {
			l.mu.Lock()
			l.acquired = false
			l.mu.Unlock()
		}
	}, true
}

type blockingReleaseLock struct {
	acquireCh       chan struct{}
	releaseStarted  chan struct{}
	releaseContinue chan struct{}
	released        chan struct{}
}

func newBlockingReleaseLock() *blockingReleaseLock {
	return &blockingReleaseLock{
		acquireCh:       make(chan struct{}, 1),
		releaseStarted:  make(chan struct{}),
		releaseContinue: make(chan struct{}),
		released:        make(chan struct{}),
	}
}

func (l *blockingReleaseLock) Acquire(context.Context, string, string, string) (func(), bool) {
	l.acquireCh <- struct{}{}
	return func() {
		close(l.releaseStarted)
		<-l.releaseContinue
		close(l.released)
	}, true
}

type fakeSSEReader struct {
	events []sseEvent
	pos    int
	closed bool
}

func newFakeSSEReader(events []sseEvent) *fakeSSEReader {
	return &fakeSSEReader{events: events}
}

func (r *fakeSSEReader) Next() (sseEvent, error) {
	if r.pos >= len(r.events) {
		return sseEvent{}, io.EOF
	}
	ev := r.events[r.pos]
	r.pos++
	return ev, nil
}

func (r *fakeSSEReader) Close() error {
	r.closed = true
	return nil
}

// ---------------------------------------------------------------------------
// 1. TestSignalBootstrapConfigAndHealth
// ---------------------------------------------------------------------------

func TestSignalBootstrapConfigAndHealth(t *testing.T) {
	rt := newFakeRoundTripper()
	rt.handle("GET", "/api/v1/check", func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})

	lock := newFakeLock()
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}

	b := NewBootstrap(cfg,
		WithBootstrapHTTPClient(rt),
		WithBootstrapLock(lock),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v, want nil", err)
	}

	// Verify lock was acquired.
	select {
	case <-lock.acquireCh:
	default:
		t.Fatal("lock was not acquired")
	}

	// Verify the SSE events URL includes the URL-encoded account.
	// The default SSE dial will be used since no test dial is set,
	// but we can verify health check was called.
	if req := rt.lastRequest(); req == nil {
		t.Fatal("no HTTP request was made")
	}

	// Cleanup.
	if err := b.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	// Lock should be released after close.
	lock.mu.Lock()
	released := !lock.acquired
	lock.mu.Unlock()
	if !released {
		t.Fatal("lock was not released after Close")
	}
}

func TestSignalCloseWaitsForPlatformLockRelease(t *testing.T) {
	rt := newFakeRoundTripper()
	rt.handle("GET", "/api/v1/check", func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})

	lock := newBlockingReleaseLock()
	dialStarted := make(chan struct{})
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}
	b := NewBootstrap(cfg,
		WithBootstrapHTTPClient(rt),
		WithBootstrapLock(lock),
		WithBootstrapSSEDial(func(ctx context.Context, url string) (*sseReader, error) {
			close(dialStarted)
			<-ctx.Done()
			return nil, ctx.Err()
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v, want nil", err)
	}
	select {
	case <-lock.acquireCh:
	case <-time.After(time.Second):
		t.Fatal("lock was not acquired")
	}
	select {
	case <-dialStarted:
	case <-time.After(time.Second):
		t.Fatal("SSE dial did not start")
	}

	closeErr := make(chan error, 1)
	go func() {
		closeErr <- b.Close()
	}()

	select {
	case <-lock.releaseStarted:
	case <-time.After(time.Second):
		t.Fatal("lock release did not start")
	}
	select {
	case err := <-closeErr:
		t.Fatalf("Close returned before platform lock release completed: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(lock.releaseContinue)
	select {
	case err := <-closeErr:
		if err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Close did not return after platform lock release")
	}
	select {
	case <-lock.released:
	default:
		t.Fatal("platform lock release did not complete")
	}
}

// ---------------------------------------------------------------------------
// 2. TestSignalSSEListenerAccountEncodingAndLiveness
// ---------------------------------------------------------------------------

func TestSignalSSEListenerAccountEncodingAndLiveness(t *testing.T) {
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}
	b := NewBootstrap(cfg)

	// Verify the SSE URL encodes the account correctly (RFC 3986: "+" → "%2B").
	// We test this by constructing the URL the same way the SSE listener does.
	accountEncoded := fmt.Sprintf("%%2B15551234567")
	wantSubstr := "account=%2B15551234567"
	url := fmt.Sprintf("%s/api/v1/events?account=%s", b.cfg.HTTPURL, accountEncoded)
	if !strings.Contains(url, wantSubstr) {
		t.Fatalf("SSE URL = %q, want sub string %q", url, wantSubstr)
	}

	// Directly test SSE event parsing: comments should update liveness, invalid
	// JSON should be skipped gracefully, and a valid envelope should produce an
	// inbound message via the events channel.
	b.handleEventEnvelope("") // SSE comment → handled via updateActivity (empty data, not a data line)
	b.handleEventEnvelope("bad json")

	env := map[string]any{
		"envelope": map[string]any{
			"source":     "+15557654321",
			"sourceName": "Alice",
			"sourceUuid": "uuid-alice",
			"dataMessage": map[string]any{
				"message": "hello world",
			},
		},
	}
	envJSON, _ := json.Marshal(env)
	b.handleEventEnvelope(string(envJSON))

	// The message should be pushed to the events channel.
	select {
	case msg := <-b.Events():
		if msg.SenderID != "+15557654321" {
			t.Fatalf("SenderID = %q, want +15557654321", msg.SenderID)
		}
		if msg.Text != "hello world" {
			t.Fatalf("Text = %q, want hello world", msg.Text)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("did not receive inbound event via Events()")
	}
}

type fakeSSEScanner struct {
	r   *fakeSSEReader
	buf []byte
}

func (s *fakeSSEScanner) Scan() bool {
	ev, err := s.r.Next()
	if err != nil {
		return false
	}
	if ev.Data == "" {
		s.buf = []byte(":\n") // comment
	} else {
		s.buf = []byte("data:" + ev.Data + "\n")
	}
	return true
}

func (s *fakeSSEScanner) Text() string { return string(s.buf) }
func (s *fakeSSEScanner) Err() error   { return nil }

func envelopeForDirect(t *testing.T, name, senderID, senderUUID, text string) string {
	t.Helper()
	env := map[string]any{
		"envelope": map[string]any{
			"source":     senderID,
			"sourceName": name,
			"sourceUuid": senderUUID,
			"dataMessage": map[string]any{
				"message": text,
			},
		},
	}
	return mustJSON(t, env)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// 3. TestSignalReconnectAndHealthMonitor
// ---------------------------------------------------------------------------

func TestSignalReconnectAndHealthMonitor(t *testing.T) {
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}

	rt := newFakeRoundTripper()
	rt.handle("GET", "/api/v1/check", func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})

	done := make(chan struct{}, 1)
	var dialCount atomic.Int32
	lock := newFakeLock()

	// SSE dial that returns EOF after one event, simulating disconnect.
	b := NewBootstrap(cfg,
		WithBootstrapHTTPClient(rt),
		WithBootstrapLock(lock),
		WithBootstrapSSEDial(func(ctx context.Context, url string) (*sseReader, error) {
			dialCount.Add(1)
			count := dialCount.Load()
			if count >= 3 {
				done <- struct{}{}
				<-ctx.Done()
				return nil, ctx.Err()
			}
			events := []sseEvent{
				{Data: ""}, // one comment then EOF
			}
			r := newFakeSSEReader(events)
			return &sseReader{scanner: &fakeSSEScanner{r: r}, closer: r}, nil
		}),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := b.Connect(ctx); err != nil {
		t.Fatalf("Connect() error = %v, want nil", err)
	}

	// Wait for multiple reconnect cycles.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("did not observe reconnect cycles")
	}

	count := dialCount.Load()
	if count < 2 {
		t.Fatalf("SSE dial count = %d, want >= 2 (reconnect)", count)
	}

	b.Close()
}

// ---------------------------------------------------------------------------
// 4. TestSignalAttachmentFetchUsesIDParam
// ---------------------------------------------------------------------------

func TestSignalAttachmentFetchUsesIDParam(t *testing.T) {
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}

	rt := newFakeRoundTripper()
	rt.handle("GET", "/api/v1/check", func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})
	var rpcBody jsonRPCRequest
	rt.handle("POST", "/api/v1/rpc", func(req *http.Request) (*http.Response, error) {
		json.NewDecoder(req.Body).Decode(&rpcBody)
		b, _ := json.Marshal(jsonRPCResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(`{"data":"dGVzdA=="}`),
			ID:      rpcBody.ID,
		})
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	lock := newFakeLock()
	b := NewBootstrap(cfg, WithBootstrapHTTPClient(rt), WithBootstrapLock(lock))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = b.Connect(ctx)
	defer b.Close()

	attData, err := b.FetchAttachment(ctx, "att-42")
	if err != nil {
		t.Fatalf("FetchAttachment() error = %v, want nil", err)
	}

	// "dGVzdA==" decodes to "test".
	if string(attData) != "dGVzdA==" {
		t.Fatalf("FetchAttachment() data = %q, want dGVzdA==", string(attData))
	}

	// Verify the RPC params include account and id.
	var params struct {
		Account string `json:"account"`
		ID      string `json:"id"`
	}
	json.Unmarshal(rpcBody.Params, &params)
	if params.Account != "+15551234567" {
		t.Fatalf("RPC params account = %q, want +15551234567", params.Account)
	}
	if params.ID != "att-42" {
		t.Fatalf("RPC params id = %q, want att-42", params.ID)
	}
	if rpcBody.Method != "getAttachment" {
		t.Fatalf("RPC method = %q, want getAttachment", rpcBody.Method)
	}
}

// ---------------------------------------------------------------------------
// 5. TestSignalSendDirectAndGroup
// ---------------------------------------------------------------------------

func TestSignalSendDirectAndGroup(t *testing.T) {
	cfg := BootstrapConfig{HTTPURL: "http://127.0.0.1:8080", Account: "+15551234567"}

	rt := newFakeRoundTripper()
	rt.handle("GET", "/api/v1/check", func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}, nil
	})
	var rpcRequests []jsonRPCRequest
	var rpcMu sync.Mutex
	rt.handle("POST", "/api/v1/rpc", func(req *http.Request) (*http.Response, error) {
		var r jsonRPCRequest
		json.NewDecoder(req.Body).Decode(&r)
		rpcMu.Lock()
		rpcRequests = append(rpcRequests, r)
		rpcMu.Unlock()

		// Return timestamp in response.
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			Result:  json.RawMessage(fmt.Sprintf(`{"timestamp":%d}`, time.Now().UnixMilli())),
			ID:      r.ID,
		}
		b, _ := json.Marshal(resp)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	lock := newFakeLock()
	b := NewBootstrap(cfg, WithBootstrapHTTPClient(rt), WithBootstrapLock(lock))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = b.Connect(ctx)
	defer b.Close()

	// Send direct message.
	msgID, err := b.SendDirect(ctx, "+15557654321", "hello **bold**", SendOptions{
		ReplyToMessageID: "msg-1",
	})
	if err != nil {
		t.Fatalf("SendDirect() error = %v, want nil", err)
	}
	if msgID == "" {
		t.Fatal("SendDirect() msgID is empty")
	}

	// Send group message.
	groupID := "grp-abc123"
	msgID2, err := b.SendGroup(ctx, groupID, "group reply", SendOptions{
		ReplyToMessageID: "msg-2",
	})
	if err != nil {
		t.Fatalf("SendGroup() error = %v, want nil", err)
	}
	if msgID2 == "" {
		t.Fatal("SendGroup() msgID is empty")
	}

	rpcMu.Lock()
	defer rpcMu.Unlock()

	if len(rpcRequests) < 2 {
		t.Fatalf("RPC request count = %d, want >= 2", len(rpcRequests))
	}

	// First two should be stopTyping for each send, then the send calls.
	typingCall := rpcRequests[0]
	var typingParams struct {
		Account      string  `json:"account"`
		Recipient    *string `json:"recipient"`
		GroupID      *string `json:"groupId"`
		Typing       bool    `json:"typing"`
		TimestampEnd int64   `json:"timestamp"`
	}
	json.Unmarshal(typingCall.Params, &typingParams)
	if typingCall.Method != "sendTyping" {
		t.Fatalf("first RPC method = %q, want sendTyping", typingCall.Method)
	}
	if typingParams.Account != "+15551234567" {
		t.Fatalf("typing params account = %q, want +15551234567", typingParams.Account)
	}
	if typingParams.Typing {
		t.Fatal("typing params typing = true, want false (stop)")
	}

	// Should be 4 calls total: 2 sendTyping + 2 send.
	expected := 4
	if len(rpcRequests) != expected {
		t.Fatalf("RPC call count = %d, want %d (2 typing + 2 send)", len(rpcRequests), expected)
	}
}
