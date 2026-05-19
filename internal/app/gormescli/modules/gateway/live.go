package gateway

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/app/gormescli"
)

var MutatingUnavailableSubcommands = []string{
	"start",
	"install",
	"uninstall",
}

var RowBackedUnavailableSubcommands = []string{
	"run",
	"setup",
	"migrate-legacy",
	"list",
}

type GatewayCommandSeams struct {
	Run                        func(*cobra.Command, []string) error
	StopCommand                func() *cobra.Command
	RestartCommand             func() *cobra.Command
	ReloadCommand              func() *cobra.Command
	StatusCommand              func() *cobra.Command
	DiscoverCommand            func() *cobra.Command
	ProbeCommand               func() *cobra.Command
	UsageCostCommand           func() *cobra.Command
	MutatingUnavailableCommand func(name string) *cobra.Command
	RowBackedCommand           func(name string, opts Options) *cobra.Command
	BootInstallCommand         func() *cobra.Command
	BootUninstallCommand       func() *cobra.Command
}

func NewGatewayCommandWithSeams(seams GatewayCommandSeams, opts Options) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:          "gateway",
		Short:        "Run Gormes as a multi-channel messaging gateway",
		Long:         "Runs every configured channel through one gateway.Manager that drives the same kernel + tool loop as the TUI.",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE:         seams.Run,
	}
	cmd.AddCommand(
		seams.StopCommand(),
		seams.RestartCommand(),
		seams.ReloadCommand(),
		seams.StatusCommand(),
		seams.DiscoverCommand(),
		seams.ProbeCommand(),
		seams.UsageCostCommand(),
	)
	for _, name := range MutatingUnavailableSubcommands {
		cmd.AddCommand(seams.MutatingUnavailableCommand(name))
	}
	for _, name := range RowBackedUnavailableSubcommands {
		cmd.AddCommand(seams.RowBackedCommand(name, opts))
	}
	if seams.BootInstallCommand != nil {
		cmd.AddCommand(seams.BootInstallCommand())
	}
	if seams.BootUninstallCommand != nil {
		cmd.AddCommand(seams.BootUninstallCommand())
	}
	return cmd
}

func (s GatewayCommandSeams) withDefaults() GatewayCommandSeams {
	if s.Run == nil {
		s.Run = func(*cobra.Command, []string) error { return fmt.Errorf("gateway run seam is not configured") }
	}
	if s.StopCommand == nil {
		s.StopCommand = unavailableChild("stop")
	}
	if s.RestartCommand == nil {
		s.RestartCommand = unavailableChild("restart")
	}
	if s.ReloadCommand == nil {
		s.ReloadCommand = unavailableChild("reload")
	}
	if s.StatusCommand == nil {
		s.StatusCommand = unavailableChild("status")
	}
	if s.DiscoverCommand == nil {
		s.DiscoverCommand = unavailableChild("discover")
	}
	if s.ProbeCommand == nil {
		s.ProbeCommand = unavailableChild("probe")
	}
	if s.UsageCostCommand == nil {
		s.UsageCostCommand = unavailableChild("usage-cost")
	}
	if s.MutatingUnavailableCommand == nil {
		s.MutatingUnavailableCommand = func(name string) *cobra.Command {
			return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error {
				return fmt.Errorf("gateway %s seam is not configured", name)
			}}
		}
	}
	if s.RowBackedCommand == nil {
		s.RowBackedCommand = func(name string, opts Options) *cobra.Command {
			return gormescli.NewRowBackedCommand(gormescli.RowBackedCommandSpec{
				Use:   name,
				Short: fmt.Sprintf("Manage gateway %s through a row-backed Hermes parity command", name),
				Row:   GatewayCronRow,
			}, gormescli.RowBackedCommandOptions{BuildProvenance: opts.BuildProvenance})
		}
	}
	return s
}

func unavailableChild(name string) func() *cobra.Command {
	return func() *cobra.Command {
		return &cobra.Command{Use: name, RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("gateway %s seam is not configured", name)
		}}
	}
}
