package kernel

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/internal/audit"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/telemetry"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// TestExecuteToolCallsConcurrent_InterruptPropagatesToEveryWorker freezes the
// Phase 2.E.2 cancel contract: when the parent context is cancelled while
// multiple tool workers are in-flight, every worker observes ctx cancellation
// promptly and the returned slice carries a single coherent cancel envelope
// per cancelled call.
func TestExecuteToolCallsConcurrent_InterruptPropagatesToEveryWorker(t *testing.T) {
	const workerCount = 4
	var (
		inFlight sync.WaitGroup
		ctxSeen  atomic.Int64
	)
	inFlight.Add(workerCount)

	reg := tools.NewRegistry()
	for i := 0; i < workerCount; i++ {
		name := "slow_" + string(rune('A'+i))
		reg.MustRegister(&tools.MockTool{
			NameStr:  name,
			TimeoutD: 5 * time.Second,
			ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
				inFlight.Done()
				select {
				case <-ctx.Done():
					ctxSeen.Add(1)
					return nil, ctx.Err()
				case <-time.After(3 * time.Second):
					return json.RawMessage(`{"ok":true}`), nil
				}
			},
		})
	}

	auditPath := filepath.Join(t.TempDir(), "concurrent-audit.jsonl")
	k := New(Config{
		Model:             "hermes-agent",
		Endpoint:          "http://mock",
		Admission:         Admission{MaxBytes: 200_000, MaxLines: 10_000},
		Tools:             reg,
		MaxToolIterations: 10,
		MaxToolDuration:   5 * time.Second,
		InitialSessionID:  "sess_concurrent",
		ToolAudit:         audit.NewJSONLWriter(auditPath),
	}, hermes.NewMockClient(), store.NewNoop(), telemetry.New(), nil)

	calls := make([]hermes.ToolCall, workerCount)
	for i := 0; i < workerCount; i++ {
		calls[i] = hermes.ToolCall{
			ID:        "call_" + string(rune('A'+i)),
			Name:      "slow_" + string(rune('A'+i)),
			Arguments: json.RawMessage(`{}`),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	resultsCh := make(chan []toolResult, 1)
	go func() {
		resultsCh <- k.executeToolCallsConcurrent(ctx, calls, workerCount)
	}()

	// Wait until every worker has entered Execute, then cancel.
	waitDone := make(chan struct{})
	go func() {
		inFlight.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %d concurrent workers to enter Execute", workerCount)
	}

	start := time.Now()
	cancel()

	var res []toolResult
	select {
	case res = <-resultsCh:
	case <-time.After(1 * time.Second):
		t.Fatalf("executeToolCallsConcurrent did not return within 1s of cancel; workers still blocked")
	}
	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Fatalf("cancel propagation too slow: %v (want <500ms)", elapsed)
	}

	if len(res) != workerCount {
		t.Fatalf("len(res)=%d, want %d", len(res), workerCount)
	}

	// Every result must be the coherent cancel envelope and preserve call order.
	for i, r := range res {
		if r.ID != calls[i].ID {
			t.Errorf("res[%d].ID=%q, want %q (order preserved)", i, r.ID, calls[i].ID)
		}
		if r.Name != calls[i].Name {
			t.Errorf("res[%d].Name=%q, want %q", i, r.Name, calls[i].Name)
		}
		var env struct {
			Error  string `json:"error"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(r.Content), &env); err != nil {
			t.Errorf("res[%d] content=%q not JSON: %v", i, r.Content, err)
			continue
		}
		if env.Error != "cancelled" {
			t.Errorf("res[%d].error=%q, want %q", i, env.Error, "cancelled")
		}
		if env.Reason != "parent_cancelled" {
			t.Errorf("res[%d].reason=%q, want %q", i, env.Reason, "parent_cancelled")
		}
	}

	if got := ctxSeen.Load(); got != int64(workerCount) {
		t.Errorf("only %d/%d workers observed ctx.Done", got, workerCount)
	}

	// Audit records: one "cancelled" entry per call.
	records := readAllAuditRecords(t, auditPath)
	if len(records) != workerCount {
		t.Fatalf("audit record count=%d, want %d", len(records), workerCount)
	}
	for i, rec := range records {
		if rec.Status != "cancelled" {
			t.Errorf("audit[%d].status=%q, want %q", i, rec.Status, "cancelled")
		}
		if rec.Source != "kernel" {
			t.Errorf("audit[%d].source=%q, want %q", i, rec.Source, "kernel")
		}
	}
}

// TestExecuteToolCallsConcurrent_PreservesOrderOnSuccess proves the ordered
// result contract for the common (non-interrupted) path.
func TestExecuteToolCallsConcurrent_PreservesOrderOnSuccess(t *testing.T) {
	reg := tools.NewRegistry()
	var seq atomic.Int64
	reg.MustRegister(&tools.MockTool{
		NameStr: "one",
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			n := seq.Add(1)
			return json.RawMessage(`{"order":` + jsonNum(n) + `,"name":"one"}`), nil
		},
	})
	reg.MustRegister(&tools.MockTool{
		NameStr: "two",
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			n := seq.Add(1)
			return json.RawMessage(`{"order":` + jsonNum(n) + `,"name":"two"}`), nil
		},
	})
	reg.MustRegister(&tools.MockTool{
		NameStr: "three",
		ExecuteFn: func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
			n := seq.Add(1)
			return json.RawMessage(`{"order":` + jsonNum(n) + `,"name":"three"}`), nil
		},
	})

	k := newKernelWithRegistry(t, reg)

	calls := []hermes.ToolCall{
		{ID: "c1", Name: "one", Arguments: json.RawMessage(`{}`)},
		{ID: "c2", Name: "two", Arguments: json.RawMessage(`{}`)},
		{ID: "c3", Name: "three", Arguments: json.RawMessage(`{}`)},
	}
	res := k.executeToolCallsConcurrent(context.Background(), calls, 3)

	if len(res) != 3 {
		t.Fatalf("len(res)=%d, want 3", len(res))
	}
	for i, c := range calls {
		if res[i].ID != c.ID || res[i].Name != c.Name {
			t.Errorf("res[%d]=%v, want id=%s name=%s", i, res[i], c.ID, c.Name)
		}
		if !strings.Contains(res[i].Content, `"name":"`+c.Name+`"`) {
			t.Errorf("res[%d].Content=%q, want name=%s payload", i, res[i].Content, c.Name)
		}
	}
}

// TestExecuteToolCallsConcurrent_QueuedWorkersCancelledBeforeStart proves that
// when the parent context is already cancelled, queued-but-not-started workers
// still return a coherent cancel envelope (no silent swallowing).
func TestExecuteToolCallsConcurrent_QueuedWorkersCancelledBeforeStart(t *testing.T) {
	reg := tools.NewRegistry()
	reg.MustRegister(&tools.MockTool{NameStr: "a"})
	reg.MustRegister(&tools.MockTool{NameStr: "b"})

	k := newKernelWithRegistry(t, reg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := k.executeToolCallsConcurrent(ctx, []hermes.ToolCall{
		{ID: "1", Name: "a", Arguments: json.RawMessage(`{}`)},
		{ID: "2", Name: "b", Arguments: json.RawMessage(`{}`)},
	}, 2)

	if len(res) != 2 {
		t.Fatalf("len(res)=%d, want 2", len(res))
	}
	for i, r := range res {
		var env struct {
			Error  string `json:"error"`
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal([]byte(r.Content), &env); err != nil {
			t.Fatalf("res[%d] content=%q not JSON: %v", i, r.Content, err)
		}
		if env.Error != "cancelled" || env.Reason != "parent_cancelled" {
			t.Errorf("res[%d] envelope=%+v, want error=cancelled reason=parent_cancelled", i, env)
		}
	}
}

func jsonNum(n int64) string {
	// Minimal int-to-string helper to keep the test dep-free.
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func readAllAuditRecords(t *testing.T, path string) []audit.Record {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	out := make([]audit.Record, 0, len(lines))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var rec audit.Record
		if err := json.Unmarshal(line, &rec); err != nil {
			t.Fatalf("Unmarshal(%s): %v", line, err)
		}
		out = append(out, rec)
	}
	return out
}
