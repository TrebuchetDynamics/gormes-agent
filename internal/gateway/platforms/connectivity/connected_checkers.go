package connectivity

import (
	"sort"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/identity"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/platforms/inventory"
)

// PlatformConnectionConfig is the minimal redacted config shape needed to
// decide whether a platform should appear as configured in status/readouts.
type PlatformConnectionConfig struct {
	ID      string
	Enabled bool
	Token   string
	APIKey  string
	Extra   map[string]string
}

type platformConnectedChecker func(PlatformConnectionConfig) bool

var genericTokenConnectedPlatforms = map[string]struct{}{
	"telegram":      {},
	"discord":       {},
	"slack":         {},
	"matrix":        {},
	"mattermost":    {},
	"homeassistant": {},
}

var platformConnectedCheckers = map[string]platformConnectedChecker{
	"weixin":                extraAll("account_id", "token"),
	"signal":                extraAny("http_url"),
	"email":                 extraAny("address"),
	"sms":                   extraAny("twilio_account_sid", "account_sid"),
	"api_server":            enabledOnly,
	"webhook":               enabledOnly,
	"msgraph_webhook":       extraAny("client_state"),
	"whatsapp":              enabledOnly,
	"feishu":                extraAny("app_id"),
	"feishu_meeting_invite": extraAny("app_id"),
	"google_chat":           extraAll("project_id", "subscription_name"),
	"irc":                   extraAll("server", "channel"),
	"line":                  extraAll("channel_access_token", "channel_secret"),
	"ntfy":                  extraAny("topic"),
	"simplex":               extraAny("ws_url"),
	"wecom":                 extraAny("bot_id"),
	"wecom_callback":        extraAny("corp_id"),
	"bluebubbles":           extraAll("server_url", "password"),
	"qqbot":                 extraAll("app_id", "client_secret"),
	"yuanbao":               extraAll("app_id", "app_secret"),
	"dingtalk":              extraAll("client_id", "client_secret"),
	"teams":                 extraAll("client_id", "client_secret", "tenant_id"),
}

func PlatformLooksConfigured(cfg PlatformConnectionConfig) (bool, bool) {
	id := identity.NormalizePlatformID(cfg.ID)
	if id == "" {
		return false, false
	}
	if _, ok := genericTokenConnectedPlatforms[id]; ok {
		return cfg.Enabled && (cfg.Token != "" || cfg.APIKey != ""), true
	}
	checker, ok := platformConnectedCheckers[id]
	if !ok {
		return false, false
	}
	return checker(cfg), true
}

func PlatformConnectedCheckerIDs() []string {
	ids := make([]string, 0, len(genericTokenConnectedPlatforms)+len(platformConnectedCheckers))
	for id := range genericTokenConnectedPlatforms {
		ids = append(ids, id)
	}
	for id := range platformConnectedCheckers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func MissingPlatformConnectedCheckers(manifest []inventory.PlatformManifestEntry) []string {
	var missing []string
	for _, entry := range manifest {
		id := identity.NormalizePlatformID(entry.ID)
		if id == "" || entry.Kind == inventory.PlatformKindLocal {
			continue
		}
		if _, ok := PlatformLooksConfigured(PlatformConnectionConfig{ID: id, Enabled: true}); !ok {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	return missing
}

func enabledOnly(cfg PlatformConnectionConfig) bool {
	return cfg.Enabled
}

func extraAny(keys ...string) platformConnectedChecker {
	return func(cfg PlatformConnectionConfig) bool {
		if !cfg.Enabled {
			return false
		}
		for _, key := range keys {
			if cfg.Extra[key] != "" {
				return true
			}
		}
		return false
	}
}

func extraAll(keys ...string) platformConnectedChecker {
	return func(cfg PlatformConnectionConfig) bool {
		if !cfg.Enabled {
			return false
		}
		for _, key := range keys {
			if cfg.Extra[key] == "" {
				return false
			}
		}
		return true
	}
}
