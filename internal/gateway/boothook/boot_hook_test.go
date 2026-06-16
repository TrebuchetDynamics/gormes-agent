package boothook

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/gatewaytest"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func TestStartSkipsMissingBootFile(t *testing.T) {
	client := llm.NewMockClient()

	started := Start(context.Background(), Config{
		Path:   filepath.Join(t.TempDir(), "BOOT.md"),
		Model:  "hermes-agent",
		Client: client,
		Log:    discardBootLogger(),
	})
	if started {
		t.Fatal("Start() = true, want false for missing BOOT.md")
	}
	if got := len(client.Requests()); got != 0 {
		t.Fatalf("client requests = %d, want 0", got)
	}
}

func TestStartAllowsNilContext(t *testing.T) {
	path := writeBootFile(t, "# Startup Checklist\n\n1. Check nil context startup.")
	client := llm.NewMockClient()
	client.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "[SILENT]"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "")

	started := Start(nil, Config{
		Path:   path,
		Model:  "boot-model",
		Client: client,
		Log:    discardBootLogger(),
	})
	if !started {
		t.Fatal("Start() = false, want true")
	}
	gatewaytest.WaitFor(t, 200*time.Millisecond, func() bool {
		return len(client.Requests()) == 1
	})
}

func TestStartRunsWrappedBootPromptInBackground(t *testing.T) {
	path := writeBootFile(t, "# Startup Checklist\n\n1. Check overnight failures.")

	client := llm.NewMockClient()
	client.Script([]llm.Event{
		{Kind: llm.EventToken, Token: "[SILENT]"},
		{Kind: llm.EventDone, FinishReason: "stop"},
	}, "")

	started := Start(context.Background(), Config{
		Path:   path,
		Model:  "boot-model",
		Client: client,
		Log:    discardBootLogger(),
	})
	if !started {
		t.Fatal("Start() = false, want true")
	}

	gatewaytest.WaitFor(t, 200*time.Millisecond, func() bool {
		return len(client.Requests()) == 1
	})

	reqs := client.Requests()
	if len(reqs) != 1 {
		t.Fatalf("client requests = %d, want 1", len(reqs))
	}
	req := reqs[0]
	if req.Model != "boot-model" {
		t.Fatalf("request model = %q, want %q", req.Model, "boot-model")
	}
	if len(req.Messages) != 1 {
		t.Fatalf("request messages len = %d, want 1", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("request message role = %q, want %q", req.Messages[0].Role, "user")
	}
	if !strings.Contains(req.Messages[0].Content, "startup boot checklist") {
		t.Fatalf("request content = %q, want boot-checklist wrapper", req.Messages[0].Content)
	}
	if !strings.Contains(req.Messages[0].Content, "Check overnight failures.") {
		t.Fatalf("request content = %q, want BOOT.md body", req.Messages[0].Content)
	}
	if !strings.Contains(req.Messages[0].Content, "[SILENT]") {
		t.Fatalf("request content = %q, want SILENT instruction", req.Messages[0].Content)
	}
}

func TestLogBootCompletionOnlySuppressesExactSilentReply(t *testing.T) {
	handler := &recordBootLogHandler{}
	log := slog.New(handler)

	logBootCompletion(log, "not [SILENT]; found an issue")

	if len(handler.records) != 1 {
		t.Fatalf("records = %d, want one completion log", len(handler.records))
	}
	if !strings.Contains(handler.records[0], "found an issue") {
		t.Fatalf("completion log = %q, want non-exact SILENT response reported", handler.records[0])
	}
	if strings.Contains(handler.records[0], "nothing to report") {
		t.Fatalf("completion log = %q, non-exact SILENT response should not be suppressed", handler.records[0])
	}
}

func TestStartDoesNotBlockStartupOnBootFailure(t *testing.T) {
	path := writeBootFile(t, "# Startup Checklist\n\n1. Try boot.")
	client := newBlockingBootClient(errors.New("boot failed"))

	start := time.Now()
	started := Start(context.Background(), Config{
		Path:   path,
		Model:  "hermes-agent",
		Client: client,
		Log:    discardBootLogger(),
	})
	if !started {
		t.Fatal("Start() = false, want true")
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("Start() blocked for %s, want background startup", elapsed)
	}

	select {
	case <-client.entered:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("boot client OpenStream was not invoked in background")
	}

	close(client.release)

	select {
	case <-client.done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("boot client did not exit after release")
	}
}

type blockingBootClient struct {
	mu       sync.Mutex
	requests []llm.ChatRequest
	entered  chan struct{}
	release  chan struct{}
	done     chan struct{}
	openErr  error
	once     sync.Once
}

func newBlockingBootClient(openErr error) *blockingBootClient {
	return &blockingBootClient{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan struct{}),
		openErr: openErr,
	}
}

func (c *blockingBootClient) OpenStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	c.mu.Lock()
	c.requests = append(c.requests, req)
	c.mu.Unlock()

	c.once.Do(func() { close(c.entered) })

	select {
	case <-c.release:
	case <-ctx.Done():
	}

	close(c.done)
	return &llm.MockStream{}, c.openErr
}

func (*blockingBootClient) OpenRunEvents(context.Context, string) (llm.RunEventStream, error) {
	return nil, llm.ErrRunEventsNotSupported
}

func (*blockingBootClient) Health(context.Context) error { return nil }

func writeBootFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "BOOT.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(BOOT.md): %v", err)
	}
	return path
}

type recordBootLogHandler struct {
	records []string
}

func (h *recordBootLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordBootLogHandler) Handle(_ context.Context, record slog.Record) error {
	text := record.Message
	record.Attrs(func(attr slog.Attr) bool {
		text += " " + attr.Key + "=" + attr.Value.String()
		return true
	})
	h.records = append(h.records, text)
	return nil
}

func (h *recordBootLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordBootLogHandler) WithGroup(string) slog.Handler      { return h }

func discardBootLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
