package mutation

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/prompts/eval/model"
)

func GenerateVariants(base model.Variant, iteration, limit int) []model.Variant {
	prompt := strings.TrimSpace(base.Prompt)
	if prompt == "" {
		prompt = "Be accurate."
	}
	strategies := []struct {
		idSuffix string
		line     string
	}{
		{
			idSuffix: "tool-selection",
			line:     "When a task maps to a tool, call the exact required tool before answering.",
		},
		{
			idSuffix: "response-quality",
			line:     "Report the concrete outcome in the response using task-specific terms.",
		},
		{
			idSuffix: "decomposition",
			line:     "Break multi-step tasks into search, edit, verify, and document steps.",
		},
		{
			idSuffix: "command-safety",
			line:     "Classify commands before execution and block destructive operations.",
		},
		{
			idSuffix: "research-quality",
			line:     "Evaluate external-project research with source-backed maturity, license, fit, limitations, and a test-backed workflow recommendation.",
		},
	}

	if limit <= 0 || limit > len(strategies) {
		limit = len(strategies)
	}
	variants := make([]model.Variant, 0, limit)
	for i := 0; i < limit; i++ {
		strategy := strategies[i]
		if strings.Contains(prompt, strategy.line) {
			continue
		}
		variants = append(variants, model.Variant{
			ID:     fmt.Sprintf("%s_iter%d_%s", StablePrefix(base.ID), iteration, strategy.idSuffix),
			Prompt: prompt + "\n" + strategy.line,
		})
	}
	if len(variants) == 0 {
		variants = append(variants, model.Variant{
			ID:     fmt.Sprintf("%s_iter%d_stable", StablePrefix(base.ID), iteration),
			Prompt: prompt + "\nKeep the same behavior when evaluation scores do not improve.",
		})
	}
	return variants
}

func StablePrefix(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "variant"
	}
	id = strings.NewReplacer(" ", "_", "\n", "_", "\t", "_").Replace(id)
	return strings.Trim(id, "_")
}
