package llm

import "encoding/json"

var emptyObjectToolSchema = json.RawMessage(`{"type":"object","properties":{}}`)

func cloneRawMessage(raw json.RawMessage) json.RawMessage {
	return append(json.RawMessage(nil), raw...)
}

func cloneRawMessageOrDefault(raw, fallback json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return cloneRawMessage(fallback)
	}
	return cloneRawMessage(raw)
}

func cloneToolSchemaOrDefault(raw json.RawMessage) json.RawMessage {
	return cloneRawMessageOrDefault(raw, emptyObjectToolSchema)
}

func cloneMessageContentParts(in []MessageContentPart) []MessageContentPart {
	if in == nil {
		return nil
	}
	return append([]MessageContentPart(nil), in...)
}

func cloneMessages(in []Message) []Message {
	if in == nil {
		return nil
	}
	out := make([]Message, len(in))
	for i := range in {
		out[i] = cloneMessage(in[i])
	}
	return out
}

func cloneMessage(msg Message) Message {
	out := msg
	out.ContentParts = cloneMessageContentParts(msg.ContentParts)
	out.CacheControl = cloneCacheControl(msg.CacheControl)
	if msg.Reasoning != nil {
		reasoning := *msg.Reasoning
		out.Reasoning = &reasoning
	}
	if msg.ReasoningContent != nil {
		reasoningContent := *msg.ReasoningContent
		out.ReasoningContent = &reasoningContent
	}
	out.ToolCalls = cloneToolCalls(msg.ToolCalls)
	return out
}

func cloneToolCalls(in []ToolCall) []ToolCall {
	if in == nil {
		return nil
	}
	out := make([]ToolCall, len(in))
	for i, call := range in {
		out[i] = ToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: cloneRawMessage(call.Arguments),
		}
	}
	return out
}

func cloneToolDescriptors(in []ToolDescriptor) []ToolDescriptor {
	if in == nil {
		return nil
	}
	out := make([]ToolDescriptor, len(in))
	for i, descriptor := range in {
		out[i] = ToolDescriptor{
			Name:        descriptor.Name,
			Description: descriptor.Description,
			Schema:      cloneRawMessage(descriptor.Schema),
		}
	}
	return out
}

func cloneCacheControl(in *CacheControl) *CacheControl {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
