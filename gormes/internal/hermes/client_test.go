package hermes

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"
)

const sseHappy = `data: {"id":"1","choices":[{"delta":{"content":"hel"}}]}

data: {"id":"1","choices":[{"delta":{"content":"lo","reasoning":"thinking..."}}]}

data: {"id":"1","choices":[{"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2}}

data: [DONE]

`

func TestOpenStream_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Hermes-Session-Id", "sess-42")
		w.WriteHeader(200)
		bw := bufio.NewWriter(w)
		fmt.Fprint(bw, sseHappy)
		bw.Flush()
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	s, err := c.OpenStream(context.Background(), ChatRequest{
		Model:    "hermes-agent",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var tokens, reasoning strings.Builder
	var final Event
	for {
		e, rerr := s.Recv(context.Background())
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			t.Fatal(rerr)
		}
		if e.Kind == EventToken {
			tokens.WriteString(e.Token)
		}
		if e.Kind == EventReasoning {
			reasoning.WriteString(e.Reasoning)
		}
		if e.Kind == EventDone {
			final = e
			break
		}
	}
	if tokens.String() != "hello" {
		t.Errorf("tokens = %q", tokens.String())
	}
	if reasoning.String() != "thinking..." {
		t.Errorf("reasoning = %q", reasoning.String())
	}
	if final.FinishReason != "stop" {
		t.Errorf("finish_reason = %q", final.FinishReason)
	}
	if final.TokensIn != 5 || final.TokensOut != 2 {
		t.Errorf("usage = %d/%d, want 5/2", final.TokensIn, final.TokensOut)
	}
	if s.SessionID() != "sess-42" {
		t.Errorf("SessionID = %q, want sess-42", s.SessionID())
	}
}

func TestOpenStream_Retry_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "slow down", 429)
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	_, err := c.OpenStream(context.Background(), ChatRequest{Model: "hermes-agent"})
	if err == nil {
		t.Fatal("expected error")
	}
	if Classify(err) != ClassRetryable {
		t.Errorf("Classify = %v, want ClassRetryable", Classify(err))
	}
}

// TestOpenStream_DropNoLeak is the goroutine-leak invariant required by the
// architecture: a mid-stream TCP drop MUST NOT leave reader goroutines behind.
func TestOpenStream_DropNoLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		// Flush one partial SSE frame then hijack + abruptly close the conn.
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("ResponseWriter does not support Hijack")
			return
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		_ = conn.Close()
	}))
	defer srv.Close()

	// Settle any test-harness goroutines first.
	time.Sleep(50 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	for i := 0; i < 20; i++ {
		c := NewHTTPClient(srv.URL, "")
		s, err := c.OpenStream(context.Background(), ChatRequest{
			Model:    "hermes-agent",
			Messages: []Message{{Role: "user", Content: "hi"}},
		})
		if err != nil {
			continue
		}
		// Drain until EOF or any error.
		for {
			_, rerr := s.Recv(context.Background())
			if rerr != nil {
				break
			}
		}
		_ = s.Close()
	}

	time.Sleep(100 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > baseline+3 {
		t.Errorf("goroutine leak: baseline=%d after=%d (delta=%d)", baseline, after, after-baseline)
	}
}

func TestHealth_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	c := NewHTTPClient(srv.URL, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Health(ctx); err != nil {
		t.Errorf("Health: %v", err)
	}
}
