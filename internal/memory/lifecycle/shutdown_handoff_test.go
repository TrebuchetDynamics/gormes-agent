package lifecycle

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type recordedHandoff struct {
	called   bool
	messages []ShutdownMessage
	err      error
}

func (r *recordedHandoff) Shutdown(_ context.Context, messages []ShutdownMessage) error {
	r.called = true
	if messages == nil {
		r.messages = []ShutdownMessage{}
	} else {
		r.messages = append(r.messages[:0:0], messages...)
	}
	return r.err
}

func TestShutdownMemoryHandoff_CompletedPassesMessages(t *testing.T) {
	provider := &recordedHandoff{}
	input := ShutdownHandoffInput{
		Messages: []ShutdownMessage{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi back"},
		},
	}

	status, err := PerformShutdownHandoff(context.Background(), provider, input)
	if err != nil {
		t.Fatalf("PerformShutdownHandoff returned err = %v", err)
	}
	if !provider.called {
		t.Fatalf("provider.Shutdown was not called")
	}
	if !reflect.DeepEqual(provider.messages, input.Messages) {
		t.Fatalf("provider.messages = %+v, want %+v", provider.messages, input.Messages)
	}
	if status.Code != ShutdownMemoryInvoked {
		t.Fatalf("status.Code = %q, want %q", status.Code, ShutdownMemoryInvoked)
	}
	if !status.Provided {
		t.Fatalf("status.Provided = false, want true after explicit invocation")
	}
}

func TestShutdownMemoryHandoff_EmptyCompletedStillCallsProvider(t *testing.T) {
	provider := &recordedHandoff{}

	status, err := PerformShutdownHandoff(context.Background(), provider, ShutdownHandoffInput{})
	if err != nil {
		t.Fatalf("PerformShutdownHandoff returned err = %v", err)
	}
	if !provider.called {
		t.Fatalf("provider.Shutdown was not called for empty completed transcript; must not fall back to no-arg behavior")
	}
	if len(provider.messages) != 0 {
		t.Fatalf("provider.messages = %+v, want empty slice", provider.messages)
	}
	if status.Code != ShutdownMemoryInvoked {
		t.Fatalf("status.Code = %q, want %q", status.Code, ShutdownMemoryInvoked)
	}
}

func TestShutdownMemoryHandoff_SkipMemorySuppressesProvider(t *testing.T) {
	provider := &recordedHandoff{}

	status, err := PerformShutdownHandoff(context.Background(), provider, ShutdownHandoffInput{
		Messages:   []ShutdownMessage{{Role: "user", Content: "noop"}},
		SkipMemory: true,
	})
	if err != nil {
		t.Fatalf("PerformShutdownHandoff returned err = %v", err)
	}
	if provider.called {
		t.Fatalf("provider.Shutdown was called despite skip_memory")
	}
	if status.Code != ShutdownMemorySkipped {
		t.Fatalf("status.Code = %q, want %q", status.Code, ShutdownMemorySkipped)
	}
	if status.Provided {
		t.Fatalf("status.Provided = true, want false when skipped")
	}
}

func TestShutdownMemoryHandoff_InterruptedSuppressesProvider(t *testing.T) {
	provider := &recordedHandoff{}

	status, err := PerformShutdownHandoff(context.Background(), provider, ShutdownHandoffInput{
		Messages:    []ShutdownMessage{{Role: "user", Content: "partial"}},
		Interrupted: true,
	})
	if err != nil {
		t.Fatalf("PerformShutdownHandoff returned err = %v", err)
	}
	if provider.called {
		t.Fatalf("provider.Shutdown was called for interrupted session; honcho-compatible memory hooks must not fire on interrupted turns")
	}
	if status.Code != ShutdownMemoryInterrupted {
		t.Fatalf("status.Code = %q, want %q", status.Code, ShutdownMemoryInterrupted)
	}
}

func TestShutdownMemoryHandoff_PropagatesProviderError(t *testing.T) {
	wantErr := errors.New("memory store offline")
	provider := &recordedHandoff{err: wantErr}

	status, err := PerformShutdownHandoff(context.Background(), provider, ShutdownHandoffInput{
		Messages: []ShutdownMessage{{Role: "user", Content: "boom"}},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if status.Code != ShutdownMemoryInvoked {
		t.Fatalf("status.Code = %q, want %q even when provider returns error", status.Code, ShutdownMemoryInvoked)
	}
}

func TestShutdownMemoryHandoff_NilProviderIsSkipped(t *testing.T) {
	status, err := PerformShutdownHandoff(context.Background(), nil, ShutdownHandoffInput{})
	if err != nil {
		t.Fatalf("PerformShutdownHandoff with nil provider returned err = %v", err)
	}
	if status.Code != ShutdownMemorySkipped {
		t.Fatalf("status.Code = %q, want %q when provider is nil", status.Code, ShutdownMemorySkipped)
	}
	if status.Provided {
		t.Fatalf("status.Provided = true, want false when provider is nil")
	}
}
