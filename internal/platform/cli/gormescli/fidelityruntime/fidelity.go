package fidelityruntime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/progress/fidelity"
)

type FidelityCommandOptions struct {
	Build         func() any
	ExitCodeError func(code int, err error) error
}

type fidelityHermesCommandOptions struct {
	repoRoot    string
	progress    string
	hermes      string
	sourcePairs string
	hermesSHA   string
	strict      bool
	json        bool
}

type fidelityHermesReportJSON struct {
	Build  any             `json:"build"`
	Action string          `json:"action"`
	OK     bool            `json:"ok"`
	Report fidelity.Report `json:"report,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func NewFidelityCommand(options FidelityCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fidelity",
		Short: "Inspect Hermes/Gormes parity evidence",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newFidelityHermesCommand(options))
	return cmd
}

func newFidelityHermesCommand(options FidelityCommandOptions) *cobra.Command {
	var opts fidelityHermesCommandOptions
	cmd := &cobra.Command{
		Use:   "hermes",
		Short: "Report static Hermes/Gormes fidelity coverage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFidelityHermesCommand(cmd, opts, options)
		},
	}
	cmd.Flags().StringVar(&opts.repoRoot, "repo-root", ".", "repository root containing progress and source-pair evidence")
	cmd.Flags().StringVar(&opts.progress, "progress", "", "path to progress.json")
	cmd.Flags().StringVar(&opts.hermes, "hermes", "", "path to Hermes source checkout")
	cmd.Flags().StringVar(&opts.sourcePairs, "source-pairs", "", "path to hermes-source-pairs.json")
	cmd.Flags().StringVar(&opts.hermesSHA, "hermes-sha", "", "Hermes source SHA to record in the report")
	cmd.Flags().BoolVar(&opts.strict, "strict", false, "return non-zero when critical fidelity surfaces or unmapped upstream files are not covered")
	cmd.Flags().BoolVar(&opts.json, "json", false, "emit machine-readable JSON")
	return cmd
}

func runFidelityHermesCommand(cmd *cobra.Command, opts fidelityHermesCommandOptions, options FidelityCommandOptions) error {
	report, err := fidelity.GenerateHermesReport(cmd.Context(), fidelity.Options{
		RepoRoot:        opts.repoRoot,
		ProgressPath:    opts.progress,
		HermesPath:      opts.hermes,
		SourcePairsPath: opts.sourcePairs,
		HermesSHA:       opts.hermesSHA,
		Strict:          opts.strict,
	})
	if err != nil {
		if opts.json {
			enc := json.NewEncoder(cmd.OutOrStdout())
			_ = enc.Encode(fidelityHermesReportJSON{
				Build:  commandBuild(options),
				Action: "fidelity_hermes",
				OK:     false,
				Error:  err.Error(),
			})
		}
		return err
	}
	payload := fidelityHermesReportJSON{
		Build:  commandBuild(options),
		Action: "fidelity_hermes",
		OK:     report.OK,
		Report: report,
	}
	if opts.json {
		enc := json.NewEncoder(cmd.OutOrStdout())
		if err := enc.Encode(payload); err != nil {
			return err
		}
	} else {
		fmt.Fprint(cmd.OutOrStdout(), formatFidelityHermesReport(payload))
	}
	if opts.strict && !report.OK {
		return commandExitError(options, 1, fmt.Errorf("fidelity hermes: uncovered critical surfaces or unmapped upstream files"))
	}
	return nil
}

func commandBuild(options FidelityCommandOptions) any {
	if options.Build == nil {
		return map[string]string{}
	}
	return options.Build()
}

func commandExitError(options FidelityCommandOptions, code int, err error) error {
	if options.ExitCodeError == nil {
		return err
	}
	return options.ExitCodeError(code, err)
}

func formatFidelityHermesReport(payload fidelityHermesReportJSON) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Hermes fidelity report\n")
	fmt.Fprintf(&b, "status: %t\n", payload.OK)
	fmt.Fprintf(&b, "hermes_sha: %s\n", payload.Report.HermesSHA)
	fmt.Fprintf(&b, "surfaces: total=%d covered=%d partial=%d planned=%d missing=%d\n",
		payload.Report.Summary.Total,
		payload.Report.Summary.ByStatus[string(fidelity.StatusCovered)],
		payload.Report.Summary.ByStatus[string(fidelity.StatusPartial)],
		payload.Report.Summary.ByStatus[string(fidelity.StatusPlanned)],
		payload.Report.Summary.ByStatus[string(fidelity.StatusMissing)],
	)
	fmt.Fprintf(&b, "unmapped_upstream: source=%d docs=%d tests=%d\n",
		len(payload.Report.UnmappedUpstream.SourceFiles),
		len(payload.Report.UnmappedUpstream.DocsFiles),
		len(payload.Report.UnmappedUpstream.TestFiles),
	)
	for _, surface := range payload.Report.Surfaces {
		fmt.Fprintf(&b, "- %s status=%s\n", surface.ID, surface.Status)
	}
	return b.String()
}
