package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

type hermesUnavailableCommandSpec struct {
	Use         string
	Aliases     []string
	Short       string
	Row         string
	Destructive bool
	FlagSet     func(*cobra.Command)
}

type hermesUnavailableReportJSON struct {
	Build       buildProvenanceJSON `json:"build"`
	Action      string              `json:"action"`
	Command     string              `json:"command"`
	Status      string              `json:"status"`
	Row         string              `json:"row"`
	Destructive bool                `json:"destructive,omitempty"`
	Error       string              `json:"error"`
}

func newHermesUnavailableCommand(spec hermesUnavailableCommandSpec, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:          spec.Use,
		Aliases:      spec.Aliases,
		Short:        spec.Short,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runHermesUnavailableCommand(cmd, spec)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON for the row-backed unavailable command")
	if spec.FlagSet != nil {
		spec.FlagSet(cmd)
	}
	cmd.AddCommand(children...)
	return cmd
}

func newHermesUnavailableParent(use, short string, children ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	cmd.AddCommand(children...)
	return cmd
}

func runHermesUnavailableCommand(cmd *cobra.Command, spec hermesUnavailableCommandSpec) error {
	row := spec.Row
	if row == "" {
		row = "Hermes CLI parity"
	}
	command := cmd.CommandPath()
	message := command + " is classified in the Hermes CLI parity manifest but is still row-backed in Gormes"
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		body, err := json.MarshalIndent(hermesUnavailableReportJSON{
			Build:       newBuildProvenance(),
			Action:      "hermes_command_unavailable",
			Command:     command,
			Status:      string(hermesCLIRowBacked),
			Row:         row,
			Destructive: spec.Destructive,
			Error:       message,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(body))
	}
	return newExitCodeError(2, fmt.Errorf("%s", message))
}

func hermesUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}
