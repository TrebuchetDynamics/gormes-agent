// gormes/internal/subagent/subagent_test.go
package subagent

import (
	"context"
	"testing"
	"time"
)

func TestSubagentEventsReadOnly(t *testing.T) {
	sa := newTestSubagent()
	// Compile-time guarantee: Events() returns a receive-only channel. We
	// can't write to it, but we can confirm the runtime type assertion path.
	var _ <-chan SubagentEvent = sa.Events()
}

func TestSubagentWaitForResultBlocksUntilDone(t *testing.T) {
	sa := newTestSubagent()
	got := make(chan *SubagentResult, 1)

	go func() {
		r, err := sa.WaitForResult(context.Background())
		if err != nil {
			t.Errorf("WaitForResult error: %v", err)
		}
		got <- r
	}()

	select {
	case <-got:
		t.Fatal("WaitForResult returned before done was closed")
	case <-time.After(50 * time.Millisecond):
	}

	want := &SubagentResult{ID: "sa_test", Status: StatusCompleted}
	sa.setResult(want)
	close(sa.done)

	select {
	case r := <-got:
		if r != want {
			t.Errorf("WaitForResult: want %+v, got %+v", want, r)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForResult did not return after done was closed")
	}
}

func TestSubagentWaitForResultRespectsCallerCtx(t *testing.T) {
	sa := newTestSubagent()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r, err := sa.WaitForResult(ctx)
	if err != context.Canceled {
		t.Errorf("err: want context.Canceled, got %v", err)
	}
	if r != nil {
		t.Errorf("result: want nil, got %+v", r)
	}
}

func newTestSubagent() *Subagent {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subagent{
		ID:           "sa_test",
		Depth:        1,
		ctx:          ctx,
		cancel:       cancel,
		publicEvents: make(chan SubagentEvent),
		done:         make(chan struct{}),
	}
}
