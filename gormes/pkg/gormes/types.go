// Package gormes re-exports the stable Phase-1 public surface for external
// consumers. Every actual definition lives in an internal/ package; this file
// is purely type aliases so "import .../gormes/pkg/gormes" works as a single
// stable entry point across refactors.
package gormes

import (
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/kernel"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/plugins"
	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/pybridge"
)

// Hermes wire surface — everything Gormes needs to speak HTTP+SSE to a
// Hermes-compatible api_server.
type (
	Client          = hermes.Client
	Stream          = hermes.Stream
	RunEventStream  = hermes.RunEventStream
	ChatRequest     = hermes.ChatRequest
	Message         = hermes.Message
	ContentPart     = hermes.ContentPart
	ContentPartType = hermes.ContentPartType
	Event           = hermes.Event
	EventKind       = hermes.EventKind
	RunEvent        = hermes.RunEvent
	RunEventType    = hermes.RunEventType
	ErrorClass      = hermes.ErrorClass
	HTTPError       = hermes.HTTPError
)

const (
	ContentPartText  = hermes.ContentPartText
	ContentPartImage = hermes.ContentPartImage
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

// Runtime seam — Phase-5 interface definitions, present in Phase 1 so
// downstream integrators can write conforming runtimes ahead of time.
type (
	Runtime    = pybridge.Runtime
	Invocation = pybridge.Invocation
)

// Plugin SDK surface — external plugin tooling can import the stable manifest
// contract without depending on internal packages directly.
type (
	PluginKind           = plugins.Kind
	PluginManifest       = plugins.Manifest
	PluginEnvRequirement = plugins.EnvRequirement
)

const (
	PluginKindGeneral        = plugins.KindGeneral
	PluginKindMemoryProvider = plugins.KindMemoryProvider
	PluginKindContextEngine  = plugins.KindContextEngine
)

func LoadPluginManifest(path string) (PluginManifest, error) {
	return plugins.LoadManifest(path)
}

func DiscoverPluginManifests(root string, kind PluginKind) ([]PluginManifest, error) {
	return plugins.Discover(root, kind)
}
