package channels

import wa "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/whatsapp"

type WhatsAppRuntimePlan = wa.RuntimePlan
type WhatsAppRuntimeConfig = wa.RuntimeConfig
type WhatsAppBridgeRuntimeConfig = wa.BridgeRuntimeConfig
type WhatsAppAccountMode = wa.AccountMode

const (
	WhatsAppAccountModeBot      = wa.AccountModeBot
	WhatsAppAccountModeSelfChat = wa.AccountModeSelfChat
)

func DecideWhatsAppRuntime(cfg WhatsAppRuntimeConfig) (WhatsAppRuntimePlan, error) {
	return wa.DecideRuntime(cfg)
}
