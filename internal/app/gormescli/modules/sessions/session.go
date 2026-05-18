package sessions

import (
	"fmt"

	"github.com/spf13/cobra"
)

type UnavailableCommandSpec struct {
	Use   string
	Short string
	Row   string
}

type SessionCommandSeams struct {
	RunList            func(*cobra.Command, []string) error
	RunExport          func(*cobra.Command, []string) error
	RunDelete          func(*cobra.Command, []string) error
	RunPrune           func(*cobra.Command, []string) error
	RunBrowse          func(*cobra.Command, []string) error
	UnavailableCommand func(UnavailableCommandSpec) *cobra.Command
}

func NewSessionCommandWithSeams(seams SessionCommandSeams) *cobra.Command {
	seams = seams.withDefaults()
	cmd := &cobra.Command{
		Use:     "session",
		Aliases: []string{"sessions"},
		Short:   "Inspect and export persisted sessions",
		Args:    cobra.NoArgs,
	}
	cmd.AddCommand(
		newSessionListCommand(seams),
		newSessionExportCommand(seams),
		newSessionDeleteCommand(seams),
		newSessionPruneCommand(seams),
		newSessionBrowseCommand(seams),
		seams.UnavailableCommand(UnavailableCommandSpec{
			Use:   "stats",
			Short: "Show Hermes-compatible session statistics",
			Row:   "Session shutdown memory transcript handoff",
		}),
		seams.UnavailableCommand(UnavailableCommandSpec{
			Use:   "rename <session-id> <title>",
			Short: "Rename a persisted session",
			Row:   "Session shutdown memory transcript handoff",
		}),
	)
	return cmd
}

func (s SessionCommandSeams) withDefaults() SessionCommandSeams {
	if s.RunList == nil {
		s.RunList = missingSessionSeam("list")
	}
	if s.RunExport == nil {
		s.RunExport = missingSessionSeam("export")
	}
	if s.RunDelete == nil {
		s.RunDelete = missingSessionSeam("delete")
	}
	if s.RunPrune == nil {
		s.RunPrune = missingSessionSeam("prune")
	}
	if s.RunBrowse == nil {
		s.RunBrowse = missingSessionSeam("browse")
	}
	if s.UnavailableCommand == nil {
		s.UnavailableCommand = func(spec UnavailableCommandSpec) *cobra.Command {
			return &cobra.Command{
				Use:   spec.Use,
				Short: spec.Short,
				RunE: func(*cobra.Command, []string) error {
					return fmt.Errorf("session %s seam is not configured", spec.Use)
				},
			}
		}
	}
	return s
}

func missingSessionSeam(name string) func(*cobra.Command, []string) error {
	return func(*cobra.Command, []string) error {
		return fmt.Errorf("session %s seam is not configured", name)
	}
}

func newSessionListCommand(seams SessionCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent sessions",
		RunE:  seams.RunList,
	}
	cmd.Flags().String("source", "", "only list sessions from this source")
	cmd.Flags().Int("limit", 20, "max sessions to list")
	cmd.Flags().Bool("json", false, "emit a `{build, sessions: [...]}` JSON document (suitable for fleet inventory automation)")
	return cmd
}

func newSessionExportCommand(seams SessionCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export <session-id-or-prefix>",
		Short: "Export a persisted session transcript",
		Args:  cobra.ExactArgs(1),
		RunE:  seams.RunExport,
	}
	cmd.Flags().String("format", "markdown", "export format")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: {build, session_id, format, content}")
	return cmd
}

func newSessionDeleteCommand(seams SessionCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <session-id-or-prefix>",
		Short: "Delete a persisted session",
		Args:  cobra.ExactArgs(1),
		RunE:  seams.RunDelete,
	}
	cmd.Flags().Bool("yes", false, "delete without prompting")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action: 'deleted'|'not_found'|'ambiguous', requested_id, resolved_id, deleted, error?}`")
	return cmd
}

func newSessionPruneCommand(seams SessionCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Delete old persisted sessions",
		RunE:  seams.RunPrune,
	}
	cmd.Flags().Int("older-than", 90, "delete sessions older than N days")
	cmd.Flags().String("source", "", "only prune sessions from this source")
	cmd.Flags().Bool("yes", false, "prune without prompting")
	cmd.Flags().Bool("json", false, "emit machine-readable JSON: `{build, action, older_than_days, source, pruned}`")
	return cmd
}

func newSessionBrowseCommand(seams SessionCommandSeams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse and resume persisted sessions",
		RunE:  seams.RunBrowse,
	}
	cmd.Flags().String("source", "", "only browse sessions from this source")
	cmd.Flags().Int("limit", 500, "max sessions to load")
	cmd.Flags().Bool("no-curses", false, "use the numbered fallback picker")
	return cmd
}
