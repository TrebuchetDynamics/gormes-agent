// Package tools is the compatibility facade for Go-native Gormes tools.
package tools

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
)

type Tool = toolkit.Tool
type ToolDescriptor = toolkit.ToolDescriptor
type OperationSpec = toolkit.OperationSpec
type Spec = toolkit.Spec
type Registry = toolkit.Registry
type ToolExecutor = toolkit.ToolExecutor
type ToolRequest = toolkit.ToolRequest
type ToolEvent = toolkit.ToolEvent
type InProcessToolExecutor = toolkit.InProcessToolExecutor

var ErrDuplicate = toolkit.ErrDuplicate
var ErrUnknownTool = toolkit.ErrUnknownTool

func DefaultSpec(name, desc string, schema json.RawMessage) OperationSpec {
	return toolkit.DefaultSpec(name, desc, schema)
}

func NewRegistry() *Registry {
	return toolkit.NewRegistry()
}

func NewInProcessToolExecutor(reg *Registry) *InProcessToolExecutor {
	return toolkit.NewInProcessToolExecutor(reg)
}
