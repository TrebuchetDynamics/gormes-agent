package routing

import (
	"context"

	capabilitypolicy "github.com/TrebuchetDynamics/gormes-agent/internal/llm/routing/capability"
)

type CapabilityRouter = capabilitypolicy.CapabilityRouter

func NewCapabilityRouter(cheap, capable []string) *CapabilityRouter {
	return capabilitypolicy.NewCapabilityRouter(cheap, capable)
}

func WithCapabilityRouter(ctx context.Context, router *CapabilityRouter) context.Context {
	return capabilitypolicy.WithCapabilityRouter(ctx, router)
}

func GetCapabilityRouter(ctx context.Context) *CapabilityRouter {
	return capabilitypolicy.GetCapabilityRouter(ctx)
}
