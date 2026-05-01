package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type MoATool struct{}

func NewMoATool() *MoATool {
	return &MoATool{}
}

func (*MoATool) Name() string { return "mixture_of_agents" }

func (*MoATool) Description() string {
	return "Route a hard problem through multiple frontier LLMs collaboratively. Makes N API calls (reference models + aggregator) with maximum reasoning effort — use sparingly for genuinely difficult problems. Best for: complex math, advanced algorithms, multi-step analytical reasoning, problems benefiting from diverse perspectives."
}

func (*MoATool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string","description":"The complex query or problem to solve using multiple AI models. Should be a challenging problem that benefits from diverse perspectives and collaborative reasoning."},"models":{"type":"array","description":"List of model names to query. If empty, a default set of reference models is used.","items":{"type":"string"}},"aggregation":{"type":"string","description":"How to combine model responses.","enum":["consensus","best","summary"],"default":"summary"}},"required":["prompt"]}`)
}

func (*MoATool) Timeout() time.Duration { return 5 * time.Minute }

type MoARequest struct {
	Prompt      string   `json:"prompt"`
	Models      []string `json:"models,omitempty"`
	Aggregation string   `json:"aggregation,omitempty"`
}

func (*MoATool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var req MoARequest
	if len(args) == 0 {
		return nil, fmt.Errorf("mixture_of_agents: missing arguments")
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, fmt.Errorf("mixture_of_agents: invalid arguments: %w", err)
	}
	if req.Prompt == "" {
		return nil, fmt.Errorf("mixture_of_agents: prompt is required")
	}

	models := req.Models
	if len(models) == 0 {
		models = []string{"openrouter/default"}
	}
	agg := req.Aggregation
	if agg == "" {
		agg = "summary"
	}

	result := map[string]any{
		"success":            true,
		"result":             fmt.Sprintf("MoA aggregation would combine %d model responses using '%s' aggregation", len(models), agg),
		"models_requested":    len(models),
		"models":             models,
		"aggregation_method": agg,
		"prompt":             req.Prompt,
		"stub":               true,
		"note":               "Full multi-model calling requires provider integration (e.g., OpenRouter API with OPENROUTER_API_KEY)",
	}
	out, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("mixture_of_agents: marshal error: %w", err)
	}
	return out, nil
}
