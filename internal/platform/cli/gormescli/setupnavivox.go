package gormescli

import (
	"github.com/TrebuchetDynamics/gormes-agent/internal/app/setupnavivox"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/network/vpnhost"
)

func SetupNavivoxPairingURI(cfg config.NavivoxCfg) (string, error) {
	return setupnavivox.PairingURI(cfg)
}

func SetupNavivoxPairingQRPath(home string) string {
	return setupnavivox.PairingQRPath(home)
}

func WriteSetupNavivoxPairingQR(path, descriptor string) error {
	return setupnavivox.WritePairingQR(path, descriptor)
}

func SetupNavivoxTerminalQR(descriptor string) (string, error) {
	return setupnavivox.TerminalQR(descriptor)
}

func SetupNavivoxProviderSetupCommand(cfg config.Config, authConfigured bool) string {
	return setupnavivox.ProviderSetupCommand(cfg, authConfigured)
}

func SetupNavivoxBindDefault(current, exposureMode string, hosts []vpnhost.Host) string {
	return setupnavivox.BindDefault(current, exposureMode, hosts)
}

func ParseSetupCSV(value string) []string {
	return setupnavivox.CSV(value)
}
