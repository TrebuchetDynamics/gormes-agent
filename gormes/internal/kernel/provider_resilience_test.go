package kernel

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestKernel_HonorsProviderRetryAfterDuringReconnect(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "2")
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Hermes-Session-Id", "sess-retry-after")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	defer srv.Close()

	k := newRealKernel(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go k.Run(ctx)

	initial := <-k.Render()
	if initial.Phase != PhaseIdle {
		t.Fatalf("initial phase = %v, want %v", initial.Phase, PhaseIdle)
	}

	start := time.Now()
	if err := k.Submit(PlatformEvent{Kind: PlatformEventSubmit, Text: "hello"}); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	final := waitForFrameMatching(t, k.Render(), func(f RenderFrame) bool {
		return f.Phase == PhaseIdle && len(f.History) >= 2
	}, 5*time.Second)

	if got := attempts.Load(); got != 2 {
		t.Fatalf("OpenStream attempts = %d, want 2", got)
	}
	if elapsed := time.Since(start); elapsed < 1800*time.Millisecond {
		t.Fatalf("elapsed = %v, want >= 1.8s so provider Retry-After is honored", elapsed)
	}
	if final.History[len(final.History)-1].Role != "assistant" {
		t.Fatalf("last history role = %q, want assistant", final.History[len(final.History)-1].Role)
	}
	if got := strings.TrimSpace(final.History[len(final.History)-1].Content); got != "ok" {
		t.Fatalf("assistant content = %q, want ok", got)
	}
}
