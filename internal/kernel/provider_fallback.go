package kernel

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
)

func (k *Kernel) maxEmptyResponseRetries() int {
	if k.cfg.MaxEmptyResponseRetries > 0 {
		return k.cfg.MaxEmptyResponseRetries
	}
	return defaultMaxEmptyResponseRetries
}

func emptyFinalResponse(draft string, final llm.Event) bool {
	if final.FinishReason == "tool_calls" || len(final.ToolCalls) > 0 {
		return false
	}
	return strings.TrimSpace(draft) == ""
}

func (k *Kernel) activateFallback(ctx context.Context, request *llm.ChatRequest, routes []llm.ModelRoute, index *int) bool {
	if k.cfg.FallbackClientFactory == nil {
		k.addSoul("fallback_unavailable: no fallback client factory")
		return false
	}
	for *index < len(routes) {
		route := routes[*index]
		*index++
		if strings.TrimSpace(route.Provider) == "" || strings.TrimSpace(route.Model) == "" {
			k.addSoul("fallback_config_invalid: incomplete fallback route")
			continue
		}
		route = route.ResolveFallbackCredential(os.Getenv)
		client, err := k.cfg.FallbackClientFactory(ctx, route)
		if err != nil || client == nil {
			if err == nil {
				err = fmt.Errorf("fallback client unavailable")
			}
			k.addSoul(fmt.Sprintf("fallback_unavailable: %s/%s: %s", route.Provider, route.Model, err))
			continue
		}

		k.client = client
		k.activeModel = route.Model
		request.Model = route.Model
		k.updateContextForFallback(route, request.Tools)
		k.addSoul(fmt.Sprintf("fallback_activated: %s/%s", route.Provider, route.Model))
		k.phase = PhaseConnecting
		k.emitFrame("fallback activated")
		return true
	}
	k.addSoul("fallback_unavailable: fallback chain exhausted")
	return false
}

func (k *Kernel) updateContextForFallback(route llm.ModelRoute, tools []llm.ToolDescriptor) {
	if k.cfg.ContextEngine == nil {
		return
	}
	metadata := llm.LookupModelMetadata(llm.ModelRegistryQuery{
		Provider: route.Provider,
		Model:    route.Model,
	})
	contextResolution := llm.ResolveDisplayContextLength(llm.ModelContextQuery{
		Provider: route.Provider,
		Model:    route.Model,
		ModelInfo: llm.ModelContextMetadata{
			ContextWindow: metadata.RawContextWindow,
		},
	})
	update := llm.ContextModelContext{
		Model:           route.Model,
		Provider:        route.Provider,
		ContextLength:   contextResolution.ContextLength,
		ToolDescriptors: append([]llm.ToolDescriptor(nil), tools...),
	}
	k.cfg.ContextEngine.UpdateModelContext(update)
}
