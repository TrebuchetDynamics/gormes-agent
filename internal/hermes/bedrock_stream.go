package hermes

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type bedrockPendingToolUse struct {
	id    string
	name  string
	input strings.Builder
}

type bedrockStreamDecodeState struct {
	toolUses     map[int]*bedrockPendingToolUse
	stopReason   string
	inputTokens  int
	outputTokens int
}

func decodeBedrockStreamEvents(frames []map[string]any, interrupt func(Event) bool) ([]Event, error) {
	state := bedrockStreamDecodeState{
		toolUses: make(map[int]*bedrockPendingToolUse),
	}
	var out []Event
	for _, frame := range frames {
		events, err := state.consume(frame)
		if err != nil {
			return nil, err
		}
		for _, ev := range events {
			out = append(out, ev)
			if interrupt != nil && interrupt(ev) {
				return out, nil
			}
		}
	}
	done, err := state.doneEvent()
	if err != nil {
		return nil, err
	}
	out = append(out, done)
	if interrupt != nil {
		interrupt(done)
	}
	return out, nil
}

func (s *bedrockStreamDecodeState) consume(frame map[string]any) ([]Event, error) {
	if start, ok := bedrockMap(frame["contentBlockStart"]); ok {
		s.consumeContentBlockStart(start)
		return nil, nil
	}
	if delta, ok := bedrockMap(frame["contentBlockDelta"]); ok {
		return s.consumeContentBlockDelta(delta), nil
	}
	if stop, ok := bedrockMap(frame["messageStop"]); ok {
		s.stopReason = bedrockString(stop["stopReason"])
		return nil, nil
	}
	if metadata, ok := bedrockMap(frame["metadata"]); ok {
		s.consumeMetadata(metadata)
		return nil, nil
	}
	return nil, nil
}

func (s *bedrockStreamDecodeState) consumeContentBlockStart(start map[string]any) {
	index := bedrockIndex(start)
	startPayload, _ := bedrockMap(start["start"])
	toolUse, ok := bedrockMap(startPayload["toolUse"])
	if !ok {
		return
	}
	call := s.toolUse(index)
	if id := bedrockString(toolUse["toolUseId"]); id != "" {
		call.id = id
	}
	if id := bedrockString(toolUse["toolUseID"]); id != "" {
		call.id = id
	}
	if name := bedrockString(toolUse["name"]); name != "" {
		call.name = name
	}
}

func (s *bedrockStreamDecodeState) consumeContentBlockDelta(block map[string]any) []Event {
	delta, ok := bedrockMap(block["delta"])
	if !ok {
		return nil
	}
	var events []Event
	if reasoning := bedrockReasoningDelta(delta); reasoning != "" {
		events = append(events, Event{Kind: EventReasoning, Reasoning: reasoning})
	}
	if text := bedrockString(delta["text"]); text != "" {
		events = append(events, Event{Kind: EventToken, Token: text})
	}
	if toolUse, ok := bedrockMap(delta["toolUse"]); ok {
		call := s.toolUse(bedrockIndex(block))
		if id := bedrockString(toolUse["toolUseId"]); id != "" {
			call.id = id
		}
		if id := bedrockString(toolUse["toolUseID"]); id != "" {
			call.id = id
		}
		if name := bedrockString(toolUse["name"]); name != "" {
			call.name = name
		}
		if input := bedrockString(toolUse["input"]); input != "" {
			call.input.WriteString(input)
		}
	}
	return events
}

func (s *bedrockStreamDecodeState) consumeMetadata(metadata map[string]any) {
	usage, ok := bedrockMap(metadata["usage"])
	if !ok {
		return
	}
	s.inputTokens = bedrockUsageInt(usage, "inputTokens", "input_tokens")
	s.outputTokens = bedrockUsageInt(usage, "outputTokens", "output_tokens")
}

func (s *bedrockStreamDecodeState) doneEvent() (Event, error) {
	toolCalls, err := flushBedrockToolUses(s.toolUses)
	if err != nil {
		return Event{}, err
	}
	finishReason := mapBedrockStopReason(s.stopReason)
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	return Event{
		Kind:         EventDone,
		FinishReason: finishReason,
		TokensIn:     s.inputTokens,
		TokensOut:    s.outputTokens,
		ToolCalls:    toolCalls,
	}, nil
}

func (s *bedrockStreamDecodeState) toolUse(index int) *bedrockPendingToolUse {
	call, ok := s.toolUses[index]
	if !ok {
		call = &bedrockPendingToolUse{}
		s.toolUses[index] = call
	}
	return call
}

func flushBedrockToolUses(toolUses map[int]*bedrockPendingToolUse) ([]ToolCall, error) {
	if len(toolUses) == 0 {
		return nil, nil
	}
	indexes := make([]int, 0, len(toolUses))
	for index := range toolUses {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	out := make([]ToolCall, 0, len(indexes))
	for _, index := range indexes {
		call := toolUses[index]
		args := strings.TrimSpace(call.input.String())
		if args == "" {
			args = "{}"
		}
		canonical, _, err := repairToolCallArguments(json.RawMessage(args))
		if err != nil {
			return nil, fmt.Errorf("bedrock tool call %q arguments are invalid JSON: %w", call.id, err)
		}
		out = append(out, ToolCall{
			ID:        call.id,
			Name:      call.name,
			Arguments: canonical,
		})
	}
	return out, nil
}

func bedrockReasoningDelta(delta map[string]any) string {
	reasoning, ok := bedrockMap(delta["reasoningContent"])
	if !ok {
		return ""
	}
	if text := bedrockString(reasoning["text"]); text != "" {
		return text
	}
	reasoningText, ok := bedrockMap(reasoning["reasoningText"])
	if !ok {
		return ""
	}
	return bedrockString(reasoningText["text"])
}

func mapBedrockStopReason(reason string) string {
	switch reason {
	case "", "end_turn", "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens", "model_context_window_exceeded":
		return "length"
	case "content_filtered", "guardrail_intervened":
		return "content_filter"
	default:
		return reason
	}
}

func bedrockIndex(payload map[string]any) int {
	index, ok := bedrockInt(payload["contentBlockIndex"])
	if ok {
		return index
	}
	index, _ = bedrockInt(payload["index"])
	return index
}

func bedrockUsageInt(payload map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := bedrockInt(payload[key])
		if ok {
			return value
		}
	}
	return 0
}

func bedrockInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int8:
		return int(v), true
	case int16:
		return int(v), true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case json.Number:
		i, err := strconv.Atoi(v.String())
		return i, err == nil
	default:
		return 0, false
	}
}

func bedrockString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.RawMessage:
		return string(v)
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func bedrockMap(value any) (map[string]any, bool) {
	m, ok := value.(map[string]any)
	return m, ok
}
