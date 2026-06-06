package doctor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
)

// TelegramGatewayRuntimeWarning returns the doctor warning for a configured
// Telegram gateway whose live gateway runtime has not activated the Telegram
// platform yet. An empty runtime status means doctor has no live runtime
// evidence and should continue with the normal offline/network check path.
func TelegramGatewayRuntimeWarning(cfg config.TelegramCfg, runtime runtimegateway.RuntimeStatus) (CheckResult, bool) {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return CheckResult{}, false
	}
	if runtime.Kind == "" && runtime.GatewayState == "" && len(runtime.Platforms) == 0 {
		return CheckResult{}, false
	}
	detail := TelegramGatewayStatusDetail(cfg)
	if runtime.GatewayState != "" && runtime.GatewayState != runtimegateway.GatewayStateRunning {
		return CheckResult{
			Name:    "gateway/telegram",
			Status:  StatusWarn,
			Summary: fmt.Sprintf("%s; gateway runtime=%s; run `gormes gateway` or `gormes gateway restart` to activate Telegram", detail, runtime.GatewayState),
		}, true
	}
	if runtime.GatewayState != runtimegateway.GatewayStateRunning {
		return CheckResult{}, false
	}
	platform, ok := runtime.Platforms["telegram"]
	if !ok {
		return CheckResult{
			Name:    "gateway/telegram",
			Status:  StatusWarn,
			Summary: detail + "; live gateway has not registered telegram; run `gormes gateway restart` to load new channel config",
		}, true
	}
	if platform.State == runtimegateway.PlatformStateRunning {
		return CheckResult{}, false
	}
	note := "state=" + string(platform.State)
	if strings.TrimSpace(platform.ErrorMessage) != "" {
		note += " error=" + platform.ErrorMessage
	}
	items := []ItemInfo{{
		Name:   "runtime",
		Status: StatusWarn,
		Note:   note,
	}}
	return CheckResult{
		Name:    "gateway/telegram",
		Status:  StatusWarn,
		Summary: fmt.Sprintf("%s; live gateway telegram lifecycle=%s; run `gormes gateway restart` to activate channel", detail, platform.State),
		Items:   items,
	}, true
}

// TelegramGatewayStatusDetail mirrors the user-visible Telegram gateway status
// detail used by the CLI gateway status command so doctor warnings carry the
// same configuration evidence without importing CLI command modules.
func TelegramGatewayStatusDetail(cfg config.TelegramCfg) string {
	detail := "first_run_discovery=" + strconv.FormatBool(cfg.FirstRunDiscovery)
	if cfg.AllowedChatID != 0 {
		detail = "allowed_chat_id=" + strconv.FormatInt(cfg.AllowedChatID, 10)
	}
	if len(cfg.AllowedUserIDs) > 0 {
		userDetail := "allowed_users=" + strconv.Itoa(len(cfg.AllowedUserIDs))
		if detail == "" {
			return userDetail
		}
		return detail + " " + userDetail
	}
	return detail
}
