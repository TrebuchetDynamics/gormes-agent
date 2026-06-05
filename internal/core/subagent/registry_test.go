package subagent

import (
	"context"
	"testing"
)

func newRegistryTestSubagent() *Subagent {
	return &Subagent{ID: newSubagentID()}
}

func TestRegistryRegisterListUnregister(t *testing.T) {
	r := NewRegistry()
	if got := len(r.List()); got != 0 {
		t.Errorf("empty registry List(): want 0, got %d", got)
	}

	sa := newRegistryTestSubagent()
	r.Register(sa)
	if got := len(r.List()); got != 1 {
		t.Fatalf("after Register: want 1, got %d", got)
	}

	r.Unregister(sa.ID)
	if got := len(r.List()); got != 0 {
		t.Errorf("after Unregister: want 0, got %d", got)
	}
}

func TestRegistryUnregisterMissingIsNoOp(t *testing.T) {
	r := NewRegistry()
	r.Unregister("not_present")
}

func TestRegistryInterruptAllCancelsContexts(t *testing.T) {
	r := NewRegistry()

	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	r.Register(&Subagent{ID: "a", ctx: ctx1, cancel: cancel1})
	r.Register(&Subagent{ID: "b", ctx: ctx2, cancel: cancel2})

	r.InterruptAll("shutdown")

	for name, ctx := range map[string]context.Context{"a": ctx1, "b": ctx2} {
		select {
		case <-ctx.Done():
		default:
			t.Errorf("subagent %q ctx not cancelled by InterruptAll", name)
		}
	}
}

func TestRegistryInterruptAllStoresMessageBeforeCancel(t *testing.T) {
	r := NewRegistry()

	var sa *Subagent
	gotMsg := make(chan string, 1)
	sa = &Subagent{
		ID: "msg-check",
		cancel: func() {
			msg, _ := sa.interruptMsg.Load().(string)
			gotMsg <- msg
		},
	}
	r.Register(sa)

	r.InterruptAll("shutdown")

	select {
	case got := <-gotMsg:
		if got != "shutdown" {
			t.Fatalf("interruptMsg at cancel time = %q, want %q", got, "shutdown")
		}
	default:
		t.Fatal("cancel was not invoked")
	}
}

func TestRegistryListIsSnapshot(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < 5; i++ {
		r.Register(&Subagent{ID: newSubagentID()})
	}
	snap := r.List()
	if len(snap) != 5 {
		t.Errorf("List length: want 5, got %d", len(snap))
	}

	snap[0] = nil
	again := r.List()
	for _, sa := range again {
		if sa == nil {
			t.Error("List returned shared underlying array (got nil after mutating prior snapshot)")
		}
	}
}
