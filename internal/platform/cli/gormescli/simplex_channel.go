package gormescli

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/channelruntime"
)

type SimpleXEnvInfo = channelruntime.SimpleXEnvInfo

func SimpleXEnv(lookup func(string) (string, bool)) SimpleXEnvInfo {
	return channelruntime.SimpleXEnv(lookup)
}

func SimpleXStartupAllowlistConfigured(lookupEnv func(string) string) bool {
	return channelruntime.SimpleXStartupAllowlistConfigured(lookupEnv)
}

func NewSimpleXGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return channelruntime.NewSimpleXGatewayChannel(cfg, log)
}
