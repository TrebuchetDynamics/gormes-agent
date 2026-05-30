package channelruntime

import (
	"log/slog"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/simplex"
	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/textvalue"
)

type SimpleXEnvInfo struct {
	Platform      string
	Enabled       bool
	HomeChannel   string
	AllowedUsers  map[string]bool
	AllowAllUsers bool
}

func SimpleXEnv(lookup func(string) (string, bool)) SimpleXEnvInfo {
	cfg := simplex.ConfigFromEnv(lookup)
	return SimpleXEnvInfo{
		Platform:      simplex.PlatformName,
		Enabled:       cfg.Enabled(),
		HomeChannel:   strings.TrimSpace(cfg.HomeChannel),
		AllowedUsers:  cfg.AllowedUserSet(),
		AllowAllUsers: cfg.AllowAllUsers,
	}
}

func SimpleXStartupAllowlistConfigured(lookupEnv func(string) string) bool {
	info := SimpleXEnv(func(key string) (string, bool) {
		value := lookupEnv(key)
		return value, textvalue.IsNonBlank(value)
	})
	return info.Enabled && textvalue.IsNonBlank(lookupEnv("SIMPLEX_ALLOWED_USERS"))
}

func NewSimpleXGatewayChannel(_ config.Config, log *slog.Logger) (gateway.Channel, error) {
	cfg := simplex.ConfigFromEnv(os.LookupEnv)
	return simplex.NewChannel(cfg, simplex.NewWebSocketTransport(cfg.WSURL), log), nil
}
