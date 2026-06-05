package config

import (
	"fmt"
	"strings"

	displayconfig "github.com/TrebuchetDynamics/gormes-agent/internal/config/display"
)

type GatewayCfg = displayconfig.GatewayCfg
type GatewayPlatformCfg = displayconfig.GatewayPlatformCfg
type DisplayCfg = displayconfig.DisplayCfg
type DisplayPlatformCfg = displayconfig.DisplayPlatformCfg

func normalizeDisplayPlatformKey(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func normalizeGatewayPlatformKey(platform string) string {
	return strings.ToLower(strings.TrimSpace(platform))
}

func normalizeGatewayPlatformMap(in map[string]GatewayPlatformCfg) map[string]GatewayPlatformCfg {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]GatewayPlatformCfg, len(in))
	for platform, cfg := range in {
		key := normalizeGatewayPlatformKey(platform)
		if key == "" {
			continue
		}
		out[key] = cfg
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (c Config) GatewayRestartNotificationEnabled(platform string) bool {
	key := normalizeGatewayPlatformKey(platform)
	if key == "" || len(c.Gateway.Platforms) == 0 {
		return true
	}
	cfg, ok := c.Gateway.Platforms[key]
	if !ok || cfg.GatewayRestartNotification == nil {
		return true
	}
	return *cfg.GatewayRestartNotification
}

func (c Config) GatewayRestartNotifications() map[string]bool {
	if len(c.Gateway.Platforms) == 0 {
		return nil
	}
	out := make(map[string]bool)
	for platform, cfg := range c.Gateway.Platforms {
		if cfg.GatewayRestartNotification == nil {
			continue
		}
		key := normalizeGatewayPlatformKey(platform)
		if key == "" {
			continue
		}
		out[key] = *cfg.GatewayRestartNotification
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeHermesToolProgressMode(raw interface{}) (string, bool) {
	switch v := raw.(type) {
	case nil:
		return "", false
	case bool:
		if v {
			return "all", true
		}
		return "off", true
	case string:
		mode := strings.ToLower(strings.TrimSpace(v))
		if mode == "" {
			return "", false
		}
		switch mode {
		case "off", "new", "all", "verbose":
			return mode, true
		default:
			return "all", true
		}
	default:
		mode := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		if mode == "" {
			return "", false
		}
		switch mode {
		case "off", "new", "all", "verbose":
			return mode, true
		default:
			return "all", true
		}
	}
}
