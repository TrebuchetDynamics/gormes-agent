package gormescli

import (
	"errors"

	"github.com/spf13/cobra"
)

const gatewayProfileStartupFlagError = "gateway commands are process-scoped and do not accept --profile; configure hosted profiles through setup/profile channel bindings"

// GatewayProfileStartupGuardOptions configures the process-scoped gateway
// profile guard used by the root command's persistent startup pre-run.
type GatewayProfileStartupGuardOptions struct {
	ExitCodeError func(int, error) error
}

// RejectGatewayProfileStartupFlag preserves the Gormes-owned contract that the
// top-level --profile flag selects chat/TUI/setup startup state, while gateway
// commands are process-scoped and must be configured through profile/channel
// bindings instead.
func RejectGatewayProfileStartupFlag(cmd *cobra.Command, opts GatewayProfileStartupGuardOptions) error {
	if !CommandIsGateway(cmd) || !commandFlagChanged(cmd, "profile") {
		return nil
	}
	return gatewayProfileStartupGuardExit(opts, 2, errors.New(gatewayProfileStartupFlagError))
}

// CommandIsGateway reports whether cmd is the gateway command or one of its
// descendants.
func CommandIsGateway(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "gateway" {
			return true
		}
	}
	return false
}

func gatewayProfileStartupGuardExit(opts GatewayProfileStartupGuardOptions, code int, err error) error {
	if opts.ExitCodeError != nil {
		return opts.ExitCodeError(code, err)
	}
	return NewExitCodeError(code, err)
}

func commandFlagChanged(cmd *cobra.Command, name string) bool {
	if cmd == nil {
		return false
	}
	if flags := cmd.Flags(); flags != nil {
		if flag := flags.Lookup(name); flag != nil && flag.Changed {
			return true
		}
	}
	if flags := cmd.PersistentFlags(); flags != nil {
		if flag := flags.Lookup(name); flag != nil && flag.Changed {
			return true
		}
	}
	if flags := cmd.InheritedFlags(); flags != nil {
		if flag := flags.Lookup(name); flag != nil && flag.Changed {
			return true
		}
	}
	if root := cmd.Root(); root != nil && root != cmd {
		if flags := root.Flags(); flags != nil {
			if flag := flags.Lookup(name); flag != nil && flag.Changed {
				return true
			}
		}
		if flags := root.PersistentFlags(); flags != nil {
			if flag := flags.Lookup(name); flag != nil && flag.Changed {
				return true
			}
		}
	}
	return false
}
