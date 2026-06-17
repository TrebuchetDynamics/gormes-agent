package apiserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardSSE_ContentType(t *testing.T) {
	s := &Server{}
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/dashboard/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	s.handleDashboardSSE(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestDashboardStatusFragment(t *testing.T) {
	s := &Server{modelName: "gpt-4", providerName: "openai"}
	req := httptest.NewRequest("GET", "/dashboard/status", nil)
	rec := httptest.NewRecorder()
	s.handleDashboardStatusFragment(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "⚕") {
		t.Fatal("status missing caduceus")
	}
	if !strings.Contains(body, "gpt-4") {
		t.Fatal("status missing model name")
	}
}

func TestDashboardMemoryFragment(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/dashboard/memory", nil)
	rec := httptest.NewRecorder()
	s.handleDashboardMemoryFragment(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "Working Memory") {
		t.Fatal("memory missing working memory")
	}
	if !strings.Contains(body, "Long-term") {
		t.Fatal("memory missing long-term")
	}
}

func TestDashboardMemoryFragment_ContentType(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/dashboard/memory", nil)
	rec := httptest.NewRecorder()
	s.handleDashboardMemoryFragment(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q", rec.Header().Get("Content-Type"))
	}
}

func TestAgentExecute_PostOnly(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("GET", "/agent/execute", nil)
	rec := httptest.NewRecorder()
	s.handleAgentExecute(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestAgentExecute_EmptyPrompt(t *testing.T) {
	s := &Server{}
	req := httptest.NewRequest("POST", "/agent/execute?prompt=", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.handleAgentExecute(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("", 10); got != "—" {
		t.Fatalf("truncateStr('') = %q, want —", got)
	}
	if got := truncateStr("short", 10); got != "short" {
		t.Fatalf("truncateStr('short',10) = %q", got)
	}
	if got := truncateStr("very-long-string", 8); !strings.HasSuffix(got, "…") {
		t.Fatalf("truncateStr('very-long-string',8) = %q, want truncated", got)
	}
}

func TestStaticHandler_ServesHTMX(t *testing.T) {
	handler := staticHandler()
	req := httptest.NewRequest("GET", "/static/htmx.min.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("static htmx status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if len(body) < 1000 {
		t.Fatal("htmx.min.js too small — likely not embedded")
	}
	if !strings.Contains(body, "htmx") && !strings.Contains(body, "HTMX") {
		t.Fatal("htmx.min.js missing htmx content")
	}
}

func TestBroadcastSSE(t *testing.T) {
	s := &Server{}
	ch1 := make(chan string, 16)
	ch2 := make(chan string, 16)
	s.registerSSEClient(ch1)
	s.registerSSEClient(ch2)

	s.broadcastSSE("frame", "test data")
	s.broadcastSSE("frame", "more data")

	s.unregisterSSEClient(ch1)
	s.unregisterSSEClient(ch2)

	received1 := drain(ch1)
	received2 := drain(ch2)

	if len(received1) != 2 || len(received2) != 2 {
		t.Fatalf("broadcast: ch1=%d ch2=%d, want 2 each", len(received1), len(received2))
	}
	// Frames must carry the named event so htmx sse-swap="frame" matches, plus
	// the data payload.
	if received1[0] != "event: frame\ndata: test data\n\n" {
		t.Fatalf("broadcast frame format = %q", received1[0])
	}
	if !strings.Contains(received1[1], "event: frame") || !strings.Contains(received1[1], "data: more data") {
		t.Fatalf("broadcast frame missing event/data: %q", received1[1])
	}
}

func drain(ch chan string) []string {
	var out []string
	for {
		select {
		case s := <-ch:
			out = append(out, s)
		default:
			return out
		}
	}
}
