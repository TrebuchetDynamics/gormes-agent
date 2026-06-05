package gormescli

import "github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/toolruntime"

type DelegationToolOptions = toolruntime.DelegationToolOptions

// RegisterDelegationTool registers the delegate tool when delegation is enabled.
func RegisterDelegationTool(opts DelegationToolOptions) {
	toolruntime.RegisterDelegationTool(opts)
}
