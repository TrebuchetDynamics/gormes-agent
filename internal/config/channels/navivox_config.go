package channels

import "github.com/TrebuchetDynamics/gormes-agent/internal/config/channels/navivox"

const (
	NavivoxDefaultBindHost     = navivox.NavivoxDefaultBindHost
	NavivoxDefaultPort         = navivox.NavivoxDefaultPort
	NavivoxDefaultGatewayLabel = navivox.NavivoxDefaultGatewayLabel

	NavivoxExposureLocal     = navivox.NavivoxExposureLocal
	NavivoxExposureTailscale = navivox.NavivoxExposureTailscale
	NavivoxExposureWireGuard = navivox.NavivoxExposureWireGuard
	NavivoxExposureVPN       = navivox.NavivoxExposureVPN
	NavivoxExposurePublic    = navivox.NavivoxExposurePublic

	NavivoxAuthPairingToken              = navivox.NavivoxAuthPairingToken
	NavivoxAuthStaticToken               = navivox.NavivoxAuthStaticToken
	NavivoxAuthTailscaleIdentity         = navivox.NavivoxAuthTailscaleIdentity
	NavivoxAuthTokenAndTailscaleIdentity = navivox.NavivoxAuthTokenAndTailscaleIdentity

	NavivoxMinExposedTokenLength        = navivox.NavivoxMinExposedTokenLength
	NavivoxMinExposedTokenDistinctChars = navivox.NavivoxMinExposedTokenDistinctChars
)

// NewNavivoxGatewayID creates the opaque public Gormes Gateway identity used
// by Navivox pairing/reconnect metadata. It is intentionally random and never
// derived from tokens, URLs, machine names, usernames, or paths.
func NewNavivoxGatewayID() (string, error) {
	return navivox.NewNavivoxGatewayID()
}

// NavivoxCfg configures the native gateway-owned HTTP/WebSocket channel used
// by the Flutter Navivox app. The disabled zero value is intentionally safe.
type NavivoxCfg struct {
	Enabled                  bool                        `toml:"enabled" yaml:"enabled"`
	GatewayID                string                      `toml:"gateway_id" yaml:"gateway_id"`
	GatewayLabel             string                      `toml:"gateway_label" yaml:"gateway_label"`
	BindHost                 string                      `toml:"bind_host" yaml:"bind_host"`
	Port                     int                         `toml:"port" yaml:"port"`
	ExposureMode             string                      `toml:"exposure_mode" yaml:"exposure_mode"`
	AuthMode                 string                      `toml:"auth_mode" yaml:"auth_mode"`
	Token                    string                      `toml:"token" yaml:"token"`
	AllowOrigins             []string                    `toml:"allow_origins" yaml:"allow_origins"`
	AllowedTailnetIdentities []string                    `toml:"allowed_tailnet_identities" yaml:"allowed_tailnet_identities"`
	PublicConfirmed          bool                        `toml:"public_confirmed" yaml:"public_confirmed"`
	Servers                  map[string]NavivoxServerCfg `toml:"servers" yaml:"servers"`
}

type NavivoxServerCfg struct {
	Enabled      bool     `toml:"enabled" yaml:"enabled"`
	Bind         string   `toml:"bind" yaml:"bind"`
	Profiles     []string `toml:"profiles" yaml:"profiles"`
	Transports   []string `toml:"transports" yaml:"transports"`
	Capabilities []string `toml:"capabilities" yaml:"capabilities"`
}

func NormalizeNavivoxConfig(cfg *NavivoxCfg) error {
	impl := cfg.asNavivoxCfg()
	if err := navivox.NormalizeNavivoxConfig(&impl); err != nil {
		return err
	}
	cfg.applyNavivoxCfg(impl)
	return nil
}

func ValidateNavivoxForRuntime(cfg *NavivoxCfg) error {
	return NormalizeNavivoxConfig(cfg)
}

// NavivoxExposureRequiresVPN reports whether the given exposure_mode value
// requires bind_host to match an active VPN interface IP.
func NavivoxExposureRequiresVPN(mode string) bool {
	return navivox.NavivoxExposureRequiresVPN(mode)
}

// ValidateNavivoxBindAgainstVPN returns nil when navivox.bind_host either is
// not required to be a VPN interface IP (exposure_mode local/public, or
// channel disabled) or matches one of the live VPN IPs supplied by the
// caller. The list is supplied as plain strings so config has no dependency
// on the network/vpnhost package.
func ValidateNavivoxBindAgainstVPN(cfg *NavivoxCfg, vpnIPs []string) error {
	if cfg == nil {
		return navivox.ValidateNavivoxBindAgainstVPN(nil, vpnIPs)
	}
	impl := cfg.asNavivoxCfg()
	return navivox.ValidateNavivoxBindAgainstVPN(&impl, vpnIPs)
}

func (c NavivoxCfg) asNavivoxCfg() navivox.NavivoxCfg {
	servers := make(map[string]navivox.NavivoxServerCfg, len(c.Servers))
	for id, server := range c.Servers {
		servers[id] = navivox.NavivoxServerCfg(server)
	}
	if len(servers) == 0 {
		servers = nil
	}
	return navivox.NavivoxCfg{
		Enabled:                  c.Enabled,
		GatewayID:                c.GatewayID,
		GatewayLabel:             c.GatewayLabel,
		BindHost:                 c.BindHost,
		Port:                     c.Port,
		ExposureMode:             c.ExposureMode,
		AuthMode:                 c.AuthMode,
		Token:                    c.Token,
		AllowOrigins:             c.AllowOrigins,
		AllowedTailnetIdentities: c.AllowedTailnetIdentities,
		PublicConfirmed:          c.PublicConfirmed,
		Servers:                  servers,
	}
}

func (c *NavivoxCfg) applyNavivoxCfg(impl navivox.NavivoxCfg) {
	servers := make(map[string]NavivoxServerCfg, len(impl.Servers))
	for id, server := range impl.Servers {
		servers[id] = NavivoxServerCfg(server)
	}
	if len(servers) == 0 {
		servers = nil
	}
	*c = NavivoxCfg{
		Enabled:                  impl.Enabled,
		GatewayID:                impl.GatewayID,
		GatewayLabel:             impl.GatewayLabel,
		BindHost:                 impl.BindHost,
		Port:                     impl.Port,
		ExposureMode:             impl.ExposureMode,
		AuthMode:                 impl.AuthMode,
		Token:                    impl.Token,
		AllowOrigins:             impl.AllowOrigins,
		AllowedTailnetIdentities: impl.AllowedTailnetIdentities,
		PublicConfirmed:          impl.PublicConfirmed,
		Servers:                  servers,
	}
}
