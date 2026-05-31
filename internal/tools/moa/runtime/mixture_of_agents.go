package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/moa/contract"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools/toolkit"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MixtureOfAgentsName     = contract.ToolName
	moaDefaultTimeout       = 120 * time.Second
	moaDefaultMinReferences = 2
	moaDefaultMaxReferences = 5
)

var ErrMOAInsufficientReferences = contract.Error("insufficient successful references")

// MOAConfig configures the mixture-of-agents tool.
type MOAConfig struct {
	Timeout       time.Duration
	MinReferences int
	MaxReferences int
}

// MOATool implements Hermes' mixture_of_agents_tool.py.
type MOATool struct {
	cfg    MOAConfig
	router MOARouter
}

// MOARouter abstracts the provider dispatch so tests can inject fakes.
type MOARouter interface {
	Call(ctx context.Context, model, prompt string) (string, error)
}

// NewMOATool creates a mixture-of-agents tool.
func NewMOATool(cfg MOAConfig, router MOARouter) toolkit.Tool {
	return &MOATool{cfg: normalizeMOAConfig(cfg), router: router}
}

func normalizeMOAConfig(cfg MOAConfig) MOAConfig {
	if cfg.Timeout <= 0 {
		cfg.Timeout = moaDefaultTimeout
	}
	if cfg.MinReferences <= 0 {
		cfg.MinReferences = moaDefaultMinReferences
	}
	if cfg.MaxReferences <= 0 {
		cfg.MaxReferences = moaDefaultMaxReferences
	}
	if cfg.MinReferences > cfg.MaxReferences {
		cfg.MinReferences = cfg.MaxReferences
	}
	return cfg
}

func (t *MOATool) Name() string { return MixtureOfAgentsName }
func (t *MOATool) Description() string {
	return "Runs multiple models in parallel and synthesizes their responses with an aggregator model."
}
func (t *MOATool) Timeout() time.Duration { return t.cfg.Timeout }
func (t *MOATool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"prompt":{"type":"string","description":"The prompt to send to reference models"},"models":{"type":"array","items":{"type":"string"},"description":"Reference model names"},"aggregator":{"type":"string","description":"Model used to synthesize reference responses (defaults to first reference model)"}},"required":["prompt","models"]}`)
}

type moaArgs struct {
	Prompt     string   `json:"prompt"`
	Models     []string `json:"models"`
	Aggregator string   `json:"aggregator,omitempty"`
}

type moaResult struct {
	Success        bool             `json:"success"`
	Response       string           `json:"response,omitempty"`
	Error          string           `json:"error,omitempty"`
	ReferenceCount int              `json:"reference_count"`
	ModelResults   []moaModelResult `json:"model_results,omitempty"`
	DebugEvidence  moaDebugEvidence `json:"debug_evidence,omitempty"`
}

type moaModelResult struct {
	Model   string `json:"model"`
	Success bool   `json:"success"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
	Retries int    `json:"retries"`
}

type moaDebugEvidence struct {
	ModelsAttempted int    `json:"models_attempted"`
	ModelsSucceeded int    `json:"models_succeeded"`
	MinRequired     int    `json:"min_required"`
	TotalAttempts   int    `json:"total_attempts"`
	AggregatorModel string `json:"aggregator_model"`
}

func (t *MOATool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in moaArgs
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, contract.Errorf("invalid args: %w", err)
	}
	if in.Prompt == "" {
		return nil, contract.Error("prompt is required")
	}

	models := in.Models
	if len(models) == 0 {
		return nil, contract.Error("at least one reference model is required")
	}
	if len(models) > t.cfg.MaxReferences {
		models = models[:t.cfg.MaxReferences]
	}

	aggregator := in.Aggregator
	if aggregator == "" {
		aggregator = models[0]
	}

	// Fan out to reference models in parallel with retry isolation.
	ctx, cancel := context.WithTimeout(ctx, t.cfg.Timeout)
	defer cancel()

	results := t.fanOutReferences(ctx, models, in.Prompt)
	successes, failures := partitionResults(results)

	if len(successes) < t.cfg.MinReferences {
		return json.Marshal(moaResult{
			Success: false,
			Error: fmt.Sprintf("Only %d of %d reference models succeeded (minimum %d required): %v",
				len(successes), len(models), t.cfg.MinReferences, failureMessages(failures)),
			ModelResults:   results,
			ReferenceCount: len(models),
			DebugEvidence: moaDebugEvidence{
				ModelsAttempted: len(models),
				ModelsSucceeded: len(successes),
				MinRequired:     t.cfg.MinReferences,
				TotalAttempts:   totalAttempts(results),
				AggregatorModel: aggregator,
			},
		})
	}

	// Aggregate successful references.
	response, err := t.aggregate(ctx, aggregator, in.Prompt, successes)
	if err != nil {
		return json.Marshal(moaResult{
			Success:        false,
			Error:          fmt.Sprintf("Aggregator model %q failed: %v", aggregator, err),
			ModelResults:   results,
			ReferenceCount: len(models),
			DebugEvidence: moaDebugEvidence{
				ModelsAttempted: len(models),
				ModelsSucceeded: len(successes),
				MinRequired:     t.cfg.MinReferences,
				TotalAttempts:   totalAttempts(results),
				AggregatorModel: aggregator,
			},
		})
	}

	return json.Marshal(moaResult{
		Success:        true,
		Response:       response,
		ModelResults:   results,
		ReferenceCount: len(models),
		DebugEvidence: moaDebugEvidence{
			ModelsAttempted: len(models),
			ModelsSucceeded: len(successes),
			MinRequired:     t.cfg.MinReferences,
			TotalAttempts:   totalAttempts(results),
			AggregatorModel: aggregator,
		},
	})
}

func (t *MOATool) fanOutReferences(ctx context.Context, models []string, prompt string) []moaModelResult {
	results := make([]moaModelResult, len(models))
	var wg sync.WaitGroup
	for i, model := range models {
		wg.Add(1)
		go func(i int, model string) {
			defer wg.Done()
			results[i] = t.runReferenceModel(ctx, model, prompt)
		}(i, model)
	}
	wg.Wait()
	return results
}

func (t *MOATool) runReferenceModel(ctx context.Context, model, prompt string) moaModelResult {
	const maxRetries = 2
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		content, err := t.router.Call(ctx, model, prompt)
		if err == nil {
			return moaModelResult{Model: model, Success: true, Content: content, Retries: attempt}
		}
		lastErr = err
	}
	return moaModelResult{Model: model, Success: false, Error: lastErr.Error(), Retries: maxRetries}
}

func (t *MOATool) aggregate(ctx context.Context, aggregator, original string, refs []moaModelResult) (string, error) {
	sort.SliceStable(refs, func(i, j int) bool { return refs[i].Model < refs[j].Model })

	var b strings.Builder
	b.WriteString("You are a synthesizer of multiple AI model responses. ")
	b.WriteString("Below are responses from different models to the same prompt. ")
	b.WriteString("Create one cohesive, well-organized answer that captures the best insights from each response, resolves contradictions, and provides a clear final answer.\n\n")
	b.WriteString("## Original Prompt\n")
	b.WriteString(original)
	b.WriteString("\n\n## Reference Model Responses\n\n")
	for i, r := range refs {
		b.WriteString(fmt.Sprintf("### Model %d: %s\n%s\n\n", i+1, r.Model, r.Content))
	}
	return t.router.Call(ctx, aggregator, b.String())
}

func partitionResults(results []moaModelResult) (ok, fail []moaModelResult) {
	for _, r := range results {
		if r.Success {
			ok = append(ok, r)
		} else {
			fail = append(fail, r)
		}
	}
	return
}

func failureMessages(failures []moaModelResult) string {
	msgs := make([]string, len(failures))
	for i, f := range failures {
		msgs[i] = f.Model + ": " + f.Error
	}
	return strings.Join(msgs, "; ")
}

func totalAttempts(results []moaModelResult) int {
	sum := len(results)
	for _, r := range results {
		sum += r.Retries
	}
	return sum
}
