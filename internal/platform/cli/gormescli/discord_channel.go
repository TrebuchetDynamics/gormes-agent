package gormescli

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/channelruntime"
)

func CheckDiscordSession(token string) error {
	return channelruntime.CheckDiscordSession(token)
}

func NewDiscordGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return channelruntime.NewDiscordGatewayChannel(cfg, log)
}
