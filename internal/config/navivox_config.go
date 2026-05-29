package config

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels"

const (
	NavivoxDefaultBindHost = channels.NavivoxDefaultBindHost
	NavivoxDefaultPort     = channels.NavivoxDefaultPort

	NavivoxExposureLocal     = channels.NavivoxExposureLocal
	NavivoxExposureTailscale = channels.NavivoxExposureTailscale
	NavivoxExposureWireGuard = channels.NavivoxExposureWireGuard
	NavivoxExposureVPN       = channels.NavivoxExposureVPN
	NavivoxExposurePublic    = channels.NavivoxExposurePublic

	NavivoxAuthPairingToken              = channels.NavivoxAuthPairingToken
	NavivoxAuthStaticToken               = channels.NavivoxAuthStaticToken
	NavivoxAuthTailscaleIdentity         = channels.NavivoxAuthTailscaleIdentity
	NavivoxAuthTokenAndTailscaleIdentity = channels.NavivoxAuthTokenAndTailscaleIdentity
)

type NavivoxCfg = channels.NavivoxCfg
type NavivoxServerCfg = channels.NavivoxServerCfg

func normalizeNavivoxConfig(cfg *NavivoxCfg) error {
	return channels.NormalizeNavivoxConfig(cfg)
}

func ValidateNavivoxForRuntime(cfg *NavivoxCfg) error {
	return channels.ValidateNavivoxForRuntime(cfg)
}

func NavivoxExposureRequiresVPN(mode string) bool {
	return channels.NavivoxExposureRequiresVPN(mode)
}

func ValidateNavivoxBindAgainstVPN(cfg *NavivoxCfg, vpnIPs []string) error {
	return channels.ValidateNavivoxBindAgainstVPN(cfg, vpnIPs)
}
