package apiserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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

func TestAgentExecuteDegradesWithoutLoop(t *testing.T) {
	// newUITestServer wires no Loop -> chat is display-only.
	h := newUITestServer().Handler()
	rec := postForm(t, h, "/agent/execute", "prompt=hello")
	if got := rec.Body.String(); !strings.Contains(got, "display-only") {
		t.Fatalf("expected display-only degrade notice; got: %s", got)
	}
}
