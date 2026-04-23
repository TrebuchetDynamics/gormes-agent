package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ClarifyPrompter is injected by interactive hosts that can surface a
// question to the user and wait for their answer.
type ClarifyPrompter func(context.Context, ClarifyRequest) (ClarifyReply, error)

// ClarifyRequest is the validated prompt payload forwarded to the host.
type ClarifyRequest struct {
	Question   string
	Choices    []string
	AllowOther bool
}

// ClarifyReply is the host's normalized answer payload.
type ClarifyReply struct {
	Answer string
}

// ClarifyTool asks the user for clarification through an injected host
// callback. It remains unavailable until Prompter is set.
type ClarifyTool struct {
	Prompter ClarifyPrompter
}

var _ Tool = (*ClarifyTool)(nil)

func (*ClarifyTool) Name() string { return "clarify" }

func (*ClarifyTool) Description() string {
	return "Ask the user a question when you need clarification, feedback, or a decision before proceeding. Supports freeform answers or up to 4 multiple-choice options."
}

func (*ClarifyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"question":{"type":"string","description":"Question to ask the user."},
			"choices":{
				"type":"array",
				"description":"Optional multiple-choice options. Maximum 4 entries.",
				"items":{"type":"string"},
				"maxItems":4
			},
			"allow_other":{
				"type":"boolean",
				"description":"When true (default), the user may answer with freeform text even when choices are provided."
			}
		},
		"required":["question"]
	}`)
}

func (*ClarifyTool) Timeout() time.Duration { return 120 * time.Second }

func (t *ClarifyTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Question   string   `json:"question"`
		Choices    []string `json:"choices"`
		AllowOther *bool    `json:"allow_other"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("clarify: invalid args: %w", err)
	}

	req, err := normalizeClarifyRequest(in.Question, in.Choices, in.AllowOther)
	if err != nil {
		return nil, err
	}
	if t.Prompter == nil {
		return nil, errors.New("clarify: no clarify prompter configured")
	}

	reply, err := t.Prompter(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("clarify: prompt failed: %w", err)
	}
	answer := strings.TrimSpace(reply.Answer)
	if answer == "" {
		return nil, errors.New("clarify: empty answer")
	}

	selectedChoice := selectedClarifyChoice(answer, req.Choices)
	if len(req.Choices) > 0 && selectedChoice == "" && !req.AllowOther {
		return nil, errors.New("clarify: answer must match one of the provided choices")
	}

	out := struct {
		Question       string `json:"question"`
		Answer         string `json:"answer"`
		SelectedChoice string `json:"selected_choice,omitempty"`
		UsedOther      bool   `json:"used_other"`
	}{
		Question:       req.Question,
		Answer:         answer,
		SelectedChoice: selectedChoice,
		UsedOther:      len(req.Choices) > 0 && selectedChoice == "",
	}
	return json.Marshal(out)
}

func normalizeClarifyRequest(question string, choices []string, allowOther *bool) (ClarifyRequest, error) {
	req := ClarifyRequest{
		Question:   strings.TrimSpace(question),
		AllowOther: true,
	}
	if allowOther != nil {
		req.AllowOther = *allowOther
	}
	if req.Question == "" {
		return ClarifyRequest{}, errors.New("clarify: 'question' is required and must be non-empty")
	}
	if len(choices) > 4 {
		return ClarifyRequest{}, errors.New("clarify: at most 4 choices are allowed")
	}

	req.Choices = make([]string, 0, len(choices))
	seen := make(map[string]struct{}, len(choices))
	for _, raw := range choices {
		choice := strings.TrimSpace(raw)
		if choice == "" {
			return ClarifyRequest{}, errors.New("clarify: choices must be non-empty")
		}
		if _, ok := seen[choice]; ok {
			return ClarifyRequest{}, fmt.Errorf("clarify: duplicate choice %q", choice)
		}
		seen[choice] = struct{}{}
		req.Choices = append(req.Choices, choice)
	}
	return req, nil
}

func selectedClarifyChoice(answer string, choices []string) string {
	for _, choice := range choices {
		if answer == choice {
			return choice
		}
	}
	return ""
}
