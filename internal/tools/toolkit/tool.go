// Package toolkit defines the Go-native tool surface that the Gormes kernel
// executes when the LLM emits tool_calls. Every Tool is a Go type compiled
// into the Gormes binary; the Registry is populated explicitly by main.go
// (init() is permitted for third-party packages but not used in core).
package toolkit

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit/core"
)

// Tool is the contract every Go-native tool satisfies. See spec §5.1.
type Tool = core.Tool

// ToolDescriptor is the serialisable form sent to the LLM in ChatRequest.Tools.
// JSON shape matches OpenAI's tool-definition wrapper.
type ToolDescriptor = core.ToolDescriptor

// OperationSpec declares the behavioral metadata for a tool. Every tool that
// implements Spec() returns its OperationSpec; tools without it report
// descriptor_missing in doctor and inherit safe defaults in the executor.
// This is the contract-first operation catalog from the Phase 5.A row.
type OperationSpec = core.OperationSpec

// Spec is an optional interface that tools implement to declare their
// OperationSpec. Tools that don't implement this get a safe default.
type Spec = core.Spec

// Registry holds a set of named Tools. Safe for concurrent use.
type Registry = core.Registry

// ErrDuplicate is returned by Register when a tool name is already taken.
var ErrDuplicate = core.ErrDuplicate

// ErrUnknownTool is returned when a caller asks for a name that's not registered.
var ErrUnknownTool = core.ErrUnknownTool

// DefaultSpec returns a conservative OperationSpec for tools that don't
// implement the Spec interface. Mutating=true, Idempotent=false, PromptSafe=true.
func DefaultSpec(name, desc string, schema json.RawMessage) OperationSpec {
	return core.DefaultSpec(name, desc, schema)
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return core.NewRegistry()
}
