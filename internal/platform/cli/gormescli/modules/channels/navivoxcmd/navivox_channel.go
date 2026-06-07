package navivoxcmd

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/navivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const NavivoxPlatformName = navivox.PlatformName

func NewNavivoxGatewayChannel(cfg config.Config, log *slog.Logger) (gateway.Channel, error) {
	return navivox.NewChannel(cfg.Navivox, log, navivox.WithProfileRouting(cfg.NavivoxProfileRouting()))
}

func NewNavivoxPairBridgeHandler(cfg config.NavivoxCfg, inbox chan gateway.InboundEvent) (http.Handler, error) {
	ch, err := navivox.NewChannel(cfg, nil, navivox.WithSingleUsePairingStream())
	if err != nil {
		return nil, fmt.Errorf("navivox pair: create local bridge: %w", err)
	}
	return ch.Handler(inbox), nil
}
