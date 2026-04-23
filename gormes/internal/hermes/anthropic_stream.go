package hermes

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
)

type anthropicStream struct {
	body   io.ReadCloser
	sse    *sseReader
	closed bool
	mu     sync.Mutex

	pending     []Event
	pendingTool map[int]*pendingToolCall
	tokensIn    int
	tokensOut   int
}

type anthropicChunk struct {
	Type string `json:"type"`

	Index int `json:"index,omitempty"`

	Message struct {
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`

	ContentBlock struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"content_block,omitempty"`

	Delta struct {
		Type        string `json:"type,omitempty"`
		Text        string `json:"text,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
		StopReason  string `json:"stop_reason,omitempty"`
	} `json:"delta,omitempty"`

	Usage struct {
		InputTokens  int `json:"input_tokens,omitempty"`
		OutputTokens int `json:"output_tokens,omitempty"`
	} `json:"usage,omitempty"`

	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func newAnthropicStream(body io.ReadCloser) *anthropicStream {
	return &anthropicStream{
		body:        body,
		sse:         newSSEReader(body),
		pendingTool: make(map[int]*pendingToolCall),
	}
}

func (s *anthropicStream) SessionID() string { return "" }

func (s *anthropicStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.body.Close()
}

func (s *anthropicStream) Recv(ctx context.Context) (Event, error) {
	for {
		select {
		case <-ctx.Done():
			return Event{}, ctx.Err()
		default:
		}

		if len(s.pending) > 0 {
			ev := s.pending[0]
			s.pending = s.pending[1:]
			return ev, nil
		}

		frame, err := s.sse.Next(ctx)
		if err != nil {
			return Event{}, err
		}
		if strings.TrimSpace(frame.data) == "" {
			continue
		}

		var chunk anthropicChunk
		if err := json.Unmarshal([]byte(frame.data), &chunk); err != nil {
			continue
		}
		raw := json.RawMessage(frame.data)

		switch chunk.Type {
		case "message_start":
			s.tokensIn = chunk.Message.Usage.InputTokens
			if chunk.Message.Usage.OutputTokens > s.tokensOut {
				s.tokensOut = chunk.Message.Usage.OutputTokens
			}
		case "content_block_start":
			if chunk.ContentBlock.Type == "tool_use" {
				s.pendingTool[chunk.Index] = &pendingToolCall{id: chunk.ContentBlock.ID, name: chunk.ContentBlock.Name}
			}
		case "content_block_delta":
			switch chunk.Delta.Type {
			case "text_delta":
				return Event{Kind: EventToken, Token: chunk.Delta.Text, Raw: raw}, nil
			case "thinking_delta":
				return Event{Kind: EventReasoning, Reasoning: chunk.Delta.Thinking, Raw: raw}, nil
			case "input_json_delta":
				p, ok := s.pendingTool[chunk.Index]
				if !ok {
					p = &pendingToolCall{}
					s.pendingTool[chunk.Index] = p
				}
				p.arguments.WriteString(chunk.Delta.PartialJSON)
			}
		case "message_delta":
			if chunk.Usage.InputTokens > 0 {
				s.tokensIn = chunk.Usage.InputTokens
			}
			if chunk.Usage.OutputTokens > 0 {
				s.tokensOut = chunk.Usage.OutputTokens
			}
			if chunk.Delta.StopReason != "" {
				ev := Event{
					Kind:         EventDone,
					FinishReason: normalizeAnthropicStopReason(chunk.Delta.StopReason),
					TokensIn:     s.tokensIn,
					TokensOut:    s.tokensOut,
					Raw:          raw,
				}
				if ev.FinishReason == "tool_calls" && len(s.pendingTool) > 0 {
					ev.ToolCalls = flushPending(s.pendingTool)
					s.pendingTool = make(map[int]*pendingToolCall)
				}
				return ev, nil
			}
		case "message_stop", "content_block_stop", "ping":
			continue
		case "error":
			return Event{}, errors.New("anthropic stream error: " + chunk.Error.Type + ": " + chunk.Error.Message)
		default:
			continue
		}
	}
}

func normalizeAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "tool_use":
		return "tool_calls"
	case "max_tokens":
		return "length"
	default:
		return reason
	}
}
