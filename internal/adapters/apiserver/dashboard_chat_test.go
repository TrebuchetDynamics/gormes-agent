package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

// resettableLoop is a TurnLoop that also satisfies SessionResetter.
type resettableLoop struct {
	fakeStreamLoop
	resetCalls int
	resetErr   error
}

func (r *resettableLoop) ResetSession() error {
	r.resetCalls++
	return r.resetErr
}

// resetRecordingKernel is a minimal kernelSubmitter for exercising
// KernelTurnLoop.ResetSession.
type resetRecordingKernel struct {
	render chan kernel.RenderFrame
	resets int
}

func (k *resetRecordingKernel) Submit(kernel.PlatformEvent) error { return nil }
func (k *resetRecordingKernel) Render() <-chan kernel.RenderFrame { return k.render }
func (k *resetRecordingKernel) ResetSession() error               { k.resets++; return nil }

func postForm(t *testing.T, h http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "127.0.0.1"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	return rec
}

// fakeStreamLoop is a deterministic TurnLoop test double: it replays a fixed
// set of tokens to the stream callback, then returns a final result or error.
type fakeStreamLoop struct {
	tokens []string
	reply  string
	err    error
}

func (f fakeStreamLoop) RunTurn(_ context.Context, _ TurnRequest) (TurnResult, error) {
	return TurnResult{Content: f.reply}, f.err
}

func (f fakeStreamLoop) StreamTurn(_ context.Context, _ TurnRequest, cb StreamCallbacks) (TurnResult, error) {
	for _, tok := range f.tokens {
		if cb.OnToken != nil {
			if err := cb.OnToken(tok); err != nil {
				return TurnResult{}, err
			}
		}
	}
	return TurnResult{Content: f.reply}, f.err
}

// drainSSE collects everything broadcast to ch until it goes quiet.
func drainSSE(ch chan string) []string {
	var out []string
	for {
		select {
		case msg := <-ch:
			out = append(out, msg)
		case <-time.After(50 * time.Millisecond):
			return out
		}
	}
}

func TestRunDashboardChatTurnStreamsEscapedTokens(t *testing.T) {
	srv := NewServer(Config{
		ModelName:          "gormes-agent",
		DashboardBoundHost: "127.0.0.1",
		Loop:               fakeStreamLoop{tokens: []string{"Hello ", "<b>world</b>"}, reply: "Hello <b>world</b>"},
	})
	ch := make(chan string, 32)
	srv.registerSSEClient(ch)
	defer srv.unregisterSSEClient(ch)

	srv.runDashboardChatTurn("hi there")

	frames := strings.Join(drainSSE(ch), "\n")
	if !strings.Contains(frames, "Hello ") {
		t.Fatalf("expected first token broadcast; got: %s", frames)
	}
	if strings.Contains(frames, "<b>world</b>") {
		t.Fatalf("streamed token was not HTML-escaped: %s", frames)
	}
	if !strings.Contains(frames, "&lt;b&gt;world&lt;/b&gt;") {
		t.Fatalf("expected escaped token broadcast; got: %s", frames)
	}
}

func TestRunDashboardChatTurnBroadcastsError(t *testing.T) {
	srv := NewServer(Config{
		ModelName:          "gormes-agent",
		DashboardBoundHost: "127.0.0.1",
		Loop:               fakeStreamLoop{err: errors.New("provider exploded")},
	})
	ch := make(chan string, 8)
	srv.registerSSEClient(ch)
	defer srv.unregisterSSEClient(ch)

	srv.runDashboardChatTurn("hi")

	frames := strings.Join(drainSSE(ch), "\n")
	if !strings.Contains(frames, "provider exploded") {
		t.Fatalf("expected error broadcast; got: %s", frames)
	}
	if !strings.Contains(frames, "line error") {
		t.Fatalf("expected error-styled frame; got: %s", frames)
	}
}

func TestRunDashboardChatTurnNonStreamingEmitsFullReply(t *testing.T) {
	srv := NewServer(Config{
		ModelName:          "gormes-agent",
		DashboardBoundHost: "127.0.0.1",
		Loop:               fakeStreamLoop{reply: "the whole answer"}, // no tokens streamed
	})
	ch := make(chan string, 8)
	srv.registerSSEClient(ch)
	defer srv.unregisterSSEClient(ch)

	srv.runDashboardChatTurn("hi")

	frames := strings.Join(drainSSE(ch), "\n")
	if !strings.Contains(frames, "the whole answer") {
		t.Fatalf("expected full reply broadcast when nothing streamed; got: %s", frames)
	}
}

func TestDashboardNewChatResetsAndRotatesSession(t *testing.T) {
	loop := &resettableLoop{}
	srv := NewServer(Config{ModelName: "gormes-agent", DashboardBoundHost: "127.0.0.1", Loop: loop})
	before := srv.currentChatSessionID()
	if before != dashboardChatSessionID {
		t.Fatalf("default chat session id = %q, want %q", before, dashboardChatSessionID)
	}
	rec := postForm(t, srv.Handler(), "/agent/reset", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /agent/reset = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "new chat") {
		t.Fatalf("reset body missing confirmation; got: %s", rec.Body.String())
	}
	if loop.resetCalls != 1 {
		t.Fatalf("kernel reset calls = %d, want 1", loop.resetCalls)
	}
	after := srv.currentChatSessionID()
	if after == before || !strings.HasPrefix(after, "dashboard-chat-") {
		t.Fatalf("chat session id not rotated: before=%q after=%q", before, after)
	}
}

func TestDashboardNewChatWithoutResetterStillRotates(t *testing.T) {
	// fakeStreamLoop does not implement SessionResetter -> kernel reset is
	// skipped but the feed clears and the session id still rotates.
	srv := NewServer(Config{ModelName: "gormes-agent", DashboardBoundHost: "127.0.0.1", Loop: fakeStreamLoop{}})
	before := srv.currentChatSessionID()
	rec := postForm(t, srv.Handler(), "/agent/reset", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /agent/reset = %d, want 200", rec.Code)
	}
	if srv.currentChatSessionID() == before {
		t.Fatalf("chat session id not rotated without a resetter")
	}
}

func TestKernelTurnLoopResetSessionCallsKernel(t *testing.T) {
	k := &resetRecordingKernel{render: make(chan kernel.RenderFrame, 1)}
	loop := NewKernelTurnLoop(k)
	if err := loop.ResetSession(); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}
	if k.resets != 1 {
		t.Fatalf("kernel ResetSession calls = %d, want 1", k.resets)
	}
}

func TestAgentExecuteDegradesWithoutLoop(t *testing.T) {
	// newUITestServer wires no Loop -> chat is display-only.
	h := newUITestServer().Handler()
	rec := postForm(t, h, "/agent/execute", "prompt=hello")
	if got := rec.Body.String(); !strings.Contains(got, "display-only") {
		t.Fatalf("expected display-only degrade notice; got: %s", got)
	}
}
