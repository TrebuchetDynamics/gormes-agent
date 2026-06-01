package repair

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair/toolcalls"
)

type ToolDescriptor = toolcalls.ToolDescriptor

type ToolCall = toolcalls.ToolCall

// ToolCallRepairError reports a provider-emitted tool call that could not be
// safely reconciled with the tool schemas advertised on the current request.
type ToolCallRepairError = toolcalls.ToolCallRepairError

// SanitizeToolDescriptors returns a deep copy of descriptors with schemas
// normalized to the conservative object-shaped subset accepted by provider
// tool parsers.
func SanitizeToolDescriptors(descriptors []ToolDescriptor) []ToolDescriptor {
	return toolcalls.SanitizeToolDescriptors(descriptors)
}

// RepairToolCalls repairs deterministic JSON malformations and validates the
// final arguments against the currently advertised tool descriptors.
func RepairToolCalls(calls []ToolCall, descriptors []ToolDescriptor) ([]ToolCall, error) {
	return toolcalls.RepairToolCalls(calls, descriptors)
}

func SanitizeToolSchema(raw json.RawMessage) json.RawMessage {
	return toolcalls.SanitizeToolSchema(raw)
}

func RepairToolCallArguments(raw json.RawMessage) (json.RawMessage, map[string]any, error) {
	return toolcalls.RepairToolCallArguments(raw)
}
