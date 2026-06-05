package llm

import (
	"encoding/json"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/repair"
)

// ToolCallRepairError reports a provider-emitted tool call that could not be
// safely reconciled with the tool schemas advertised on the current request.
type ToolCallRepairError = repair.ToolCallRepairError

// SanitizeToolDescriptors returns a deep copy of descriptors with schemas
// normalized to the conservative object-shaped subset accepted by provider
// tool parsers.
func SanitizeToolDescriptors(descriptors []ToolDescriptor) []ToolDescriptor {
	repaired := repair.SanitizeToolDescriptors(toRepairToolDescriptors(descriptors))
	return fromRepairToolDescriptors(repaired)
}

// RepairToolCalls repairs deterministic JSON malformations and validates the
// final arguments against the currently advertised tool descriptors.
func RepairToolCalls(calls []ToolCall, descriptors []ToolDescriptor) ([]ToolCall, error) {
	repaired, err := repair.RepairToolCalls(toRepairToolCalls(calls), toRepairToolDescriptors(descriptors))
	if err != nil {
		return nil, err
	}
	return fromRepairToolCalls(repaired), nil
}

func toRepairToolDescriptors(descriptors []ToolDescriptor) []repair.ToolDescriptor {
	if len(descriptors) == 0 {
		return nil
	}
	out := make([]repair.ToolDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, repair.ToolDescriptor{
			Name:        d.Name,
			Description: d.Description,
			Schema:      append(json.RawMessage(nil), d.Schema...),
		})
	}
	return out
}

func fromRepairToolDescriptors(descriptors []repair.ToolDescriptor) []ToolDescriptor {
	if len(descriptors) == 0 {
		return nil
	}
	out := make([]ToolDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, ToolDescriptor{
			Name:        d.Name,
			Description: d.Description,
			Schema:      append(json.RawMessage(nil), d.Schema...),
		})
	}
	return out
}

func toRepairToolCalls(calls []ToolCall) []repair.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]repair.ToolCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, repair.ToolCall{
			ID:        c.ID,
			Name:      c.Name,
			Arguments: append(json.RawMessage(nil), c.Arguments...),
		})
	}
	return out
}

func fromRepairToolCalls(calls []repair.ToolCall) []ToolCall {
	if len(calls) == 0 {
		return nil
	}
	out := make([]ToolCall, 0, len(calls))
	for _, c := range calls {
		out = append(out, ToolCall{
			ID:        c.ID,
			Name:      c.Name,
			Arguments: append(json.RawMessage(nil), c.Arguments...),
		})
	}
	return out
}

func sanitizeToolSchema(raw json.RawMessage) json.RawMessage {
	return repair.SanitizeToolSchema(raw)
}

func repairToolCallArguments(raw json.RawMessage) (json.RawMessage, map[string]any, error) {
	return repair.RepairToolCallArguments(raw)
}
