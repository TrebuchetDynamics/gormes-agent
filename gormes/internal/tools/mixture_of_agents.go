package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/gormes-agent/gormes/internal/hermes"
)

const (
	defaultMOAToolTimeout         = 2 * time.Minute
	defaultMOAMinSuccess          = 1
	defaultMOAFailureResponseText = "MoA processing failed. Please try again or use a single model for this query."
)

var defaultMOAReferenceModels = []string{
	"anthropic/claude-opus-4.6",
	"google/gemini-3-pro-preview",
	"openai/gpt-5.4-pro",
	"deepseek/deepseek-v3.2",
}

const (
	defaultMOAAggregatorModel = "anthropic/claude-opus-4.6"
	moaAggregatorSystemPrompt = "You have been provided with a set of responses from various open-source models to the latest user query. Your task is to synthesize these responses into a single, high-quality response. It is crucial to critically evaluate the information provided in these responses, recognizing that some of it may be biased or incorrect. Your response should not simply replicate the given answers but should offer a refined, accurate, and comprehensive reply to the instruction. Ensure your response is well-structured, coherent, and adheres to the highest standards of accuracy and reliability.\n\nResponses from models:"
)

type MixtureOfAgentsTool struct {
	Client                 hermes.Client
	ClientFactory          func() hermes.Client
	ReferenceModels        []string
	AggregatorModel        string
	MinSuccessfulResponses int
	TimeoutD               time.Duration
}

type mixtureOfAgentsArgs struct {
	UserPrompt      string   `json:"user_prompt"`
	ReferenceModels []string `json:"reference_models"`
	AggregatorModel string   `json:"aggregator_model"`
}

type mixtureOfAgentsModels struct {
	ReferenceModels []string `json:"reference_models"`
	AggregatorModel string   `json:"aggregator_model"`
}

type mixtureOfAgentsResult struct {
	Success      bool                  `json:"success"`
	Response     string                `json:"response"`
	ModelsUsed   mixtureOfAgentsModels `json:"models_used"`
	FailedModels []string              `json:"failed_models,omitempty"`
	Error        string                `json:"error,omitempty"`
}

type mixtureOfAgentsReferenceResult struct {
	Model   string
	Content string
	Err     error
}

var _ Tool = (*MixtureOfAgentsTool)(nil)

func (*MixtureOfAgentsTool) Name() string { return "mixture_of_agents" }

func (*MixtureOfAgentsTool) Description() string {
	return "Route a hard problem through multiple frontier LLMs collaboratively. Fans out reference-model drafts, then asks an aggregator model to synthesize the best final answer."
}

func (*MixtureOfAgentsTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"user_prompt":{"type":"string","description":"The difficult prompt or problem to solve with multiple models."},
			"reference_models":{"type":"array","items":{"type":"string"},"description":"Optional override for the reference-model fanout list."},
			"aggregator_model":{"type":"string","description":"Optional override for the synthesis model."}
		},
		"required":["user_prompt"]
	}`)
}

func (t *MixtureOfAgentsTool) Timeout() time.Duration {
	if t == nil || t.TimeoutD <= 0 {
		return defaultMOAToolTimeout
	}
	return t.TimeoutD
}

func (t *MixtureOfAgentsTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in mixtureOfAgentsArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("mixture_of_agents: invalid args: %w", err)
	}

	userPrompt := strings.TrimSpace(in.UserPrompt)
	if userPrompt == "" {
		return nil, errors.New("mixture_of_agents: user_prompt is required")
	}

	client := t.client()
	if client == nil {
		return nil, errors.New("mixture_of_agents: no openrouter client configured")
	}

	referenceModels := firstNonEmptyModelList(in.ReferenceModels, t.ReferenceModels, defaultMOAReferenceModels)
	aggregatorModel := firstNonEmptyString(in.AggregatorModel, t.AggregatorModel, defaultMOAAggregatorModel)
	result := runMixtureOfAgents(ctx, client, userPrompt, referenceModels, aggregatorModel, t.minSuccessfulResponses())
	return json.Marshal(result)
}

func (t *MixtureOfAgentsTool) minSuccessfulResponses() int {
	if t != nil && t.MinSuccessfulResponses > 0 {
		return t.MinSuccessfulResponses
	}
	return defaultMOAMinSuccess
}

func (t *MixtureOfAgentsTool) client() hermes.Client {
	if t != nil && t.Client != nil {
		return t.Client
	}
	if t != nil && t.ClientFactory != nil {
		return t.ClientFactory()
	}
	return newOpenRouterClientFromEnv()
}

func MixtureOfAgentsAvailable() bool {
	_, _, ok := openRouterEnvConfig()
	return ok
}

func newOpenRouterClientFromEnv() hermes.Client {
	endpoint, key, ok := openRouterEnvConfig()
	if !ok {
		return nil
	}
	return hermes.NewClient("openrouter", hermes.EffectiveEndpoint("openrouter", endpoint), key)
}

func runMixtureOfAgents(ctx context.Context, client hermes.Client, userPrompt string, referenceModels []string, aggregatorModel string, minSuccess int) mixtureOfAgentsResult {
	result := newMixtureOfAgentsResult(referenceModels, aggregatorModel)

	referenceResults := queryReferenceModels(ctx, client, userPrompt, referenceModels)
	successful := make([]string, 0, len(referenceResults))
	failedModels := make([]string, 0, len(referenceResults))
	for _, ref := range referenceResults {
		if ref.Err != nil {
			failedModels = append(failedModels, ref.Model)
			continue
		}
		successful = append(successful, ref.Content)
	}
	if len(failedModels) > 0 {
		result.FailedModels = failedModels
	}
	if len(successful) < minSuccess {
		result.Error = fmt.Sprintf("Error in MoA processing: insufficient successful reference models (%d/%d). Need at least %d successful responses.", len(successful), len(referenceModels), minSuccess)
		return result
	}

	finalResponse, err := queryMOAModel(ctx, client, aggregatorModel, []hermes.Message{
		{Role: "system", Content: constructAggregatorPrompt(successful)},
		{Role: "user", Content: userPrompt},
	})
	if err != nil {
		result.Error = "Error in MoA processing: " + err.Error()
		return result
	}

	result.Success = true
	result.Response = finalResponse
	return result
}

func queryReferenceModels(ctx context.Context, client hermes.Client, userPrompt string, referenceModels []string) []mixtureOfAgentsReferenceResult {
	results := make([]mixtureOfAgentsReferenceResult, len(referenceModels))
	var wg sync.WaitGroup
	for i, model := range referenceModels {
		i := i
		model := model
		wg.Add(1)
		go func() {
			defer wg.Done()
			content, err := queryMOAModel(ctx, client, model, []hermes.Message{{Role: "user", Content: userPrompt}})
			results[i] = mixtureOfAgentsReferenceResult{Model: model, Content: content, Err: err}
		}()
	}
	wg.Wait()
	return results
}

func newMixtureOfAgentsResult(referenceModels []string, aggregatorModel string) mixtureOfAgentsResult {
	return mixtureOfAgentsResult{
		Success:  false,
		Response: defaultMOAFailureResponseText,
		ModelsUsed: mixtureOfAgentsModels{
			ReferenceModels: append([]string(nil), referenceModels...),
			AggregatorModel: aggregatorModel,
		},
	}
}

func openRouterEnvConfig() (endpoint string, key string, ok bool) {
	key = strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	endpoint = strings.TrimSpace(os.Getenv("GORMES_ENDPOINT"))
	if key == "" && strings.EqualFold(strings.TrimSpace(os.Getenv("GORMES_PROVIDER")), "openrouter") {
		key = strings.TrimSpace(os.Getenv("GORMES_API_KEY"))
	}
	return endpoint, key, key != ""
}

func queryMOAModel(ctx context.Context, client hermes.Client, model string, messages []hermes.Message) (string, error) {
	stream, err := client.OpenStream(ctx, hermes.ChatRequest{
		Model:    model,
		Stream:   true,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("%s: open stream: %w", model, err)
	}
	defer stream.Close()

	var (
		text      strings.Builder
		reasoning strings.Builder
		gotDone   bool
		final     hermes.Event
	)

	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("%s: recv: %w", model, err)
		}
		switch ev.Kind {
		case hermes.EventReasoning:
			reasoning.WriteString(ev.Reasoning)
		case hermes.EventToken:
			text.WriteString(ev.Token)
		case hermes.EventDone:
			gotDone = true
			final = ev
		}
	}

	if !gotDone {
		return "", fmt.Errorf("%s: stream closed without finish_reason", model)
	}
	if final.FinishReason == "tool_calls" {
		return "", fmt.Errorf("%s: unexpected tool_calls finish", model)
	}

	content := strings.TrimSpace(text.String())
	if content == "" {
		content = strings.TrimSpace(reasoning.String())
	}
	if content == "" {
		return "", fmt.Errorf("%s: empty response", model)
	}
	return content, nil
}

func constructAggregatorPrompt(responses []string) string {
	lines := make([]string, 0, len(responses))
	for i, response := range responses {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, response))
	}
	if len(lines) == 0 {
		return moaAggregatorSystemPrompt
	}
	return moaAggregatorSystemPrompt + "\n\n" + strings.Join(lines, "\n")
}

func firstNonEmptyModelList(candidates ...[]string) []string {
	for _, candidate := range candidates {
		trimmed := make([]string, 0, len(candidate))
		for _, raw := range candidate {
			if model := strings.TrimSpace(raw); model != "" {
				trimmed = append(trimmed, model)
			}
		}
		if len(trimmed) > 0 {
			return append([]string(nil), trimmed...)
		}
	}
	return nil
}

func firstNonEmptyString(candidates ...string) string {
	for _, candidate := range candidates {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
