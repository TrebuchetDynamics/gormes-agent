package gateway

import (
	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli/modules/gateway/live"
)

// NewGatewayCommand returns a fresh gateway cobra command with the given
// Run function (injected from the root package) and standard Options.
func NewGatewayCommand(run func(*cobra.Command, []string) error, opts Options) *cobra.Command {
	seams := live.GatewayCommandSeams{
		Run: run,
		StopCommand: func() *cobra.Command {
			return NewStopCommand(opts)
		},
		RestartCommand: func() *cobra.Command {
			return NewRestartCommand(opts)
		},
		ReloadCommand: func() *cobra.Command {
			return NewReloadCommand(opts)
		},
		StatusCommand: func() *cobra.Command {
			return NewStatusCommand(opts)
		},
		FleetCommand: func() *cobra.Command {
			return NewFleetCommand(opts)
		},
		DiscoverCommand: func() *cobra.Command {
			return NewDiscoverCommand(opts)
		},
		ProbeCommand: func() *cobra.Command {
			return NewProbeCommand(opts)
		},
		UsageCostCommand: func() *cobra.Command {
			return NewUsageCostCommand(opts)
		},
		MutatingUnavailableCommand: func(name string) *cobra.Command {
			return NewMutatingUnavailableCommand(name, opts)
		},
		BootInstallCommand:   NewBootInstallCommand,
		BootUninstallCommand: NewBootUninstallCommand,
	}
	cmd := live.NewGatewayCommandWithSeams(seams, opts)
	cmd.Flags().Bool("no-wakelock", false, "skip automatic termux-wake-lock acquisition on Termux (gateway foreground mode only)")
	return cmd
}

// NewGatewayCommandOptions builds gateway Options from version metadata
// injected from the root package.
func NewGatewayCommandOptions(
	buildVersion func() string,
	buildCommit func() string,
	exitError func(int, error) error,
) Options {
	return Options{
		BuildProvenance: func() gormescli.BuildProvenance {
			return gormescli.BuildProvenance{
				Version:   buildVersion(),
				GitCommit: buildCommit(),
			}
		},
		ExitError:                    exitError,
		TermuxDetected:               TermuxDetected,
		TermuxLifecycleGuidanceLine:  TermuxLifecycleGuidanceLine,
		TermuxLifecycleGuidanceError: TermuxLifecycleGuidanceError,
		TermuxNotificationStatus:     TermuxNotificationStatusLine,
	}
}