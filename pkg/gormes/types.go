// Package gormes re-exports the stable Phase-1 public surface for external
// consumers. Every actual definition lives in an internal/ package; this file
// is purely type aliases so "import .../gormes/pkg/gormes" works as a single
// stable entry point across refactors.
package gormes

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/runtime/bridge"
)

// Hermes wire surface — everything Gormes needs to speak HTTP+SSE to a
// Hermes-compatible api_server.
type (
	Client         = llm.Client
	Stream         = llm.Stream
	RunEventStream = llm.RunEventStream
	ChatRequest    = llm.ChatRequest
	Message        = llm.Message
	Event          = llm.Event
	EventKind      = llm.EventKind
	RunEvent       = llm.RunEvent
	RunEventType   = llm.RunEventType
	ErrorClass     = llm.ErrorClass
	HTTPError      = llm.HTTPError
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
