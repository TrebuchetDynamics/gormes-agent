package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func newKanbanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "Manage the durable local multi-agent Kanban board",
	}
	cmd.AddCommand(
		newKanbanInitCommand(),
		newKanbanCreateCommand(),
		newKanbanListCommand(),
		newKanbanShowCommand(),
		newKanbanCompleteCommand(),
		newKanbanClaimCommand(),
		newKanbanBlockCommand(),
		newKanbanUnblockCommand(),
		newKanbanLinkCommand(),
	)
	return cmd
}

func newKanbanInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the local Kanban database",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := openKanbanStore(cmd.Context())
			if err != nil {
				return err
			}
			defer store.Close()
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "kanban initialized at %s\n", config.KanbanDBPath())
			return err
		},
	}
}

func newKanbanCreateCommand() *cobra.Command {
	var input kanban.CreateTaskInput
	var workspaceKind string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a durable Kanban task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input.Title = args[0]
			input.WorkspaceKind = kanban.WorkspaceKind(workspaceKind)
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
				return writeKanbanJSON(cmd, task)
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
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func newKanbanListCommand() *cobra.Command {
	var status string
	var assignee string
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List durable Kanban tasks",
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
				return writeKanbanJSON(cmd, tasks)
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

func newKanbanShowCommand() *cobra.Command {
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
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, task)
			}
			return writeKanbanTaskText(cmd, task)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func newKanbanCompleteCommand() *cobra.Command {
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
				return err
			}
			if jsonOut {
				return writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
					Build:  newBuildProvenance(),
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

func newKanbanClaimCommand() *cobra.Command {
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
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, struct {
					Task    kanban.Task `json:"task"`
					Claimed bool        `json:"claimed"`
				}{Task: task, Claimed: claimed})
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

func newKanbanBlockCommand() *cobra.Command {
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
					Build:  newBuildProvenance(),
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

func newKanbanUnblockCommand() *cobra.Command {
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
					Build:  newBuildProvenance(),
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

func newKanbanLinkCommand() *cobra.Command {
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
					Build:  newBuildProvenance(),
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

// kanbanLifecycleReportJSON is the wire shape for `kanban
// {complete,block,unblock,link} ... --json`. Fleet automation orchestrating
// task state across machines parses this to observe outcomes without
// scraping prose. Build provenance leads — same convention as the rest of
// the `--json` arc. `action` discriminates the verb; `id` is the affected
// task for unary verbs; `parent`/`child` carry the link arguments;
// `reason` is included on `block`.
type kanbanLifecycleReportJSON struct {
	Build  buildProvenanceJSON `json:"build"`
	Action string              `json:"action"`
	ID     string              `json:"id,omitempty"`
	Parent string              `json:"parent,omitempty"`
	Child  string              `json:"child,omitempty"`
	Reason string              `json:"reason,omitempty"`
}

func writeKanbanLifecycleJSON(cmd *cobra.Command, report kanbanLifecycleReportJSON) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func openKanbanStore(ctx context.Context) (*kanban.Store, error) {
	return kanban.Open(ctx, config.KanbanDBPath())
}

func writeKanbanJSON(cmd *cobra.Command, v any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
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
