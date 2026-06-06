package gormescli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/planning/kanban"
)

const kanbanRowBackedRow = "Hermes Kanban durable board core"

type KanbanCommandOptions struct {
	BuildProvenance    func() BuildProvenance
	ExitCodeError      func(int, error) error
	NewTriageSpecifier func(config.Config) (kanban.TriageSpecifier, error)
}

type kanbanCommand struct {
	opts KanbanCommandOptions
}

func (opts KanbanCommandOptions) withDefaults() KanbanCommandOptions {
	if opts.BuildProvenance == nil {
		opts.BuildProvenance = func() BuildProvenance { return BuildProvenance{} }
	}
	if opts.ExitCodeError == nil {
		opts.ExitCodeError = NewExitCodeError
	}
	if opts.NewTriageSpecifier == nil {
		opts.NewTriageSpecifier = defaultKanbanTriageSpecifier
	}
	return opts
}

func NewKanbanCommand(opts KanbanCommandOptions) *cobra.Command {
	return kanbanCommand{opts: opts.withDefaults()}.newKanbanCommand()
}

func (k kanbanCommand) newKanbanRowBackedCommand(spec RowBackedCommandSpec, children ...*cobra.Command) *cobra.Command {
	return NewRowBackedCommand(spec, RowBackedCommandOptions{BuildProvenance: k.opts.BuildProvenance}, children...)
}

func kanbanUnavailableYesFlag(cmd *cobra.Command) {
	cmd.Flags().BoolP("yes", "y", false, "skip confirmation")
}

func (k kanbanCommand) newKanbanCommand() *cobra.Command {
	var boardOverride string
	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "Manage the durable local multi-agent Kanban board",
		Args:  cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			slug := kanban.NormalizeBoardSlug(boardOverride)
			if slug == "" {
				return nil
			}
			if err := validateKanbanBoardOverride(slug); err != nil {
				return err
			}
			cmd.SetContext(context.WithValue(cmd.Context(), kanbanBoardOverrideContextKey{}, slug))
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&boardOverride, "board", "", "operate on a named Kanban board for this invocation")
	cmd.AddCommand(
		k.newKanbanInitCommand(),
		k.newKanbanCreateCommand(),
		k.newKanbanListCommand(),
		k.newKanbanShowCommand(),
		k.newKanbanRunsCommand(),
		k.newKanbanStatsCommand(),
		k.newKanbanLogCommand(),
		k.newKanbanTailCommand(),
		k.newKanbanGCCommand(),
		k.newKanbanNotifySubscribeCommand(),
		k.newKanbanNotifyListCommand(),
		k.newKanbanNotifyUnsubscribeCommand(),
		k.newKanbanSpecifyCommand(),
		k.newKanbanCompleteCommand(),
		k.newKanbanClaimCommand(),
		k.newKanbanBlockCommand(),
		k.newKanbanUnblockCommand(),
		k.newKanbanPromoteCommand(),
		k.newKanbanLinkCommand(),
		k.newKanbanBoardsCommand(),
	)
	cmd.AddCommand(k.newKanbanRowBackedCommands()...)
	return cmd
}

func (k kanbanCommand) newKanbanRowBackedCommands() []*cobra.Command {
	return []*cobra.Command{
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "assign <task-id> <agent>",
			Short: "Assign a Kanban task to an agent",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:         "unlink <task-id> <target>",
			Short:       "Unlink a Kanban task reference",
			Row:         kanbanRowBackedRow,
			Destructive: true,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "comment <task-id> <text>",
			Short: "Add a Kanban task comment",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:         "archive <task-id>",
			Short:       "Archive a Kanban task",
			Row:         kanbanRowBackedRow,
			Destructive: true,
			FlagSet:     kanbanUnavailableYesFlag,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "dispatch",
			Short: "Dispatch queued Kanban work",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "daemon",
			Short: "Run the Kanban dispatch daemon",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "watch",
			Short: "Watch Kanban board changes",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "heartbeat",
			Short: "Record a Kanban worker heartbeat",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "assignees",
			Short: "List Kanban assignees",
			Row:   kanbanRowBackedRow,
		}),
		k.newKanbanRowBackedCommand(RowBackedCommandSpec{
			Use:   "context <task-id>",
			Short: "Show Kanban task context",
			Row:   kanbanRowBackedRow,
		}),
	}
}

type kanbanBoardOverrideContextKey struct{}

func (k kanbanCommand) newKanbanInitCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize the local Kanban database",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			path := store.DBPath()
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanInitReportJSON{
					Build:  k.opts.BuildProvenance(),
					Action: "initialized",
					Path:   path,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "kanban initialized at %s\n", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON: {build, action, path}")
	return cmd
}

// kanbanInitReportJSON is the wire shape for `kanban init --json`.
// Fleet automation provisioning the local Kanban database across
// machines parses this to verify the seed outcome with binary
// attribution.
type kanbanInitReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	Path   string          `json:"path"`
}

func (k kanbanCommand) newKanbanCreateCommand() *cobra.Command {
	var input kanban.CreateTaskInput
	var workspaceKind string
	var jsonOut bool
	var triage bool
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a durable Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Title = args[0]
			input.WorkspaceKind = kanban.WorkspaceKind(workspaceKind)
			input.Triage = triage
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			task, err := store.CreateTask(cmd.Context(), input)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanTaskReportJSON{
					Build: k.opts.BuildProvenance(),
					Task:  task,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%s, assignee=%s)\n", task.ID, task.Status, displayAssignee(task.Assignee))
			return err
		},
	}
	cmd.Flags().StringVar(&input.Assignee, "assignee", "", "profile name assigned to the task")
	cmd.Flags().StringVar(&input.Body, "body", "", "task context body")
	cmd.Flags().StringArrayVar(&input.ParentIDs, "parent", nil, "parent task id dependency")
	cmd.Flags().IntVar(&input.Priority, "priority", 0, "task priority")
	cmd.Flags().StringVar(&workspaceKind, "workspace-kind", string(kanban.WorkspaceScratch), "workspace kind: scratch, worktree, or dir")
	cmd.Flags().StringVar(&input.WorkspacePath, "workspace-path", "", "workspace path for dir/worktree tasks")
	cmd.Flags().BoolVar(&triage, "triage", false, "park the task in triage for later specification")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanListCommand() *cobra.Command {
	var status string
	var assignee string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List durable Kanban tasks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			tasks, err := store.ListTasks(cmd.Context(), kanban.ListFilter{
				Status:   kanban.Status(strings.TrimSpace(status)),
				Assignee: assignee,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				// Normalize a nil slice to an empty slice so the JSON
				// surface emits `"tasks": []` rather than
				// `"tasks": null` — fleet automation iterating without
				// nil-checks then crashes on null. Same convention as
				// emitSessionListJSON / collectSystemSnapshotForJSON.
				if tasks == nil {
					tasks = []kanban.Task{}
				}
				return writeKanbanJSON(cmd, kanbanListReportJSON{
					Build: k.opts.BuildProvenance(),
					Tasks: tasks,
				})
			}
			if len(tasks) == 0 {
				// Friendly placeholder so an empty board reads as a known
				// state, not a silent/hung command. Mirrors `gormes
				// plugins`'s "No plugins installed." convention. JSON
				// mode skips this and keeps `tasks: []` for parsers.
				msg := "No Kanban tasks."
				if strings.TrimSpace(status) != "" || strings.TrimSpace(assignee) != "" {
					msg = "No Kanban tasks match the given filters."
				}
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), msg); err != nil {
					return err
				}
				return nil
			}
			for _, task := range tasks {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s %s\n", kanbanStatusIcon(task.Status), task.ID, task.Status, task.Title); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().StringVar(&assignee, "assignee", "", "filter by assignee")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanShowCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "Show one Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			task, err := store.GetTask(cmd.Context(), args[0])
			if err != nil {
				// In --json mode the not-found path must still
				// emit a parseable document on stdout — fleet
				// automation scraping `kanban show ID --json`
				// can't distinguish a missing task from a crashed
				// command otherwise. Mirrors `session delete
				// --json`'s `action: "not_found"` shape.
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  k.opts.BuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanTaskReportJSON{
					Build: k.opts.BuildProvenance(),
					Task:  task,
				})
			}
			return writeKanbanTaskText(cmd, task)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanRunsCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "runs <task-id>",
		Short: "Show Kanban task run history",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			runs, err := store.ListRuns(cmd.Context(), taskID)
			if err != nil {
				return err
			}
			if jsonOut {
				if runs == nil {
					runs = []kanban.TaskRun{}
				}
				return writeKanbanJSON(cmd, kanbanRunsReportJSON{
					Build:  k.opts.BuildProvenance(),
					TaskID: taskID,
					Runs:   runs,
				})
			}
			return writeKanbanRunsText(cmd, taskID, runs)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanStatsCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show Kanban board status and assignee counts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			stats, err := store.BoardStats(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanStatsReportJSON{
					Build:      k.opts.BuildProvenance(),
					BoardStats: stats,
				})
			}
			return writeKanbanStatsText(cmd, stats)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanLogCommand() *cobra.Command {
	var tailBytes int64
	cmd := &cobra.Command{
		Use:   "log <task-id>",
		Short: "Print the worker log for a Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if tailBytes < 0 {
				return errors.New("tail must be >= 0")
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			content, ok, err := store.ReadWorkerLog(args[0], tailBytes)
			if err != nil {
				return err
			}
			if !ok {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "(no log for %s - task may not have spawned yet)\n", args[0])
				return fmt.Errorf("no log for %s", args[0])
			}
			if _, err := fmt.Fprint(cmd.OutOrStdout(), content); err != nil {
				return err
			}
			if !strings.HasSuffix(content, "\n") {
				_, err = fmt.Fprintln(cmd.OutOrStdout())
			}
			return err
		},
	}
	cmd.Flags().Int64Var(&tailBytes, "tail", 0, "only print the last N bytes")
	return cmd
}

func (k kanbanCommand) newKanbanTailCommand() *cobra.Command {
	var intervalSeconds float64
	cmd := &cobra.Command{
		Use:   "tail <task-id>",
		Short: "Follow a Kanban task's event stream",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			return runKanbanTail(cmd.Context(), cmd.OutOrStdout(), store, args[0], intervalSeconds)
		},
	}
	cmd.Flags().Float64Var(&intervalSeconds, "interval", 1.0, "poll interval in seconds")
	return cmd
}

type kanbanEventLister interface {
	ListEvents(context.Context, string) ([]kanban.Event, error)
}

func runKanbanTail(ctx context.Context, out io.Writer, lister kanbanEventLister, taskID string, intervalSeconds float64) error {
	if _, err := fmt.Fprintf(out, "Tailing events for %s. Ctrl-C to stop.\n", taskID); err != nil {
		return err
	}
	interval := kanbanTailPollInterval(intervalSeconds)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastID int64
	for {
		events, err := lister.ListEvents(ctx, taskID)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		for _, event := range events {
			if event.ID <= lastID {
				continue
			}
			payload := ""
			if event.Payload != "" {
				payload = " " + event.Payload
			}
			if _, err := fmt.Fprintf(out, "[%s] %s%s\n", formatKanbanTailTimestamp(event.CreatedAt), event.Kind, payload); err != nil {
				return err
			}
			lastID = event.ID
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func kanbanTailPollInterval(seconds float64) time.Duration {
	interval := time.Duration(seconds * float64(time.Second))
	if interval < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	return interval
}

func formatKanbanTailTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

func (k kanbanCommand) newKanbanGCCommand() *cobra.Command {
	var eventRetentionDays int
	var logRetentionDays int
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "gc",
		Short: "Garbage-collect terminal Kanban events and worker logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if eventRetentionDays < 0 {
				return errors.New("event retention days must be >= 0")
			}
			if logRetentionDays < 0 {
				return errors.New("log retention days must be >= 0")
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			eventsDeleted, err := store.PruneTerminalEvents(cmd.Context(), kanbanRetentionDuration(eventRetentionDays))
			if err != nil {
				return err
			}
			logsDeleted, err := store.PruneWorkerLogs(kanbanRetentionDuration(logRetentionDays))
			if err != nil {
				return err
			}
			report := kanbanGCReportJSON{
				Build:              k.opts.BuildProvenance(),
				Action:             "gc",
				EventRetentionDays: eventRetentionDays,
				LogRetentionDays:   logRetentionDays,
				EventsDeleted:      eventsDeleted,
				LogsDeleted:        logsDeleted,
			}
			if jsonOut {
				return writeKanbanJSON(cmd, report)
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Kanban GC pruned %d terminal event(s) and %d worker log file(s).\n", eventsDeleted, logsDeleted)
			return err
		},
	}
	cmd.Flags().IntVar(&eventRetentionDays, "event-retention-days", 30, "terminal task event retention window in days")
	cmd.Flags().IntVar(&logRetentionDays, "log-retention-days", 30, "worker log retention window in days")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanNotifySubscribeCommand() *cobra.Command {
	var input kanban.NotifySubscriptionInput
	cmd := &cobra.Command{
		Use:   "notify-subscribe <task-id>",
		Short: "Subscribe a gateway source to Kanban task events",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			sub, err := store.AddNotifySubscription(cmd.Context(), args[0], input)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Subscribed %s to %s\n", formatKanbanNotifyTarget(sub.Platform, sub.ChatID, sub.ThreadID), sub.TaskID)
			return err
		},
	}
	cmd.Flags().StringVar(&input.Platform, "platform", "", "gateway platform for the subscription")
	cmd.Flags().StringVar(&input.ChatID, "chat-id", "", "gateway chat id for the subscription")
	cmd.Flags().StringVar(&input.ThreadID, "thread-id", "", "gateway thread id for the subscription")
	cmd.Flags().StringVar(&input.UserID, "user-id", "", "gateway user id for the subscription")
	_ = cmd.MarkFlagRequired("platform")
	_ = cmd.MarkFlagRequired("chat-id")
	return cmd
}

func (k kanbanCommand) newKanbanNotifyListCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "notify-list [task-id]",
		Short: "List Kanban notification subscriptions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := ""
			if len(args) == 1 {
				taskID = args[0]
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			subscriptions, err := store.ListNotifySubscriptions(cmd.Context(), taskID)
			if err != nil {
				return err
			}
			if subscriptions == nil {
				subscriptions = []kanban.NotifySubscription{}
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanNotifyListReportJSON{
					Build:         k.opts.BuildProvenance(),
					Subscriptions: subscriptions,
				})
			}
			if len(subscriptions) == 0 {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "(no subscriptions)")
				return err
			}
			for _, sub := range subscriptions {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-10s  %s  (since event %d)\n", sub.TaskID, formatKanbanNotifyTarget(sub.Platform, sub.ChatID, sub.ThreadID), sub.LastEventID); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanNotifyUnsubscribeCommand() *cobra.Command {
	var input kanban.NotifySubscriptionInput
	cmd := &cobra.Command{
		Use:   "notify-unsubscribe <task-id>",
		Short: "Remove a Kanban notification subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			removed, err := store.RemoveNotifySubscription(cmd.Context(), args[0], input)
			if err != nil {
				return err
			}
			if !removed {
				return fmt.Errorf("no such subscription for %s", args[0])
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Unsubscribed from %s\n", args[0])
			return err
		},
	}
	cmd.Flags().StringVar(&input.Platform, "platform", "", "gateway platform for the subscription")
	cmd.Flags().StringVar(&input.ChatID, "chat-id", "", "gateway chat id for the subscription")
	cmd.Flags().StringVar(&input.ThreadID, "thread-id", "", "gateway thread id for the subscription")
	_ = cmd.MarkFlagRequired("platform")
	_ = cmd.MarkFlagRequired("chat-id")
	return cmd
}

func (k kanbanCommand) newKanbanSpecifyCommand() *cobra.Command {
	var author string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "specify <task-id>",
		Short: "Flesh out a triage Kanban task with the configured model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			specifier, err := k.opts.NewTriageSpecifier(cfg)
			if err != nil {
				return err
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			outcome, err := kanban.SpecifyTriageTask(cmd.Context(), store, args[0], specifier, kanban.SpecifyOptions{Author: author})
			if err != nil {
				return err
			}
			task, taskErr := store.GetTask(cmd.Context(), args[0])
			if jsonOut {
				report := kanbanSpecifyReportJSON{
					Build:   k.opts.BuildProvenance(),
					Action:  "specified",
					Outcome: outcome,
				}
				if taskErr == nil {
					report.Task = task
				}
				if err := writeKanbanJSON(cmd, report); err != nil {
					return err
				}
			}
			if !outcome.OK {
				return fmt.Errorf("kanban specify %s: %s", outcome.TaskID, outcome.Reason)
			}
			if !jsonOut {
				titleSuffix := ""
				if outcome.NewTitle != "" {
					titleSuffix = fmt.Sprintf(" - retitled: %q", outcome.NewTitle)
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Specified %s -> %s%s\n", outcome.TaskID, outcome.Status, titleSuffix)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "author name recorded on the audit comment")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

// isKanbanNotFoundErr matches the un-typed `kanban task %q not found`
// shape returned by internal/kanban/store.go. The error isn't
// exported as a sentinel; substring matching is brittle but bounded
// — only one production callsite formats this string, and the test
// fence above pins the shape so any future refactor that breaks it
// is loud.
func isKanbanNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "not found")
}

func (k kanbanCommand) newKanbanCompleteCommand() *cobra.Command {
	var result string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "complete <task-id>",
		Short: "Mark a Kanban task done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.CompleteTask(cmd.Context(), args[0], kanban.CompleteTaskInput{Result: result}); err != nil {
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  k.opts.BuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
				return err
			}
			if jsonOut {
				return writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
					Build:  k.opts.BuildProvenance(),
					Action: "completed",
					ID:     args[0],
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Completed %s\n", args[0])
			return err
		},
	}
	cmd.Flags().StringVar(&result, "result", "", "completion summary")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanClaimCommand() *cobra.Command {
	var worker string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "claim <task-id>",
		Short: "Atomically claim a ready Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			task, claimed, err := store.ClaimTask(cmd.Context(), args[0], kanban.ClaimTaskInput{Worker: worker})
			if err != nil {
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  k.opts.BuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanClaimReportJSON{
					Build:   k.opts.BuildProvenance(),
					Task:    task,
					Claimed: claimed,
				})
			}
			if claimed {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Claimed %s\n", task.ID)
			} else {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Not claimed %s (%s)\n", task.ID, task.Status)
			}
			return err
		},
	}
	cmd.Flags().StringVar(&worker, "worker", "", "worker/profile name claiming the task")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBlockCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "block <task-id> [reason]",
		Short: "Block a Kanban task with a reason",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reason := ""
			if len(args) == 2 {
				reason = args[1]
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.BlockTask(cmd.Context(), args[0], kanban.BlockTaskInput{Reason: reason}); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
					Build:  k.opts.BuildProvenance(),
					Action: "blocked",
					ID:     args[0],
					Reason: reason,
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Blocked %s\n", args[0])
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanUnblockCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "unblock <task-id>",
		Short: "Unblock a Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.UnblockTask(cmd.Context(), args[0]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
					Build:  k.opts.BuildProvenance(),
					Action: "unblocked",
					ID:     args[0],
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Unblocked %s\n", args[0])
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanPromoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "promote <task-id> [reason...] [--ids <task-id>...] [--force] [--dry-run] [--json]",
		Short:              "Manually promote todo or blocked Kanban tasks to ready",
		DisableFlagParsing: true,
		DisableSuggestions: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kanbanPromoteHelpRequested(args) {
				return cmd.Help()
			}
			parsed, err := parseKanbanPromoteArgs(args)
			if err != nil {
				return err
			}
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()

			reason := strings.TrimSpace(strings.Join(parsed.reason, " "))
			ids := dedupeKanbanPromoteIDs(append([]string{parsed.taskID}, parsed.ids...))
			results := make([]kanbanPromoteResultJSON, 0, len(ids))
			failed := false
			for _, id := range ids {
				result, err := store.PromoteTask(cmd.Context(), id, kanban.PromoteTaskInput{
					Actor:  kanbanPromoteActor(),
					Reason: reason,
					Force:  parsed.force,
					DryRun: parsed.dryRun,
				})
				if err != nil {
					return err
				}
				if !result.Promoted {
					failed = true
				}
				results = append(results, kanbanPromoteResultFromStore(result))
			}

			if parsed.jsonOut {
				if len(results) == 1 {
					if err := writeKanbanJSON(cmd, results[0]); err != nil {
						return err
					}
				} else if err := writeKanbanJSON(cmd, results); err != nil {
					return err
				}
				if failed {
					return k.opts.ExitCodeError(1, errors.New("kanban promote failed"))
				}
				return nil
			}

			label := "Promoted"
			tag := ""
			if parsed.dryRun {
				label = "Would promote"
				tag = " (dry)"
			}
			for _, result := range results {
				if result.Promoted {
					suffix := ""
					if result.Reason != nil && *result.Reason != "" {
						suffix = ": " + *result.Reason
					}
					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s -> ready%s%s\n", label, result.TaskID, tag, suffix); err != nil {
						return err
					}
					continue
				}
				msg := "unknown error"
				if result.Error != nil && *result.Error != "" {
					msg = *result.Error
				}
				if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "cannot promote %s: %s\n", result.TaskID, msg); err != nil {
					return err
				}
			}
			if failed {
				return k.opts.ExitCodeError(1, errors.New("kanban promote failed"))
			}
			return nil
		},
	}
	cmd.Flags().StringArray("ids", nil, "Additional task ids to promote with the same reason")
	cmd.Flags().Bool("force", false, "Promote even if parent dependencies are not yet done/archived")
	cmd.Flags().Bool("dry-run", false, "Validate the promotion without mutating state")
	cmd.Flags().Bool("json", false, "Emit machine-readable JSON result")
	return cmd
}

type kanbanPromoteArgs struct {
	taskID  string
	reason  []string
	ids     []string
	force   bool
	dryRun  bool
	jsonOut bool
}

func kanbanPromoteHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func parseKanbanPromoteArgs(args []string) (kanbanPromoteArgs, error) {
	var parsed kanbanPromoteArgs
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--force":
			parsed.force = true
		case arg == "--dry-run":
			parsed.dryRun = true
		case arg == "--json":
			parsed.jsonOut = true
		case arg == "--ids":
			start := len(parsed.ids)
			for i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
				parsed.ids = append(parsed.ids, args[i])
			}
			if len(parsed.ids) == start {
				return parsed, errors.New("kanban promote: --ids requires at least one task id")
			}
		case strings.HasPrefix(arg, "--ids="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--ids="))
			if value == "" {
				return parsed, errors.New("kanban promote: --ids requires at least one task id")
			}
			for _, id := range strings.Split(value, ",") {
				if id = strings.TrimSpace(id); id != "" {
					parsed.ids = append(parsed.ids, id)
				}
			}
		case strings.HasPrefix(arg, "--"):
			return parsed, fmt.Errorf("unknown flag: %s", arg)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 || strings.TrimSpace(positional[0]) == "" {
		return parsed, errors.New("kanban promote: task_id is required")
	}
	parsed.taskID = positional[0]
	parsed.reason = positional[1:]
	return parsed, nil
}

func dedupeKanbanPromoteIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func kanbanPromoteActor() string {
	for _, value := range []string{os.Getenv("GORMES_PROFILE"), os.Getenv("USER"), os.Getenv("USERNAME")} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "gormes"
}

func kanbanPromoteResultFromStore(result kanban.PromoteTaskResult) kanbanPromoteResultJSON {
	out := kanbanPromoteResultJSON{
		TaskID:   result.TaskID,
		Promoted: result.Promoted,
		DryRun:   result.DryRun,
		Forced:   result.Forced,
		Reason:   kanbanPromoteStringPtr(result.Reason),
		Error:    kanbanPromoteStringPtr(result.Error),
	}
	return out
}

func kanbanPromoteStringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func (k kanbanCommand) newKanbanLinkCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "link <parent-id> <child-id>",
		Short: "Add a dependency link",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			if err := store.LinkTasks(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
					Build:  k.opts.BuildProvenance(),
					Action: "linked",
					Parent: args[0],
					Child:  args[1],
				})
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Linked %s -> %s\n", args[0], args[1])
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

// kanbanListReportJSON is the wire shape for `kanban list --json`.
// Fleet automation aggregating Kanban inventory across machines parses
// this to attribute each list snapshot to the binary version that
// emitted it. Build provenance leads — same convention as the rest of
// the `--json` arc. Existing list consumers parsed a bare `[]Task`;
// this slice wraps under `tasks` to make room for `build`.
type kanbanListReportJSON struct {
	Build BuildProvenance `json:"build"`
	Tasks []kanban.Task   `json:"tasks"`
}

// kanbanClaimReportJSON is the wire shape for `kanban claim --json`.
// The pre-build-provenance shape was `{task, claimed}`; this slice
// adds `build` at the top so fleet automation orchestrating worker
// assignment across machines can attribute each claim outcome.
type kanbanClaimReportJSON struct {
	Build   BuildProvenance `json:"build"`
	Task    kanban.Task     `json:"task"`
	Claimed bool            `json:"claimed"`
}

type kanbanRunsReportJSON struct {
	Build  BuildProvenance  `json:"build"`
	TaskID string           `json:"task_id"`
	Runs   []kanban.TaskRun `json:"runs"`
}

type kanbanStatsReportJSON struct {
	Build BuildProvenance `json:"build"`
	kanban.BoardStats
}

type kanbanGCReportJSON struct {
	Build              BuildProvenance `json:"build"`
	Action             string          `json:"action"`
	EventRetentionDays int             `json:"event_retention_days"`
	LogRetentionDays   int             `json:"log_retention_days"`
	EventsDeleted      int64           `json:"events_deleted"`
	LogsDeleted        int             `json:"logs_deleted"`
}

type kanbanNotifyListReportJSON struct {
	Build         BuildProvenance             `json:"build"`
	Subscriptions []kanban.NotifySubscription `json:"subscriptions"`
}

// kanbanTaskReportJSON wraps a single kanban.Task with build
// provenance for `kanban create --json` and `kanban show --json`.
// Fleet automation orchestrating Kanban state across machines parses
// this to attribute each task document to the binary version that
// emitted it. Existing kanban.Task fields stay top-level via struct
// embedding — callers parsing the old shape continue to work because
// Go's JSON decoder ignores the unknown `build` field by default.
type kanbanTaskReportJSON struct {
	Build BuildProvenance `json:"build"`
	kanban.Task
}

type kanbanSpecifyReportJSON struct {
	Build   BuildProvenance       `json:"build"`
	Action  string                `json:"action"`
	Outcome kanban.SpecifyOutcome `json:"outcome"`
	Task    kanban.Task           `json:"task,omitempty"`
}

type kanbanPromoteResultJSON struct {
	TaskID   string  `json:"task_id"`
	Promoted bool    `json:"promoted"`
	DryRun   bool    `json:"dry_run"`
	Forced   bool    `json:"forced"`
	Reason   *string `json:"reason"`
	Error    *string `json:"error"`
}

// kanbanLifecycleReportJSON is the wire shape for `kanban
// {complete,block,unblock,link} ... --json`. Fleet automation orchestrating
// task state across machines parses this to observe outcomes without
// scraping prose. Build provenance leads — same convention as the rest of
// the `--json` arc. `action` discriminates the verb; `id` is the affected
// task for unary verbs; `parent`/`child` carry the link arguments;
// `reason` is included on `block`.
type kanbanLifecycleReportJSON struct {
	Build  BuildProvenance `json:"build"`
	Action string          `json:"action"`
	ID     string          `json:"id,omitempty"`
	Parent string          `json:"parent,omitempty"`
	Child  string          `json:"child,omitempty"`
	Reason string          `json:"reason,omitempty"`
	Error  string          `json:"error,omitempty"`
}

func writeKanbanLifecycleJSON(cmd *cobra.Command, report kanbanLifecycleReportJSON) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func writeKanbanRunsText(cmd *cobra.Command, taskID string, runs []kanban.TaskRun) error {
	if len(runs) == 0 {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "(no runs yet for %s)\n", taskID)
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-3s  %-14s  %8s  %s\n", "#", "OUTCOME", "ELAPSED", "STARTED"); err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, run := range runs {
		outcome := strings.TrimSpace(string(run.Outcome))
		if outcome == "" {
			outcome = "unknown"
			if run.EndedAt.IsZero() {
				outcome = "running"
			}
		}
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%3d  %-14s  %8s  %s\n",
			i+1,
			outcome,
			formatKanbanRunElapsed(run.StartedAt, run.EndedAt, now),
			formatKanbanRunStarted(run.StartedAt),
		); err != nil {
			return err
		}
		if summary := kanbanRunFirstLine(run.Summary, 100); summary != "" {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "     summary: %s\n", summary); err != nil {
				return err
			}
		}
		if message := kanbanRunFirstLine(run.Error, 100); message != "" {
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "     error: %s\n", message); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeKanbanStatsText(cmd *cobra.Command, stats kanban.BoardStats) error {
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "By status:"); err != nil {
		return err
	}
	for _, status := range []kanban.Status{
		kanban.StatusTriage,
		kanban.StatusTodo,
		kanban.StatusReady,
		kanban.StatusRunning,
		kanban.StatusBlocked,
		kanban.StatusDone,
	} {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-8s  %d\n", status, stats.ByStatus[string(status)]); err != nil {
			return err
		}
	}
	if len(stats.ByAssignee) > 0 {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "\nBy assignee:"); err != nil {
			return err
		}
		assignees := make([]string, 0, len(stats.ByAssignee))
		for assignee := range stats.ByAssignee {
			assignees = append(assignees, assignee)
		}
		sort.Strings(assignees)
		for _, assignee := range assignees {
			counts := stats.ByAssignee[assignee]
			statuses := make([]string, 0, len(counts))
			for status := range counts {
				statuses = append(statuses, status)
			}
			sort.Strings(statuses)
			parts := make([]string, 0, len(statuses))
			for _, status := range statuses {
				parts = append(parts, fmt.Sprintf("%s=%d", status, counts[status]))
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "  %-20s  %s\n", assignee, strings.Join(parts, ", ")); err != nil {
				return err
			}
		}
	}
	if stats.OldestReadyAgeSeconds != nil {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "\nOldest ready task age: %ds\n", *stats.OldestReadyAgeSeconds); err != nil {
			return err
		}
	}
	return nil
}

func formatKanbanNotifyTarget(platform, chatID, threadID string) string {
	target := strings.TrimSpace(platform) + ":" + strings.TrimSpace(chatID)
	if threadID = strings.TrimSpace(threadID); threadID != "" {
		target += ":" + threadID
	}
	return target
}

func formatKanbanRunStarted(start time.Time) string {
	if start.IsZero() {
		return "-"
	}
	return start.UTC().Format(time.RFC3339)
}

func formatKanbanRunElapsed(start, end, fallbackEnd time.Time) string {
	if start.IsZero() {
		return "-"
	}
	if end.IsZero() {
		end = fallbackEnd
	}
	if end.Before(start) {
		return "0s"
	}
	elapsed := end.Sub(start).Round(time.Second)
	if elapsed < time.Minute {
		return fmt.Sprintf("%ds", int(elapsed.Seconds()))
	}
	if elapsed < time.Hour {
		return fmt.Sprintf("%dm", int(elapsed.Minutes()))
	}
	return fmt.Sprintf("%.1fh", elapsed.Hours())
}

func kanbanRunFirstLine(value string, max int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	line := strings.SplitN(trimmed, "\n", 2)[0]
	if len(line) <= max {
		return line
	}
	if max <= 0 {
		return ""
	}
	return line[:max]
}

func openKanbanStore(ctx context.Context) (*kanban.Store, error) {
	path, err := currentKanbanDBPath(ctx)
	if err != nil {
		return nil, err
	}
	return kanban.Open(ctx, path)
}

func currentKanbanDBPath(ctx context.Context) (string, error) {
	if strings.TrimSpace(os.Getenv("GORMES_KANBAN_DB")) != "" {
		return config.KanbanDBPath(), nil
	}

	reg := newBoardRegistry()
	if override := kanbanBoardOverrideFromContext(ctx); override != "" {
		if err := validateKanbanBoardOverride(override); err != nil {
			return "", err
		}
		return reg.BoardPath(override), nil
	}

	current, err := reg.Current()
	if err != nil {
		return "", err
	}
	if current.Name == "default" {
		return current.Path, nil
	}
	if err := kanban.ValidateBoardSlug(current.Name); err != nil {
		return "", fmt.Errorf("current kanban board %q is invalid: %w", current.Name, err)
	}
	if _, err := os.Stat(filepath.Dir(current.Path)); os.IsNotExist(err) {
		return "", fmt.Errorf("current kanban board %q does not exist", current.Name)
	} else if err != nil {
		return "", fmt.Errorf("inspect current kanban board %q: %w", current.Name, err)
	}
	return current.Path, nil
}

func kanbanBoardOverrideFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(kanbanBoardOverrideContextKey{}).(string); ok {
		return value
	}
	return ""
}

func validateKanbanBoardOverride(slug string) error {
	if slug == "" {
		return nil
	}
	if slug != "default" {
		if err := kanban.ValidateBoardSlug(slug); err != nil {
			return err
		}
	}
	if strings.TrimSpace(os.Getenv("GORMES_KANBAN_DB")) != "" {
		return nil
	}
	if slug == "default" {
		return nil
	}
	boardDir := filepath.Dir(newBoardRegistry().BoardPath(slug))
	if _, err := os.Stat(boardDir); os.IsNotExist(err) {
		return fmt.Errorf("board %q does not exist", slug)
	} else if err != nil {
		return fmt.Errorf("inspect board %q: %w", slug, err)
	}
	return nil
}

var kanbanProviderPool = NewProviderClientPool()

var kanbanTriageSpecifierForTesting func(config.Config) (kanban.TriageSpecifier, error)

func SetKanbanTriageSpecifierForTesting(t interface{ Cleanup(func()) }, fn func(config.Config) (kanban.TriageSpecifier, error)) {
	previous := kanbanTriageSpecifierForTesting
	kanbanTriageSpecifierForTesting = fn
	t.Cleanup(func() { kanbanTriageSpecifierForTesting = previous })
}

func KanbanTailPollInterval(seconds float64) time.Duration { return kanbanTailPollInterval(seconds) }

func OpenKanbanStore(ctx context.Context) (*kanban.Store, error) { return openKanbanStore(ctx) }

func defaultKanbanTriageSpecifier(cfg config.Config) (kanban.TriageSpecifier, error) {
	if kanbanTriageSpecifierForTesting != nil {
		return kanbanTriageSpecifierForTesting(cfg)
	}
	providerName := strings.TrimSpace(cfg.Hermes.Provider)
	if providerName == "" && strings.TrimSpace(cfg.Hermes.Endpoint) == "" {
		return nil, nil
	}
	client, err := kanbanProviderPool.GetOrCreate(cfg, providerName)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(cfg.Hermes.Model)
	if model == "" {
		model = "hermes-agent"
	}
	return kanban.HermesTriageSpecifier{
		Client:      client,
		Model:       model,
		MaxTokens:   1500,
		Temperature: 0.3,
		Timeout:     120 * time.Second,
	}, nil
}

func RunTUIKanbanSlashCommand(ctx context.Context, input string, opts KanbanCommandOptions) (string, error) {
	args, err := parseTUIKanbanSlashArgs(input)
	if err != nil {
		return "", err
	}
	if isKanbanSlashHelpAlias(args) {
		return slashKanbanHelp, nil
	}

	cmd := kanbanCommand{opts: opts.withDefaults()}.newKanbanCommand()
	if action := firstKanbanSlashAction(args); action != "" && !knownKanbanSlashAction(cmd, action) {
		return fmt.Sprintf("⚠ /kanban usage error: unknown action %q\nRun `/kanban` for common subcommands.", action), nil
	}

	var stdout, stderr bytes.Buffer
	cmd.SetContext(ctx)
	cmd.SetArgs(args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	target, _, _ := cmd.Find(args)
	if target == nil {
		target = cmd
	}
	err = cmd.Execute()
	output := rewriteKanbanSlashOutput(strings.TrimSpace(strings.Join(nonEmptyStrings(stdout.String(), stderr.String()), "\n")))
	if err != nil && isKanbanSlashUsageError(err) {
		return formatKanbanSlashUsageError(err, output, target), nil
	}
	return output, err
}

const slashKanbanHelp = `**/kanban** - manage the shared task board.

Common subcommands:
  ` + "`list` (alias `ls`)   List tasks on the current board" + `
  ` + "`show <id>`           Task details, comments, and events" + `
  ` + "`stats`               Per-status and per-assignee counts" + `
  ` + "`create <title>...`   Create a task" + `
  ` + "`complete <id>...`    Mark task(s) done" + `
  ` + "`block <id> [reason]` Mark blocked; `unblock <id>` to revive" + `
  ` + "`link <parent> <child>` Add a dependency link" + `
  ` + "`boards list`         Show all boards" + `
  ` + "`specify <id>`        Flesh out a triage task" + `
  ` + "`notify-list <id>`    Notification subscriptions" + `
  ` + "`runs <id>`           Attempt history" + `
  ` + "`log <id>`            Worker log" + `
  ` + "`gc`                  Prune terminal events and old worker logs" + `

Run ` + "`/kanban <subcommand> -h`" + ` for arguments. Read-only commands are safe while an agent is running.`

func isKanbanSlashHelpAlias(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch strings.TrimSpace(args[0]) {
	case "help", "--help", "-h", "?":
		return true
	default:
		return false
	}
}

func firstKanbanSlashAction(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if arg == "" {
			continue
		}
		if arg == "--" {
			if i+1 < len(args) {
				return strings.TrimSpace(args[i+1])
			}
			return ""
		}
		if arg == "--board" || arg == "-p" || arg == "--profile" || arg == "--skills" {
			i++
			continue
		}
		if strings.HasPrefix(arg, "--board=") || strings.HasPrefix(arg, "--profile=") || strings.HasPrefix(arg, "--skills=") {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return arg
	}
	return ""
}

func knownKanbanSlashAction(cmd *cobra.Command, action string) bool {
	action = strings.TrimSpace(action)
	if action == "" {
		return true
	}
	for _, child := range cmd.Commands() {
		if child.Name() == action {
			return true
		}
		for _, alias := range child.Aliases {
			if alias == action {
				return true
			}
		}
	}
	return false
}

func isKanbanSlashUsageError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"accepts ",
		"unknown flag",
		"unknown shorthand flag",
		"required flag",
		"requires at least",
		"requires a subcommand",
		"invalid argument",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func formatKanbanSlashUsageError(err error, output string, target *cobra.Command) string {
	body := strings.TrimSpace(output)
	if body == "" && target != nil {
		body = strings.TrimSpace(rewriteKanbanSlashOutput(target.UsageString()))
	}
	if body == "" {
		return "⚠ /kanban usage error: " + err.Error()
	}
	return "⚠ /kanban usage error: " + err.Error() + "\n" + body
}

func rewriteKanbanSlashOutput(output string) string {
	replacer := strings.NewReplacer(
		"Usage:\n  kanban", "Usage:\n  /kanban",
		"Usage:\n  gormes kanban", "Usage:\n  /kanban",
		"Use \"kanban ", "Use \"/kanban ",
		"Use \"gormes kanban ", "Use \"/kanban ",
		"Use \"kanban", "Use \"/kanban",
		"Use \"gormes kanban", "Use \"/kanban",
	)
	return replacer.Replace(output)
}

func parseTUIKanbanSlashArgs(input string) ([]string, error) {
	fields, err := splitTUIKanbanSlashFields(input)
	if err != nil {
		return nil, err
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty kanban command")
	}
	name := strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	if name != "kanban" {
		return nil, fmt.Errorf("expected /kanban command, got %q", fields[0])
	}
	return fields[1:], nil
}

func splitTUIKanbanSlashFields(input string) ([]string, error) {
	var fields []string
	var b strings.Builder
	var quote rune
	escaped := false
	flush := func() {
		if b.Len() == 0 {
			return
		}
		fields = append(fields, b.String())
		b.Reset()
	}

	for _, r := range strings.TrimSpace(input) {
		if escaped {
			b.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
			continue
		}
		switch {
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	if escaped {
		b.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in /kanban command")
	}
	flush()
	return fields, nil
}

func nonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func writeKanbanJSON(cmd *cobra.Command, v any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func kanbanRetentionDuration(days int) time.Duration {
	return time.Duration(days) * 24 * time.Hour
}

func writeKanbanTaskText(cmd *cobra.Command, task kanban.Task) error {
	lines := []string{
		fmt.Sprintf("%s %s %s", kanbanStatusIcon(task.Status), task.ID, task.Title),
		fmt.Sprintf("status: %s", task.Status),
		fmt.Sprintf("assignee: %s", displayAssignee(task.Assignee)),
	}
	if task.Body != "" {
		lines = append(lines, "body: "+task.Body)
	}
	if len(task.ParentIDs) > 0 {
		lines = append(lines, "parents: "+strings.Join(task.ParentIDs, ", "))
	}
	if len(task.ChildIDs) > 0 {
		lines = append(lines, "children: "+strings.Join(task.ChildIDs, ", "))
	}
	if task.Result != "" {
		lines = append(lines, "result: "+task.Result)
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), strings.Join(lines, "\n"))
	return err
}

func displayAssignee(assignee string) string {
	if strings.TrimSpace(assignee) == "" {
		return "unassigned"
	}
	return assignee
}

func kanbanStatusIcon(status kanban.Status) string {
	switch status {
	case kanban.StatusTodo:
		return "[ ]"
	case kanban.StatusReady:
		return ">"
	case kanban.StatusRunning:
		return "*"
	case kanban.StatusBlocked:
		return "!"
	case kanban.StatusDone:
		return "ok"
	case kanban.StatusArchived:
		return "-"
	default:
		return "?"
	}
}

func newBoardRegistry() *kanban.BoardRegistry {
	return kanban.NewBoardRegistry(config.KanbanHome())
}

func (k kanbanCommand) newKanbanBoardsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boards",
		Short: "Manage named Kanban boards",
	}
	cmd.AddCommand(
		k.newKanbanBoardListCommand(),
		k.newKanbanBoardCreateCommand(),
		k.newKanbanBoardSwitchCommand(),
		k.newKanbanBoardShowCommand(),
		k.newKanbanBoardRenameCommand(),
		k.newKanbanBoardRemoveCommand(),
	)
	return cmd
}

type kanbanBoardListReportJSON struct {
	Build   BuildProvenance          `json:"build"`
	Current string                   `json:"current"`
	Boards  []kanbanBoardSummaryJSON `json:"boards"`
}

type kanbanBoardShowReportJSON struct {
	Build   BuildProvenance        `json:"build"`
	Current string                 `json:"current"`
	Board   kanbanBoardSummaryJSON `json:"board"`
}

type kanbanBoardSummaryJSON struct {
	Name    string         `json:"name"`
	Path    string         `json:"path"`
	Current bool           `json:"current"`
	Counts  map[string]int `json:"counts"`
	Total   int            `json:"total"`
}

func (k kanbanCommand) newKanbanBoardListCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all Kanban boards",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg := newBoardRegistry()
			boards, current, err := kanbanBoardSummaries(cmd.Context(), reg)
			if err != nil {
				return err
			}
			if jsonOut {
				if boards == nil {
					boards = []kanbanBoardSummaryJSON{}
				}
				return writeKanbanJSON(cmd, kanbanBoardListReportJSON{
					Build:   k.opts.BuildProvenance(),
					Current: current,
					Boards:  boards,
				})
			}
			for _, b := range boards {
				marker := "  "
				if b.Current {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%-24s %4d %s\n", marker, b.Name, b.Total, formatKanbanBoardCounts(b.Counts))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBoardCreateCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"new"},
		Short:   "Create a new Kanban board",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Create(args[0]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": k.opts.BuildProvenance(),
					"board": args[0],
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created board %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBoardSwitchCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "switch <name>",
		Aliases: []string{"use"},
		Short:   "Switch to a different Kanban board",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Switch(args[0]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": k.opts.BuildProvenance(),
					"board": args[0],
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched to board %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBoardShowCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:     "show [name]",
		Aliases: []string{"current"},
		Short:   "Show one Kanban board",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			board, current, err := kanbanBoardSummary(cmd.Context(), reg, name)
			if err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanBoardShowReportJSON{
					Build:   k.opts.BuildProvenance(),
					Current: current,
					Board:   board,
				})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Board: %s\n", board.Name); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Current: %t\n", board.Current); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "DB path: %s\n", board.Path); err != nil {
				return err
			}
			counts := formatKanbanBoardCounts(board.Counts)
			if counts == "" {
				counts = "(empty)"
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Tasks: %d total %s\n", board.Total, counts)
			return err
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBoardRenameCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a Kanban board",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Rename(args[0], args[1]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build":   k.opts.BuildProvenance(),
					"oldName": args[0],
					"newName": args[1],
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Renamed board %q to %q\n", args[0], args[1])
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func (k kanbanCommand) newKanbanBoardRemoveCommand() *cobra.Command {
	var jsonOut bool
	var force bool
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a Kanban board and its database",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Remove(args[0]); err != nil {
				return err
			}
			_ = force
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": k.opts.BuildProvenance(),
					"board": args[0],
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed board %q\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation")
	return cmd
}

func kanbanBoardSummaries(ctx context.Context, reg *kanban.BoardRegistry) ([]kanbanBoardSummaryJSON, string, error) {
	current, err := reg.Current()
	if err != nil {
		return nil, "", err
	}
	boards := []kanban.Board{{Name: "default", Path: reg.BoardPath("default")}}
	named, err := reg.List()
	if err != nil {
		return nil, "", err
	}
	boards = append(boards, named...)

	summaries := make([]kanbanBoardSummaryJSON, 0, len(boards))
	for _, board := range boards {
		counts, total, err := countKanbanBoardTasks(ctx, board.Path)
		if err != nil {
			return nil, "", fmt.Errorf("count board %q: %w", board.Name, err)
		}
		summaries = append(summaries, kanbanBoardSummaryJSON{
			Name:    board.Name,
			Path:    board.Path,
			Current: board.Name == current.Name,
			Counts:  counts,
			Total:   total,
		})
	}
	return summaries, current.Name, nil
}

func kanbanBoardSummary(ctx context.Context, reg *kanban.BoardRegistry, rawName string) (kanbanBoardSummaryJSON, string, error) {
	current, err := reg.Current()
	if err != nil {
		return kanbanBoardSummaryJSON{}, "", err
	}
	name := kanban.NormalizeBoardSlug(rawName)
	if name == "" {
		name = current.Name
	}
	if name != "default" {
		if err := kanban.ValidateBoardSlug(name); err != nil {
			return kanbanBoardSummaryJSON{}, "", err
		}
		if _, err := os.Stat(filepath.Dir(reg.BoardPath(name))); os.IsNotExist(err) {
			return kanbanBoardSummaryJSON{}, "", fmt.Errorf("board %q does not exist", name)
		} else if err != nil {
			return kanbanBoardSummaryJSON{}, "", fmt.Errorf("inspect board %q: %w", name, err)
		}
	}
	path := reg.BoardPath(name)
	counts, total, err := countKanbanBoardTasks(ctx, path)
	if err != nil {
		return kanbanBoardSummaryJSON{}, "", fmt.Errorf("count board %q: %w", name, err)
	}
	return kanbanBoardSummaryJSON{
		Name:    name,
		Path:    path,
		Current: name == current.Name,
		Counts:  counts,
		Total:   total,
	}, current.Name, nil
}

func countKanbanBoardTasks(ctx context.Context, dbPath string) (map[string]int, int, error) {
	counts := map[string]int{}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return counts, 0, nil
	} else if err != nil {
		return nil, 0, err
	}
	store, err := kanban.Open(ctx, dbPath)
	if err != nil {
		return nil, 0, err
	}
	defer store.Close()
	tasks, err := store.ListTasks(ctx, kanban.ListFilter{})
	if err != nil {
		return nil, 0, err
	}
	for _, task := range tasks {
		counts[string(task.Status)]++
	}
	return counts, len(tasks), nil
}

func formatKanbanBoardCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}
	ordered := []string{
		string(kanban.StatusTriage),
		string(kanban.StatusTodo),
		string(kanban.StatusReady),
		string(kanban.StatusRunning),
		string(kanban.StatusBlocked),
		string(kanban.StatusDone),
		string(kanban.StatusArchived),
	}
	seen := map[string]bool{}
	var parts []string
	for _, status := range ordered {
		if n := counts[status]; n > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", status, n))
			seen[status] = true
		}
	}
	var rest []string
	for status, n := range counts {
		if !seen[status] && n > 0 {
			rest = append(rest, fmt.Sprintf("%s=%d", status, n))
		}
	}
	sort.Strings(rest)
	parts = append(parts, rest...)
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
