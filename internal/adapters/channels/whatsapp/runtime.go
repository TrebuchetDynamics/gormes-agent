package whatsapp

import "github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/whatsapp/runtimeplan"

type RuntimeKind = runtimeplan.RuntimeKind

const (
	RuntimeKindBridge RuntimeKind = runtimeplan.RuntimeKindBridge
	RuntimeKindNative RuntimeKind = runtimeplan.RuntimeKindNative
)

type RuntimePreference = runtimeplan.RuntimePreference

const (
	RuntimePreferenceBridgeFirst RuntimePreference = runtimeplan.RuntimePreferenceBridgeFirst
	RuntimePreferenceNativeFirst RuntimePreference = runtimeplan.RuntimePreferenceNativeFirst
	RuntimePreferenceBridgeOnly  RuntimePreference = runtimeplan.RuntimePreferenceBridgeOnly
	RuntimePreferenceNativeOnly  RuntimePreference = runtimeplan.RuntimePreferenceNativeOnly
)

type AccountMode = runtimeplan.AccountMode

const (
	AccountModeBot      AccountMode = runtimeplan.AccountModeBot
	AccountModeSelfChat AccountMode = runtimeplan.AccountModeSelfChat
)

type RuntimeConfig = runtimeplan.RuntimeConfig

type BridgeRuntimeConfig = runtimeplan.BridgeRuntimeConfig

type NativeRuntimeConfig = runtimeplan.NativeRuntimeConfig

type RuntimePlan = runtimeplan.RuntimePlan

type StartupPlan = runtimeplan.StartupPlan

type SessionPlan = runtimeplan.SessionPlan

type BridgePlan = runtimeplan.BridgePlan

type NativePlan = runtimeplan.NativePlan

type AccountPlan = runtimeplan.AccountPlan

type IdentitySource = runtimeplan.IdentitySource

const (
	IdentitySourceBridgeMessage IdentitySource = runtimeplan.IdentitySourceBridgeMessage
	IdentitySourceNativeSession IdentitySource = runtimeplan.IdentitySourceNativeSession
)

type IdentityPlan = runtimeplan.IdentityPlan

func DecideRuntime(cfg RuntimeConfig) (RuntimePlan, error) {
	return runtimeplan.DecideRuntime(cfg)
}

func identitySourceForRuntime(runtime RuntimeKind) IdentitySource {
	if runtime == RuntimeKindNative {
		return IdentitySourceNativeSession
	}
	return IdentitySourceBridgeMessage
}
