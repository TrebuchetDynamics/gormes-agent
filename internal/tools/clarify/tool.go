package clarify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrClarifyInvalidArgs  = errors.New("clarify_invalid_args")
	ErrClarifyUnavailable  = errors.New("clarify_unavailable")
	ErrClarifyRouteMissing = errors.New("clarify_route_missing")
	ErrClarifyTimeout      = errors.New("clarify_timeout")
)

type ClarifyCallback interface {
	Clarify(ctx context.Context, req ClarifyRequest) (ClarifyResponse, error)
}

type ClarifyCallbackFunc func(context.Context, ClarifyRequest) (ClarifyResponse, error)

func (f ClarifyCallbackFunc) Clarify(ctx context.Context, req ClarifyRequest) (ClarifyResponse, error) {
	return f(ctx, req)
}

type ClarifyRequest struct {
	Question string
	Choices  []string
}

type ClarifyResponse struct {
	UserResponse string
}

type ClarifyTool struct {
	callback ClarifyCallback
}

func NewClarifyTool(callback ClarifyCallback) *ClarifyTool {
	return &ClarifyTool{callback: callback}
}

func (*ClarifyTool) Name() string { return "clarify" }

func (*ClarifyTool) Description() string {
	return "Ask the user a clarifying question with optional multiple-choice answers."
}

func (*ClarifyTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"question":{"type":"string","description":"Question to ask the user."},"choices":{"type":"array","description":"Optional multiple-choice answers; at most four are shown.","items":{"type":"string"}}},"required":["question"],"additionalProperties":false}`)
}

func (*ClarifyTool) Timeout() time.Duration { return 30 * time.Second }

func (t *ClarifyTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	req, truncated, err := parseClarifyRequest(args)
	if err != nil {
		payload := clarifyPayload("clarify_invalid_args", req, map[string]any{"error": err.Error()})
		return payload, fmt.Errorf("%w: %s", ErrClarifyInvalidArgs, err.Error())
	}
	extra := map[string]any{}
	if truncated {
		extra["truncated"] = true
	}
	if t == nil || t.callback == nil {
		extra["noninteractive"] = true
		extra["assumption"] = "no interactive clarify callback is available; continue using existing context or the best listed choice"
		return clarifyPayload("clarify_unavailable", req, extra), ErrClarifyUnavailable
	}
	resp, err := t.callback.Clarify(ctx, req)
	if err != nil {
		status := "clarify_route_missing"
		wrapped := ErrClarifyRouteMissing
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrClarifyTimeout) {
			status = "clarify_timeout"
			wrapped = ErrClarifyTimeout
		}
		extra["error"] = status
		return clarifyPayload(status, req, extra), fmt.Errorf("%w: %v", wrapped, err)
	}
	extra["user_response"] = strings.TrimSpace(resp.UserResponse)
	return clarifyPayload("clarify_answered", req, extra), nil
}

func parseClarifyRequest(raw json.RawMessage) (ClarifyRequest, bool, error) {
	var obj map[string]any
	if len(raw) == 0 {
		obj = map[string]any{}
	} else if err := json.Unmarshal(raw, &obj); err != nil {
		return ClarifyRequest{}, false, fmt.Errorf("invalid_json")
	}
	question, _ := obj["question"].(string)
	question = strings.TrimSpace(question)
	req := ClarifyRequest{Question: question}
	if question == "" {
		return req, false, fmt.Errorf("question_required")
	}
	choicesRaw, hasChoices := obj["choices"]
	if !hasChoices || choicesRaw == nil {
		return req, false, nil
	}
	items, ok := choicesRaw.([]any)
	if !ok {
		return req, false, fmt.Errorf("choices_must_be_list")
	}
	choices := make([]string, 0, len(items))
	for _, item := range items {
		choice := strings.TrimSpace(fmt.Sprint(item))
		if choice == "" {
			continue
		}
		choices = append(choices, choice)
	}
	truncated := len(choices) > 4
	if truncated {
		choices = choices[:4]
	}
	req.Choices = choices
	return req, truncated, nil
}

func clarifyPayload(status string, req ClarifyRequest, extra map[string]any) json.RawMessage {
	payload := map[string]any{
		"status":   status,
		"question": req.Question,
	}
	if len(req.Choices) > 0 {
		payload["choices_offered"] = append([]string(nil), req.Choices...)
	}
	for k, v := range extra {
		payload[k] = v
	}
	out, _ := json.Marshal(payload)
	return out
}
