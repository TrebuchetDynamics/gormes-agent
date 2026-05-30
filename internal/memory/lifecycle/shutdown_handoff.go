package lifecycle

import "context"

// ShutdownMessage is a small role/content message used to hand a session's
// completed transcript to the shutdown memory provider. Keeping the type
// local to internal/memory keeps this helper a pure boundary contract; the
// gateway/CLI layers convert their richer message structs into this slice.
type ShutdownMessage struct {
	Role    string
	Content string
}

// ShutdownMemoryProvider receives the session's transcript when a session
// exits. Implementations persist the messages — the helper invokes
// Shutdown exactly once with the explicit slice (possibly empty) so
// providers cannot fall back to no-argument behavior that loses transcript
// context, mirroring upstream Hermes 500774e3/a59a98b1.
type ShutdownMemoryProvider interface {
	Shutdown(ctx context.Context, messages []ShutdownMessage) error
}

// ShutdownMemoryStatusCode classifies the handoff outcome for telemetry.
type ShutdownMemoryStatusCode string

const (
	ShutdownMemoryInvoked     ShutdownMemoryStatusCode = "invoked"
	ShutdownMemorySkipped     ShutdownMemoryStatusCode = "shutdown_memory_skipped"
	ShutdownMemoryInterrupted ShutdownMemoryStatusCode = "shutdown_memory_interrupted"
)

// ShutdownMemoryStatus carries the structured evidence about whether the
// provider ran and why.
type ShutdownMemoryStatus struct {
	Code     ShutdownMemoryStatusCode
	Reason   string
	Provided bool
}

// ShutdownHandoffInput captures the per-session signals the helper needs:
// the completed transcript and the suppression flags that gateway/CLI
// runtimes already track.
type ShutdownHandoffInput struct {
	Messages    []ShutdownMessage
	SkipMemory  bool
	Interrupted bool
}

// PerformShutdownHandoff routes a session-end transcript to the provider
// exactly once. SkipMemory short-circuits with shutdown_memory_skipped,
// Interrupted short-circuits with shutdown_memory_interrupted, and a nil
// provider is treated as skip. Any provider error is returned verbatim
// while the status still records that the call was attempted.
func PerformShutdownHandoff(ctx context.Context, provider ShutdownMemoryProvider, input ShutdownHandoffInput) (ShutdownMemoryStatus, error) {
	if provider == nil {
		return ShutdownMemoryStatus{Code: ShutdownMemorySkipped, Reason: "no shutdown memory provider configured"}, nil
	}
	if input.SkipMemory {
		return ShutdownMemoryStatus{Code: ShutdownMemorySkipped, Reason: "skip_memory set on session"}, nil
	}
	if input.Interrupted {
		return ShutdownMemoryStatus{Code: ShutdownMemoryInterrupted, Reason: "session was interrupted before completion"}, nil
	}

	messages := input.Messages
	if messages == nil {
		messages = []ShutdownMessage{}
	}
	err := provider.Shutdown(ctx, messages)
	return ShutdownMemoryStatus{Code: ShutdownMemoryInvoked, Provided: true}, err
}
