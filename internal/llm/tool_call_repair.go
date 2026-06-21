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

// sanitizeAnthropicToolDescriptors sanitizes descriptors for the native
// Anthropic API, additionally stripping top-level oneOf/allOf/anyOf that the
// Anthropic validator rejects. Mirrors Hermes fix(anthropic) a219a0a4d.
func sanitizeAnthropicToolDescriptors(descriptors []ToolDescriptor) []ToolDescriptor {
	if len(descriptors) == 0 {
		return nil
	}
	out := make([]ToolDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		out = append(out, ToolDescriptor{
			Name:         d.Name,
			Description:  d.Description,
			Schema:       sanitizeAnthropicToolSchema(d.Schema),
			CacheControl: d.CacheControl,
		})
	}
	return out
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
			Schema:      cloneRawMessage(d.Schema),
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
			Schema:      cloneRawMessage(d.Schema),
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
			Arguments: cloneRawMessage(c.Arguments),
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
			Arguments: cloneRawMessage(c.Arguments),
		})
	}
	return out
}

func sanitizeToolSchema(raw json.RawMessage) json.RawMessage {
	return repair.SanitizeToolSchema(raw)
}

// sanitizeAnthropicToolSchema sanitizes a tool schema for the native Anthropic
// API: runs the standard sanitizer then strips top-level oneOf/allOf/anyOf which
// the Anthropic validator rejects with HTTP 400.
// Mirrors Hermes fix(anthropic): strip top-level oneOf/allOf/anyOf from tool
// input_schema (a219a0a4d).
func sanitizeAnthropicToolSchema(raw json.RawMessage) json.RawMessage {
	out := repair.SanitizeToolSchema(raw)
	return stripAnthropicForbiddenTopLevel(out)
}

// stripAnthropicForbiddenTopLevel removes oneOf/allOf/anyOf from the root of
// a tool parameter schema. These union keywords cause HTTP 400 from the native
// Anthropic API. Nested occurrences inside properties are preserved.
func stripAnthropicForbiddenTopLevel(raw json.RawMessage) json.RawMessage {
	var node map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &node) != nil {
		return raw
	}
	modified := false
	for _, key := range []string{"oneOf", "allOf", "anyOf"} {
		if _, ok := node[key]; ok {
			delete(node, key)
			modified = true
		}
	}
	if !modified {
		return raw
	}
	if _, ok := node["type"]; !ok {
		node["type"] = "object"
	}
	out, err := json.Marshal(node)
	if err != nil {
		return raw
	}
	return out
}

func repairToolCallArguments(raw json.RawMessage) (json.RawMessage, map[string]any, error) {
	return repair.RepairToolCallArguments(raw)
}
