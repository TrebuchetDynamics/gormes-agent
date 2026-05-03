// Package gormes re-exports the stable Phase-1 public surface for external
// consumers. Every actual definition lives in an internal/ package; this file
// is purely type aliases so "import .../gormes/pkg/gormes" works as a single
// stable entry point across refactors.
package gormes

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/runtimebridge"
)

// Hermes wire surface — everything Gormes needs to speak HTTP+SSE to a
// Hermes-compatible api_server.
type (
	Client         = hermes.Client
	Stream         = hermes.Stream
	RunEventStream = hermes.RunEventStream
	ChatRequest    = hermes.ChatRequest
	Message        = hermes.Message
	Event          = hermes.Event
	EventKind      = hermes.EventKind
	RunEvent       = hermes.RunEvent
	RunEventType   = hermes.RunEventType
	ErrorClass     = hermes.ErrorClass
	HTTPError      = hermes.HTTPError
)

// Kernel surface — the RenderFrame the TUI consumes plus the PlatformEvent
// it emits. External TUIs (future Bubble Tea alternatives, web UIs, etc.)
// can re-implement a UI by importing only these.
type (
	RenderFrame       = kernel.RenderFrame
	Phase             = kernel.Phase
	SoulEntry         = kernel.SoulEntry
	PlatformEvent     = kernel.PlatformEvent
	PlatformEventKind = kernel.PlatformEventKind
)

// Runtime seam — Python-neutral interface definitions for future external
// runtimes. The current Gormes runtime does not use this seam.
type (
	Runtime    = runtimebridge.Runtime
	Invocation = runtimebridge.Invocation
)
