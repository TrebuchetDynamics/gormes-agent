package gormescli

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/channelruntime"
)

// NewSlackGatewayChannel binds gateway Slack config to the live Slack channel.
func NewSlackGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return channelruntime.NewSlackGatewayChannel(cfg, log)
}
