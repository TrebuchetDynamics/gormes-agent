package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/cli"
)

type updateCommandSeams struct {
	CheckoutDir  func() (string, error)
	RunLifecycle func(context.Context, cli.UpdateLifecycleOptions) cli.UpdateReport
}

func newUpdateCommand() *cobra.Command {
	return newUpdateCommandWithSeams(updateCommandSeams{})
}

func newUpdateCommandWithSeams(seams updateCommandSeams) *cobra.Command {
	var branch string
	var checkOnly bool
	var yes bool
	var restartGateway string
	var killStaleDashboard bool

	if seams.CheckoutDir == nil {
		seams.CheckoutDir = os.Getwd
	}
	if seams.RunLifecycle == nil {
		seams.RunLifecycle = cli.RunUpdateLifecycle
	}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a managed Gormes source checkout",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checkoutDir, err := seams.CheckoutDir()
			if err != nil {
				return err
			}
			report := seams.RunLifecycle(cmd.Context(), cli.UpdateLifecycleOptions{
				CheckoutDir:        checkoutDir,
				Branch:             branch,
				CheckOnly:          checkOnly,
				Yes:                yes,
				RestartGateway:     restartGateway,
				KillStaleDashboard: killStaleDashboard,
				Git:                cli.RealUpdateGitRunner{},
			})
			if report.Branch == "" {
				report.Branch = branch
			}
			if checkOnly && !updateReportHasEvidence(report, cli.UpdateEvidenceCheck) {
				report.Evidence = append([]cli.UpdateEvidence{{Kind: cli.UpdateEvidenceCheck, Detail: "no checkout mutations requested"}}, report.Evidence...)
			}
			printUpdateReport(cmd, report)
			if report.Failed {
				message := "gormes update failed"
				if report.OperatorRecovery != "" {
					message += ": " + report.OperatorRecovery
				}
				return newExitCodeError(1, fmt.Errorf("%s", message))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "main", "Git branch to fetch and fast-forward")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "check update readiness without mutating the checkout")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "assume yes for non-destructive recovery prompts")
	cmd.Flags().StringVar(&restartGateway, "restart-gateway", "auto", "restart policy for a live gateway: auto, always, or never")
	cmd.Flags().BoolVar(&killStaleDashboard, "kill-stale-dashboard", false, "stop stale dashboard processes after a successful update")
	return cmd
}

func updateReportHasEvidence(report cli.UpdateReport, kind cli.UpdateEvidenceKind) bool {
	for _, evidence := range report.Evidence {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

func printUpdateReport(cmd *cobra.Command, report cli.UpdateReport) {
	out := cmd.OutOrStdout()
	branch := report.Branch
	if branch == "" {
		branch = "main"
	}
	fmt.Fprintf(out, "update branch: %s\n", branch)
	if report.PreviousBranch != "" {
		fmt.Fprintf(out, "previous branch: %s\n", report.PreviousBranch)
	}
	for _, evidence := range report.Evidence {
		detail := strings.TrimSpace(evidence.Detail)
		if detail == "" {
			fmt.Fprintf(out, "%s\n", evidence.Kind)
			continue
		}
		fmt.Fprintf(out, "%s\t%s\n", evidence.Kind, detail)
	}
	if report.OperatorRecovery != "" {
		fmt.Fprintln(cmd.ErrOrStderr(), report.OperatorRecovery)
	}
}
