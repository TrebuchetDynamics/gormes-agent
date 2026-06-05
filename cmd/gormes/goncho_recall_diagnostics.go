package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/goncho/service"
)

func newGonchoRecallDiagnosticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall-diagnostics --trace <trace.json>",
		Short: "Explain a durable Goncho RecallTrace ranking decision",
		Args:  cobra.NoArgs,
		RunE:  runGonchoRecallDiagnostics,
	}
	cmd.Flags().String("trace", "", "path to a RecallTrace JSON file")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

func runGonchoRecallDiagnostics(cmd *cobra.Command, _ []string) error {
	tracePath, _ := cmd.Flags().GetString("trace")
	emitJSON, _ := cmd.Flags().GetBool("json")
	tracePath = strings.TrimSpace(tracePath)
	if tracePath == "" {
		return newExitCodeError(1, errors.New("goncho recall-diagnostics: --trace is required"))
	}
	f, err := os.Open(tracePath)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("goncho recall-diagnostics: open trace: %w", err))
	}
	defer f.Close()

	trace, err := goncho.DecodeRecallTraceJSON(f)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("goncho recall-diagnostics: decode trace: %w", err))
	}
	report := goncho.BuildRecallDiagnostics(trace)
	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		if err := enc.Encode(report); err != nil {
			return err
		}
		return nil
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), goncho.FormatRecallDiagnosticsReport(report))
	return err
}
