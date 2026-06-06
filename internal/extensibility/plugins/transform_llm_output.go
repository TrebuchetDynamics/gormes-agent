package plugins

import (
	"context"
	"log/slog"
)

// TransformLLMOutputInput is the argument bundle passed to each
// transform_llm_output hook callback. It mirrors the upstream Hermes
// hook signature from run_agent.py:L14279–14298.
type TransformLLMOutputInput struct {
	// ResponseText is the assistant's final response text for this turn
	// after the tool-calling loop completes.
	ResponseText string
	// SessionID is the current session identifier (may be empty).
	SessionID string
	// Model is the model that produced the response (e.g. "anthropic/claude-sonnet-4.6").
	Model string
	// Platform is the delivery platform (e.g. "cli", "telegram", "discord").
	Platform string
}

// TransformLLMOutputFunc is a single hook callback. It receives the raw
// LLM output and may reshape, redact, or filter the content.
//
// Return values:
//   - non-empty string → replaces the response text (first non-empty wins)
//   - empty string ""    → leaves the response unchanged
//   - error              → logged as a warning; original response preserved
//
// This is fail-open: a misbehaving hook never blocks or corrupts the turn.
type TransformLLMOutputFunc func(ctx context.Context, input TransformLLMOutputInput) (string, error)

// TransformLLMOutputRunner executes the ordered chain of registered hooks.
// The default implementation is *TransformLLMOutputRegistry.
type TransformLLMOutputRunner interface {
	// Run executes all registered hooks in registration order.
	// First non-empty result wins immediately (short-circuit).
	// Errors from individual hooks are logged and skipped; the original
	// response is preserved when all hooks return empty or all fail.
	Run(ctx context.Context, input TransformLLMOutputInput) string
}

// TransformLLMOutputRegistry holds an ordered chain of transform hooks
// and implements TransformLLMOutputRunner.
type TransformLLMOutputRegistry struct {
	hooks []TransformLLMOutputFunc
	log   *slog.Logger
}

// NewTransformLLMOutputRegistry returns a ready-to-use registry.
func NewTransformLLMOutputRegistry(log *slog.Logger) *TransformLLMOutputRegistry {
	if log == nil {
		log = slog.Default()
	}
	return &TransformLLMOutputRegistry{log: log}
}

// Register appends a hook callback. Hooks are invoked in registration order.
func (r *TransformLLMOutputRegistry) Register(hook TransformLLMOutputFunc) {
	r.hooks = append(r.hooks, hook)
}

// Run executes all registered hooks in order against the input.
// First non-empty string result wins (short-circuits). Errors are logged
// and the offending hook is skipped. When no hook returns a non-empty
// string, the original ResponseText is returned unchanged.
func (r *TransformLLMOutputRegistry) Run(ctx context.Context, input TransformLLMOutputInput) string {
	log := r.log
	if log == nil {
		log = slog.Default()
	}
	for _, hook := range r.hooks {
		if hook == nil {
			log.Warn("transform_llm_output hook missing; preserving original response")
			continue
		}
		result, err := hook(ctx, input)
		if err != nil {
			log.Warn("transform_llm_output hook failed; preserving original response", "err", err)
			continue
		}
		if result != "" {
			return result
		}
	}
	return input.ResponseText
}
