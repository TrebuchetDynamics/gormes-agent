package gormescli

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/bridge"
)

type BridgeConfig = bridge.Config
type BridgeServer = bridge.Server

func DefaultBridgeConfig() BridgeConfig { return bridge.DefaultConfig() }

func NewBridgeServer(cfg BridgeConfig) *BridgeServer { return bridge.New(cfg) }

func RunBootstrapTermux(ctx context.Context, cfg BridgeConfig, dryRun bool) any {
	return bridge.RunBootstrapTermux(ctx, cfg, dryRun)
}
