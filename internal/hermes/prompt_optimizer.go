package hermes

import (
	"context"
	"math"
	"strings"
)

type PromptVariant struct {
	ID      string
	Prompt  string
	Score   float64
}

type PromptOptimizer struct {
	evaluator   *PromptEvaluator
	mutationRate float64
	maxIterations int
	improvementThreshold float64
	toolCallsFn func(context.Context, string) ([]string, error)
}

func NewPromptOptimizer(evaluator *PromptEvaluator, toolCallsFn func(context.Context, string) ([]string, error)) *PromptOptimizer {
	return &PromptOptimizer{
		evaluator:   evaluator,
		mutationRate: 0.3,
		maxIterations: 50,
		improvementThreshold: 0.05,
		toolCallsFn: toolCallsFn,
	}
}

func (o *PromptOptimizer) Optimize(ctx context.Context, basePrompt string) (*PromptVariant, error) {
	baseResults, err := o.evaluator.Evaluate(ctx, "base", func(ctx context.Context, p string) ([]string, error) {
		return o.toolCallsFn(ctx, p)
	})
	if err != nil {
		return nil, err
	}
	baseScore := AggregateScore(baseResults)

	best := &PromptVariant{ID: "base", Prompt: basePrompt, Score: baseScore}

	for i := 0; i < o.maxIterations; i++ {
		variant := o.mutate(best.Prompt)
		results, err := o.evaluator.Evaluate(ctx, variant.ID, func(ctx context.Context, p string) ([]string, error) {
			return o.toolCallsFn(ctx, p)
		})
		if err != nil {
			continue
		}
		score := AggregateScore(results)

		if score > best.Score+o.improvementThreshold {
			best = &PromptVariant{ID: variant.ID, Prompt: variant.Prompt, Score: score}
		}

		if score >= 0.95 {
			break
		}
	}

	return best, nil
}

func (o *PromptOptimizer) mutate(prompt string) PromptVariant {
	words := strings.Fields(prompt)
	if len(words) == 0 {
		return PromptVariant{ID: "mut_empty", Prompt: prompt}
	}
	numMutations := int(math.Max(1, math.Ceil(float64(len(words))*o.mutationRate)))
	id := "mut_" + strings.Join(words[:min(3, len(words))], "_")

	mutated := words
	for i := 0; i < numMutations && i < len(mutated); i++ {
		pos := i * (len(mutated) / (numMutations + 1))
		if pos >= len(mutated) {
			pos = len(mutated) - 1
		}
		mutated = append(mutated[:pos], mutated[pos+1:]...)
		if len(mutated) == 0 {
			break
		}
	}

	return PromptVariant{ID: id, Prompt: strings.Join(mutated, " ")}
}
