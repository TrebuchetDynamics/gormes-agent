package gateway

import (
	"context"

	gatewayboothook "github.com/TrebuchetDynamics/gormes-agent/internal/gateway/boothook"
)

// BootHookConfig configures the built-in BOOT.md startup hook.
type BootHookConfig = gatewayboothook.Config

// StartBootHook starts a background BOOT.md run when the file exists and is
// non-empty. It returns false when there is nothing to do.
func StartBootHook(ctx context.Context, cfg BootHookConfig) bool {
	return gatewayboothook.Start(ctx, cfg)
}
