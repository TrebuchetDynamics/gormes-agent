package main

import (
	"errors"

	"github.com/spf13/cobra"

	securitycmd "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/security"
)

func newSecurityCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "security",
		Short:        "Audit gateway, channel, tool, filesystem, and credential security",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(newSecurityAuditCommand())
	return cmd
}

func newSecurityAuditCommand() *cobra.Command {
	var opts securitycmd.AuditOptions
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run a redacted security audit with optional deep probes and safe fixes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := securitycmd.RunAudit(cmd.Context(), cmd.OutOrStdout(), opts, securityBuildProvenance())
			if err != nil {
				return err
			}
			if !result.OK {
				return newExitCodeError(1, errors.New("security audit found failures"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&opts.Deep, "deep", false, "include live gateway probe checks")
	cmd.Flags().BoolVar(&opts.Fix, "fix", false, "apply safe deterministic fixes")
	cmd.Flags().BoolVar(&opts.JSON, "json", false, "print machine-readable JSON")
	return cmd
}

func securityBuildProvenance() securitycmd.BuildProvenance {
	build := newBuildProvenance()
	return securitycmd.BuildProvenance{Version: build.Version, GitCommit: build.GitCommit}
}
