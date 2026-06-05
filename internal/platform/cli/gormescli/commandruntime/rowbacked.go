package commandruntime

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// RowBackedCommandSpec describes a CLI surface that is intentionally present
// but still backed by a future progress row.
type RowBackedCommandSpec struct {
	Use         string
	Aliases     []string
	Short       string
	Row         string
	Destructive bool
	FlagSet     func(*cobra.Command)
}

// RowBackedCommandOptions carries binary-owned build values into importable
// row-backed command modules.
type RowBackedCommandOptions struct {
	BuildProvenance func() BuildProvenance
}

func (o RowBackedCommandOptions) buildProvenance() BuildProvenance {
	if o.BuildProvenance == nil {
		return BuildProvenance{}
	}
	return o.BuildProvenance()
}

// RowBackedReportJSON is the machine-readable report emitted by row-backed
// compatibility commands.
type RowBackedReportJSON struct {
	Build       BuildProvenance `json:"build"`
	Action      string          `json:"action"`
	Command     string          `json:"command"`
	Status      string          `json:"status"`
	Row         string          `json:"row"`
	Destructive bool            `json:"destructive,omitempty"`
	Error       string          `json:"error"`
}

// NewRowBackedCommand creates a command that is visible in help and emits a
// structured unavailable report with exit code 2 when invoked.
func NewRowBackedCommand(spec RowBackedCommandSpec, opts RowBackedCommandOptions, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:          spec.Use,
		Aliases:      spec.Aliases,
		Short:        spec.Short,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return RunRowBackedCommand(cmd, spec, opts)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON for the row-backed unavailable command")
	if spec.FlagSet != nil {
		spec.FlagSet(cmd)
	}
	cmd.AddCommand(children...)
	return cmd
}

// RunRowBackedCommand emits row-backed evidence and returns exit code 2.
func RunRowBackedCommand(cmd *cobra.Command, spec RowBackedCommandSpec, opts RowBackedCommandOptions) error {
	row := spec.Row
	if row == "" {
		row = "Hermes CLI parity"
	}
	command := cmd.CommandPath()
	message := command + " is classified in the Hermes CLI parity manifest but is still row-backed in Gormes"
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, err := json.MarshalIndent(RowBackedReportJSON{
			Build:       opts.buildProvenance(),
			Action:      "hermes_command_unavailable",
			Command:     command,
			Status:      RowBackedStatus,
			Row:         row,
			Destructive: spec.Destructive,
			Error:       message,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	}
	return NewExitCodeError(2, fmt.Errorf("%s", message))
}
