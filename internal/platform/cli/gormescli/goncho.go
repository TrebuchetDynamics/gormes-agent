package gormescli

import (
	appgoncho "github.com/TrebuchetDynamics/gormes-agent/internal/app/goncho"
	"github.com/spf13/cobra"
)

// GonchoCommandOptions carries root-owned values for the goncho command tree.
type GonchoCommandOptions struct {
	BuildProvenance func() appgoncho.BuildProvenance
}

func (o GonchoCommandOptions) buildProvenance() appgoncho.BuildProvenance {
	if o.BuildProvenance == nil {
		return appgoncho.BuildProvenance{}
	}
	return o.BuildProvenance()
}

// NewGonchoCommand returns a fresh goncho command tree. Constructor pattern
// eliminates the `resetGonchoDoctorFlags` workaround the package-level var
// version needed for cross-test FlagSet isolation.
func NewGonchoCommand(opts GonchoCommandOptions) *cobra.Command {
	bp := opts.buildProvenance()
	cmd := &cobra.Command{
		Use:   "goncho",
		Short: "Inspect local Goncho memory diagnostics",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(NewGonchoDoctorCommand(bp))
	cmd.AddCommand(NewGonchoRecallDiagnosticsCommand())
	cmd.AddCommand(NewGonchoRecallReplayCommand())
	cmd.AddCommand(NewGonchoMemoryCommand())
	cmd.AddCommand(NewGonchoContinueCommand())
	return cmd
}

// NewGonchoDoctorCommand returns the goncho doctor subcommand.
func NewGonchoDoctorCommand(bp appgoncho.BuildProvenance) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose local Goncho memory topology, queues, and degraded modes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appgoncho.RunGonchoDoctor(cmd, args, bp)
		},
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	cmd.Flags().String("peer", "operator:diagnostic", "peer id for the context dry-run")
	cmd.Flags().String("session", "", "optional session key for the context dry-run")
	cmd.Flags().String("scope", "", "optional recall scope for the context dry-run, for example user")
	cmd.Flags().StringSlice("sources", nil, "optional source allowlist for user-scoped context dry-run")
	cmd.Flags().Bool("require-provider", false, "treat provider/auth readiness as required for this diagnostic")
	return cmd
}

// NewGonchoContinueCommand returns the goncho continue subcommand.
func NewGonchoContinueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "continue [session-key]",
		Short: "List recent sessions or resume a specific session",
		Long: `Without arguments, lists recent sessions with their summaries.
With a session-key argument, loads that session's full context for resumption.`,
		RunE: appgoncho.RunGonchoContinue,
	}
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	cmd.Flags().Int("limit", 10, "max sessions to list")
	return cmd
}

// NewGonchoMemoryCommand returns the goncho memory parent subcommand.
func NewGonchoMemoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Search and inspect Goncho memories",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(NewGonchoMemorySearchCommand())
	cmd.AddCommand(NewGonchoMemoryInspectCommand())
	return cmd
}

// NewGonchoMemorySearchCommand returns the goncho memory search subcommand.
func NewGonchoMemorySearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Goncho memories by keyword",
		Args:  cobra.MinimumNArgs(1),
		RunE:  appgoncho.RunGonchoMemorySearch,
	}
	cmd.Flags().String("peer", "operator", "peer ID for search scope")
	cmd.Flags().String("session", "", "optional session key to scope search")
	cmd.Flags().Int("limit", 10, "max results to return")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

// NewGonchoMemoryInspectCommand returns the goncho memory inspect subcommand.
func NewGonchoMemoryInspectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect <id>",
		Short: "Inspect a single memory entry",
		Args:  cobra.ExactArgs(1),
		RunE:  appgoncho.RunGonchoMemoryInspect,
	}
	cmd.Flags().String("peer", "operator", "peer ID for search scope")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

// NewGonchoRecallDiagnosticsCommand returns the goncho recall-diagnostics subcommand.
func NewGonchoRecallDiagnosticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall-diagnostics --trace <trace.json>",
		Short: "Explain a durable Goncho RecallTrace ranking decision",
		Args:  cobra.NoArgs,
		RunE:  appgoncho.RunGonchoRecallDiagnostics,
	}
	cmd.Flags().String("trace", "", "path to a RecallTrace JSON file")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}

// NewGonchoRecallReplayCommand returns the goncho recall-replay subcommand.
func NewGonchoRecallReplayCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recall-replay --trace <trace.json>",
		Short: "Replay a durable Goncho RecallTrace retrieval decision",
		Args:  cobra.NoArgs,
		RunE:  appgoncho.RunGonchoRecallReplay,
	}
	cmd.Flags().String("trace", "", "path to a RecallTrace JSON file")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	return cmd
}
