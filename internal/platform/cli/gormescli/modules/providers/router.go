package providers

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	routerpkg "github.com/TrebuchetDynamics/gormes-agent/internal/provider/router"
)

const (
	RouterDefaultListen    = routerpkg.DefaultListen
	RouterDefaultSetupMode = routerpkg.DefaultSetupMode
	RouterDefaultTransport = routerpkg.DefaultTransport
)

func ValidateRouterNoRecursion(cfg config.RouterCfg) error {
	return routerpkg.ValidateNoRecursion(cfg)
}
