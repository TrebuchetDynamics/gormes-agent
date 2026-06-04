package gormescli

import (
	"context"

	appbridgecmd "github.com/TrebuchetDynamics/gormes-agent/internal/app/bridgecmd"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/bridgeruntime"
)

type BridgeConfig = bridgeruntime.BridgeConfig
type BridgeServer = bridgeruntime.BridgeServer
type BridgeCommandOptions = appbridgecmd.Options
type BridgeCommandServer = appbridgecmd.Server

func DefaultBridgeConfig() BridgeConfig { return bridgeruntime.DefaultBridgeConfig() }

func NewBridgeServer(cfg BridgeConfig) *BridgeServer { return bridgeruntime.NewBridgeServer(cfg) }

func RunBridgeCommand(ctx context.Context, opts BridgeCommandOptions) error {
	return appbridgecmd.Run(ctx, opts)
}

func RunBootstrapTermux(ctx context.Context, cfg BridgeConfig, dryRun bool) any {
	return bridgeruntime.RunBootstrapTermux(ctx, cfg, dryRun)
}
