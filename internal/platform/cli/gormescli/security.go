package gormescli

import (
	"context"
	"errors"
	"io"

	"github.com/spf13/cobra"

	appsecurity "github.com/TrebuchetDynamics/gormes-agent/internal/app/security"
	toolspkg "github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type SecurityAuditOptions = appsecurity.AuditOptions
type SecurityBuildProvenance = appsecurity.BuildProvenance

type SecurityCommandOptions struct {
	BuildProvenance func() SecurityBuildProvenance
	ExitCodeError   func(int, error) error
}

func NewSecurityCommand(opts SecurityCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "security",
		Short:        "Audit gateway, channel, tool, filesystem, and credential security",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
	}
	cmd.AddCommand(NewSecurityAuditCommand(opts))
	return cmd
}

func NewSecurityAuditCommand(opts SecurityCommandOptions) *cobra.Command {
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() SecurityBuildProvenance { return SecurityBuildProvenance{} }
	}
	if opts.ExitCodeError == nil {
		opts.ExitCodeError = func(_ int, err error) error { return err }
	}
	var auditOpts SecurityAuditOptions
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Run a redacted security audit with optional deep probes and safe fixes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := RunSecurityAudit(cmd.Context(), cmd.OutOrStdout(), auditOpts, opts.BuildProvenance())
			if err != nil {
				return err
			}
			if !result.OK {
				return opts.ExitCodeError(1, errors.New("security audit found failures"))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&auditOpts.Deep, "deep", false, "include live gateway probe checks")
	cmd.Flags().BoolVar(&auditOpts.Fix, "fix", false, "apply safe deterministic fixes")
	cmd.Flags().BoolVar(&auditOpts.JSON, "json", false, "print machine-readable JSON")
	return cmd
}

func RunSecurityAudit(ctx context.Context, out io.Writer, opts SecurityAuditOptions, build SecurityBuildProvenance) (toolspkg.SecurityAuditResult, error) {
	return appsecurity.RunAudit(ctx, out, opts, build)
}
