package llm

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing"
)

type CapabilityRouter = routing.CapabilityRouter

func NewCapabilityRouter(cheap, capable []string) *CapabilityRouter {
	return routing.NewCapabilityRouter(cheap, capable)
}

func WithCapabilityRouter(ctx context.Context, router *CapabilityRouter) context.Context {
	return routing.WithCapabilityRouter(ctx, router)
}

func GetCapabilityRouter(ctx context.Context) *CapabilityRouter {
	return routing.GetCapabilityRouter(ctx)
}
