package goncho

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/goncho/service"
)

// RunGonchoRecallDiagnostics executes the goncho recall-diagnostics command.
func RunGonchoRecallDiagnostics(cmd *cobra.Command, _ []string) error {
	tracePath, _ := cmd.Flags().GetString("trace")
	emitJSON, _ := cmd.Flags().GetBool("json")
	tracePath = strings.TrimSpace(tracePath)
	if tracePath == "" {
		return exitCodeError(1, errors.New("goncho recall-diagnostics: --trace is required"))
	}
	f, err := os.Open(tracePath)
	if err != nil {
		return exitCodeError(1, fmt.Errorf("goncho recall-diagnostics: open trace: %w", err))
	}
	defer f.Close()

	trace, err := goncho.DecodeRecallTraceJSON(f)
	if err != nil {
		return exitCodeError(1, fmt.Errorf("goncho recall-diagnostics: decode trace: %w", err))
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

// RunGonchoRecallReplay executes the goncho recall-replay command.
func RunGonchoRecallReplay(cmd *cobra.Command, _ []string) error {
	tracePath, _ := cmd.Flags().GetString("trace")
	emitJSON, _ := cmd.Flags().GetBool("json")
	tracePath = strings.TrimSpace(tracePath)
	if tracePath == "" {
		return exitCodeError(1, errors.New("goncho recall-replay: --trace is required"))
	}
	f, err := os.Open(tracePath)
	if err != nil {
		return exitCodeError(1, fmt.Errorf("goncho recall-replay: open trace: %w", err))
	}
	defer f.Close()

	trace, err := goncho.DecodeRecallTraceJSON(f)
	if err != nil {
		return exitCodeError(1, fmt.Errorf("goncho recall-replay: decode trace: %w", err))
	}
	replay := goncho.BuildRecallReplay(trace)
	if emitJSON {
		enc := json.NewEncoder(cmd.OutOrStdout())
		if err := enc.Encode(replay); err != nil {
			return err
		}
		return nil
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), goncho.FormatRecallReplay(replay))
	return err
}