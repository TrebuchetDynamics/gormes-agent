package kernel

import (
	"testing"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/persistence/store"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/telemetry"
)

// Subscribe must give each consumer an independent render stream. Cron runs and
// the gateway share one kernel; with the old single shared channel they
// competed as receivers and stole each other's frames. Each subscription must
// receive every emitted frame.
func TestKernelSubscribeDeliversToAllSubscribersIndependently(t *testing.T) {
	k := New(Config{Model: "m", Endpoint: "http://mock"}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)

	chA, unsubA := k.Subscribe()
	defer unsubA()
	chB, unsubB := k.Subscribe()
	defer unsubB()

	k.emitFrame("hello")

	for _, tc := range []struct {
		name string
		ch   <-chan RenderFrame
	}{{"A", chA}, {"B", chB}} {
		select {
		case f := <-tc.ch:
			if f.Seq == 0 {
				t.Fatalf("subscriber %s got an unpopulated frame", tc.name)
			}
		default:
			t.Fatalf("subscriber %s received no frame (frame stolen by another consumer)", tc.name)
		}
	}
}

// After unsubscribe, the subscription channel is closed and no longer competes
// for frames.
func TestKernelUnsubscribeClosesChannel(t *testing.T) {
	k := New(Config{Model: "m", Endpoint: "http://mock"}, llm.NewMockClient(), store.NewNoop(), telemetry.New(), nil)

	ch, unsub := k.Subscribe()
	unsub()

	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after unsubscribe")
	}

	// A frame emitted after unsubscribe must not panic (no send on closed chan).
	k.emitFrame("after-unsub")
}
