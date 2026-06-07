package channels

import (
	"log/slog"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	navivoxcmd "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/channels/navivoxcmd"
)

const NavivoxPlatformName = navivoxcmd.NavivoxPlatformName

func NewNavivoxGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return navivoxcmd.NewNavivoxGatewayChannel(cfg, log)
}

func NewNavivoxPairBridgeHandler(cfg config.NavivoxCfg, inbox chan gateway.InboundEvent) (http.Handler, error) {
	return navivoxcmd.NewNavivoxPairBridgeHandler(cfg, inbox)
}
