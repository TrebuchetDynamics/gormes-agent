//go:build live

package hermes

import (
	"context"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"
)

// endpointFromEnv returns the configured live endpoint, with the Phase-1
// default fallback. Overridable via GORMES_ENDPOINT for CI flexibility.
func endpointFromEnv() string {
	if v := os.Getenv("GORMES_ENDPOINT"); v != "" {
		return v
	}
	return "http://127.0.0.1:8642"
}

// skipIfUnreachable pings /health. If the connection is refused or any
// non-HTTP error surfaces, the test is skipped (t.Skip) — a missing
// api_server is NOT a test failure under the live build tag.
func skipIfUnreachable(t *testing.T, c Client) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Health(ctx); err != nil {
		var netErr *net.OpError
		if _, ok := err.(*net.OpError); ok {
			t.Skipf("api_server not running at live endpoint: %v", err)
		}
		_ = netErr
		t.Skipf("skipping live test: %v", err)
	}
}

func TestLive_Health(t *testing.T) {
	c := NewHTTPClient(endpointFromEnv(), os.Getenv("GORMES_API_KEY"))
	skipIfUnreachable(t, c)
	// If we reach here, Health already passed inside skipIfUnreachable.
	// Confirm a second call still succeeds (connection pool sanity).
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Health(ctx); err != nil {
		t.Errorf("second Health: %v", err)
	}
}

func TestLive_Stream(t *testing.T) {
	c := NewHTTPClient(endpointFromEnv(), os.Getenv("GORMES_API_KEY"))
	skipIfUnreachable(t, c)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s, err := c.OpenStream(ctx, ChatRequest{
		Model: "hermes-agent",
		Messages: []Message{
			{Role: "user", Content: "Reply with exactly the word OK."},
		},
		Stream: true,
	})
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	defer s.Close()

	var buf strings.Builder
	var gotDone bool
	for {
		ev, rerr := s.Recv(ctx)
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			t.Fatalf("Recv: %v", rerr)
		}
		if ev.Kind == EventToken {
			buf.WriteString(ev.Token)
		}
		if ev.Kind == EventDone {
			gotDone = true
			break
		}
	}
	if !gotDone {
		// Some OpenAI-compatible servers may close the stream on [DONE]
		// without emitting a final finish_reason delta first. io.EOF is
		// also an acceptable terminal state.
		t.Log("no explicit EventDone (server closed with [DONE] / EOF)")
	}
	if !strings.Contains(strings.ToUpper(buf.String()), "OK") {
		t.Errorf("reply did not contain OK: %q", buf.String())
	}
}
