package routing

import (
	"context"
	"strings"
)

type CapabilityRouter struct {
	cheapTier   []string
	capableTier []string
}

func NewCapabilityRouter(cheap, capable []string) *CapabilityRouter {
	return &CapabilityRouter{cheapTier: cheap, capableTier: capable}
}

func (r *CapabilityRouter) Route(prompt string) string {
	complexity := estimateComplexity(prompt)
	if complexity == "simple" && len(r.cheapTier) > 0 {
		return r.cheapTier[0]
	}
	if len(r.capableTier) > 0 {
		return r.capableTier[0]
	}
	if len(r.cheapTier) > 0 {
		return r.cheapTier[0]
	}
	return ""
}

func estimateComplexity(prompt string) string {
	indicators := map[string]bool{
		"refactor":     true,
		"rewrite":      true,
		"architecture": true,
		"design":       true,
		"multi-file":   true,
		"system":       true,
		"migrate":      true,
	}
	lower := strings.ToLower(prompt)
	words := strings.Fields(lower)
	complexCount := 0
	for _, w := range words {
		if indicators[w] {
			complexCount++
		}
	}
	if complexCount > 0 || len(words) > 50 {
		return "complex"
	}
	return "simple"
}

type capabilityRouterKey struct{}

func WithCapabilityRouter(ctx context.Context, router *CapabilityRouter) context.Context {
	return context.WithValue(ctx, capabilityRouterKey{}, router)
}

func GetCapabilityRouter(ctx context.Context) *CapabilityRouter {
	r, _ := ctx.Value(capabilityRouterKey{}).(*CapabilityRouter)
	return r
}
