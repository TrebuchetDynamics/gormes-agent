// Package pybridge reserves the Phase-5 runtime seam for Python subprocesses.
// Phase 1 ships only interface definitions — no concrete Runtime exists yet.
// All methods on a zero-value Runtime return ErrNotImplemented.
//
// The lifecycle-oriented shape (Start, Stop, Health, Catalog, Invoke) is
// intentionally richer than a simple Tool.Call(...) seam: Phase 5 will need
// warm-pool management, heartbeats, capability discovery, streamed output,
// and cancellation of long-running Python jobs. Locking that shape now
// prevents a breaking-change churn when Phase 5 lands.
package pybridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotImplemented is returned by NoRuntime until Phase 5 ships.
var ErrNotImplemented = errors.New("gormes/pybridge: runtime lands in Phase 5")

// Runtime is the lifecycle-oriented seam for Python subprocesses.
type Runtime interface {
	ID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
	Catalog(ctx context.Context) (ToolCatalog, error)
	Invoke(ctx context.Context, req InvocationRequest) (Invocation, error)
}

// Invocation represents one in-flight tool call. Events() streams progress
// and partial payloads; Wait() blocks for the final result; Cancel() aborts.
type Invocation interface {
	Events() <-chan InvocationEvent
	Wait(ctx context.Context) (InvocationResult, error)
	Cancel() error
}

type ToolCatalog struct {
	Tools []ToolDescriptor
}

type ToolDescriptor struct {
	Name        string
	Description string
	Schema      json.RawMessage
}

type InvocationRequest struct {
	Tool     string
	Args     json.RawMessage
	Deadline time.Duration
	TraceID  string
}

type InvocationEvent struct {
	Kind    string // "log" | "progress" | "partial"
	Payload json.RawMessage
}

type InvocationResult struct {
	Payload  json.RawMessage
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Compile-time interface check: NoRuntime must satisfy Runtime. If a future
// refactor drops a method, the build fails loudly.
var _ Runtime = (*NoRuntime)(nil)

// NoRuntime is the zero-value compile-checkable implementation for Phase 1.
// It is NOT used by Phase-1 code; it exists to prove the interface is
// concretely implementable and to satisfy the compile-time check above.
type NoRuntime struct{}

func (*NoRuntime) ID() string                                       { return "noop" }
func (*NoRuntime) Start(context.Context) error                      { return ErrNotImplemented }
func (*NoRuntime) Stop(context.Context) error                       { return ErrNotImplemented }
func (*NoRuntime) Health(context.Context) error                     { return ErrNotImplemented }
func (*NoRuntime) Catalog(context.Context) (ToolCatalog, error)     { return ToolCatalog{}, ErrNotImplemented }
func (*NoRuntime) Invoke(context.Context, InvocationRequest) (Invocation, error) {
	return nil, ErrNotImplemented
}
