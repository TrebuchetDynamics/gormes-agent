package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupnavivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

func SetupNavivoxPairingURI(cfg config.NavivoxCfg) (string, error) {
	return setupnavivox.PairingURI(cfg)
}

func SetupNavivoxBindDefault(current, exposureMode string, hosts []vpnhost.Host) string {
	return setupnavivox.BindDefault(current, exposureMode, hosts)
}

func ParseSetupCSV(value string) []string {
	return setupnavivox.CSV(value)
}
