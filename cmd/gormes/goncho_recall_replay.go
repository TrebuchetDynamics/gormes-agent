package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/goncho"
)

func newGonchoRecallReplayCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall-replay --trace <trace.json>",
		Short: "Replay a durable Goncho RecallTrace retrieval decision",
		Args:  cobra.NoArgs,
		RunE:  runGonchoRecallReplay,
	}
	cmd.Flags().String("trace", "", "path to a RecallTrace JSON file")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

func runGonchoRecallReplay(cmd *cobra.Command, _ []string) error {
	tracePath, _ := cmd.Flags().GetString("trace")
	emitJSON, _ := cmd.Flags().GetBool("json")
	tracePath = strings.TrimSpace(tracePath)
	if tracePath == "" {
		return newExitCodeError(1, errors.New("goncho recall-replay: --trace is required"))
	}
	f, err := os.Open(tracePath)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("goncho recall-replay: open trace: %w", err))
	}
	defer f.Close()

	trace, err := goncho.DecodeRecallTraceJSON(f)
	if err != nil {
		return newExitCodeError(1, fmt.Errorf("goncho recall-replay: decode trace: %w", err))
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
