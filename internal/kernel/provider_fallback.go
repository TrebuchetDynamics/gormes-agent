package kernel

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func (k *Kernel) maxEmptyResponseRetries() int {
	if k.cfg.MaxEmptyResponseRetries > 0 {
		return k.cfg.MaxEmptyResponseRetries
	}
	return defaultMaxEmptyResponseRetries
}

func emptyFinalResponse(draft string, final hermes.Event) bool {
	if final.FinishReason == "tool_calls" || len(final.ToolCalls) > 0 {
		return false
	}
	return strings.TrimSpace(draft) == ""
}

func (k *Kernel) activateFallback(ctx context.Context, request *hermes.ChatRequest, routes []hermes.ModelRoute, index *int) bool {
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

func (k *Kernel) updateContextForFallback(route hermes.ModelRoute, tools []hermes.ToolDescriptor) {
	if k.cfg.ContextEngine == nil {
		return
	}
	metadata := hermes.LookupModelMetadata(hermes.ModelRegistryQuery{
		Provider: route.Provider,
		Model:    route.Model,
	})
	contextResolution := hermes.ResolveDisplayContextLength(hermes.ModelContextQuery{
		Provider: route.Provider,
		Model:    route.Model,
		ModelInfo: hermes.ModelContextMetadata{
			ContextWindow: metadata.RawContextWindow,
		},
	})
	update := hermes.ContextModelContext{
		Model:           route.Model,
		Provider:        route.Provider,
		ContextLength:   contextResolution.ContextLength,
		ToolDescriptors: append([]hermes.ToolDescriptor(nil), tools...),
	}
	k.cfg.ContextEngine.UpdateModelContext(update)
}
