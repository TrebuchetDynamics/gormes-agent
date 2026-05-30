package gormescli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/bridgeruntime"
)

type BridgeConfig = bridgeruntime.BridgeConfig
type BridgeServer = bridgeruntime.BridgeServer

func DefaultBridgeConfig() BridgeConfig { return bridgeruntime.DefaultBridgeConfig() }

func NewBridgeServer(cfg BridgeConfig) *BridgeServer { return bridgeruntime.NewBridgeServer(cfg) }

func RunBootstrapTermux(ctx context.Context, cfg BridgeConfig, dryRun bool) any {
	return bridgeruntime.RunBootstrapTermux(ctx, cfg, dryRun)
}
