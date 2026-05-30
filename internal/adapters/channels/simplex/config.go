// Package simplex provides a native, fakeable SimpleX Chat gateway channel.
package simplex

import (
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/internal/channelutil"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

const (
	PlatformName     = "simplex"
	DisplayName      = "SimpleX Chat"
	MaxMessageLength = 16000
)

// Config mirrors Hermes' SimpleX platform-plugin env surface without requiring
// the Python plugin or a live simplex-chat daemon during tests.
type Config struct {
	WSURL           string
	AllowedUsers    []string
	AllowAllUsers   bool
	HomeChannel     string
	HomeChannelName string
}

// ConfigFromEnv reads the Hermes-compatible SIMPLEX_* env keys. lookup may be
// nil, in which case os.LookupEnv is used.
func ConfigFromEnv(lookup func(string) (string, bool)) Config {
	if lookup == nil {
		lookup = os.LookupEnv
	}
	value := func(key string) string {
		if v, ok := lookup(key); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	return Config{
		WSURL:           value("SIMPLEX_WS_URL"),
		AllowedUsers:    parseCSV(value("SIMPLEX_ALLOWED_USERS")),
		AllowAllUsers:   parseBool(value("SIMPLEX_ALLOW_ALL_USERS")),
		HomeChannel:     value("SIMPLEX_HOME_CHANNEL"),
		HomeChannelName: value("SIMPLEX_HOME_CHANNEL_NAME"),
	}
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.WSURL) != ""
}

func (c Config) MissingConfig() []string {
	if c.Enabled() {
		return nil
	}
	return []string{"ws_url"}
}

func (c Config) AllowedUserSet() map[string]bool { return channelutil.BoolSet(c.AllowedUsers) }

func (c Config) HomeDeliveryTarget() gateway.DeliveryTarget {
	chatID := strings.TrimSpace(c.HomeChannel)
	if chatID == "" {
		return gateway.DeliveryTarget{Platform: PlatformName}
	}
	return gateway.DeliveryTarget{Platform: PlatformName, ChatID: chatID}
}

func (c Config) RedactedStatus() string {
	parts := []string{PlatformName}
	if missing := c.MissingConfig(); len(missing) > 0 {
		parts = append(parts, "missing_config="+strings.Join(missing, ","))
	} else {
		parts = append(parts, "configured", "ws_url="+redactEvidence(c.WSURL))
	}
	if len(c.AllowedUserSet()) > 0 {
		parts = append(parts, fmt.Sprintf("allowed_users=%d", len(c.AllowedUserSet())))
	}
	if c.AllowAllUsers {
		parts = append(parts, "allow_all_users=true")
	}
	if home := strings.TrimSpace(c.HomeChannel); home != "" {
		parts = append(parts, "home_channel="+home)
	}
	return strings.Join(parts, " ")
}

type PluginInfo struct {
	Name             string
	Label            string
	RequiredEnv      []string
	AllowedUsersEnv  string
	AllowAllEnv      string
	HomeChannelEnv   string
	InstallHint      string
	MaxMessageLength int
	SetupAvailable   bool
	PlatformHint     string
}

func PluginMetadata() PluginInfo {
	return PluginInfo{
		Name:             PlatformName,
		Label:            DisplayName,
		RequiredEnv:      []string{"SIMPLEX_WS_URL"},
		AllowedUsersEnv:  "SIMPLEX_ALLOWED_USERS",
		AllowAllEnv:      "SIMPLEX_ALLOW_ALL_USERS",
		HomeChannelEnv:   "SIMPLEX_HOME_CHANNEL",
		InstallHint:      "simplex-chat -p 5225   # exposes ws://127.0.0.1:5225",
		MaxMessageLength: MaxMessageLength,
		SetupAvailable:   true,
		PlatformHint:     "You are chatting via SimpleX Chat, a private decentralised messenger with opaque contact IDs and no typing indicator.",
	}
}

func parseCSV(value string) []string { return channelutil.UniqueCommaList(value) }

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "t", "true", "y", "yes", "on":
		return true
	default:
		return false
	}
}
