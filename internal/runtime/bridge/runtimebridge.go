// Package runtimebridge reserves a Python-neutral seam for future external
// runtimes. The current Gormes runtime does not use this package.
package runtimebridge

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("gormes/runtimebridge: runtime is not implemented")

type Runtime interface {
	ID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Health(ctx context.Context) error
	Catalog(ctx context.Context) (ToolCatalog, error)
	Invoke(ctx context.Context, req InvocationRequest) (Invocation, error)
}

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
	Kind    string
	Payload json.RawMessage
}

type InvocationResult struct {
	Payload  json.RawMessage
	Stderr   string
	ExitCode int
	Duration time.Duration
}

var _ Runtime = (*NoRuntime)(nil)

type NoRuntime struct{}

func (*NoRuntime) ID() string                   { return "noop" }
func (*NoRuntime) Start(context.Context) error  { return ErrNotImplemented }
func (*NoRuntime) Stop(context.Context) error   { return ErrNotImplemented }
func (*NoRuntime) Health(context.Context) error { return ErrNotImplemented }
func (*NoRuntime) Catalog(context.Context) (ToolCatalog, error) {
	return ToolCatalog{}, ErrNotImplemented
}
func (*NoRuntime) Invoke(context.Context, InvocationRequest) (Invocation, error) {
	return nil, ErrNotImplemented
}
