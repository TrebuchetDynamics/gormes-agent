package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func newKanbanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kanban",
		Short: "Manage the durable local multi-agent Kanban board",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(
		newKanbanInitCommand(),
		newKanbanCreateCommand(),
		newKanbanListCommand(),
		newKanbanShowCommand(),
		newKanbanSpecifyCommand(),
		newKanbanCompleteCommand(),
		newKanbanClaimCommand(),
		newKanbanBlockCommand(),
		newKanbanUnblockCommand(),
		newKanbanLinkCommand(),
		newKanbanBoardsCommand(),
	)
	return cmd
}

func newKanbanInitCommand() *cobra.Command {
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
					Build:  newBuildProvenance(),
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
	Build  buildProvenanceJSON `json:"build"`
	Action string              `json:"action"`
	Path   string              `json:"path"`
}

func newKanbanCreateCommand() *cobra.Command {
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
					Build: newBuildProvenance(),
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
				// Normalize a nil slice to an empty slice so the JSON
				// surface emits `"tasks": []` rather than
				// `"tasks": null` — fleet automation iterating without
				// nil-checks then crashes on null. Same convention as
				// emitSessionListJSON / collectSystemSnapshotForJSON.
				if tasks == nil {
					tasks = []kanban.Task{}
				}
				return writeKanbanJSON(cmd, kanbanListReportJSON{
					Build: newBuildProvenance(),
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
				// In --json mode the not-found path must still
				// emit a parseable document on stdout — fleet
				// automation scraping `kanban show ID --json`
				// can't distinguish a missing task from a crashed
				// command otherwise. Mirrors `session delete
				// --json`'s `action: "not_found"` shape.
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  newBuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanTaskReportJSON{
					Build: newBuildProvenance(),
					Task:  task,
				})
			}
			return writeKanbanTaskText(cmd, task)
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func newKanbanSpecifyCommand() *cobra.Command {
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
			specifier, err := newKanbanTriageSpecifier(cfg)
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
					Build:   newBuildProvenance(),
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
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  newBuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
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
				if jsonOut && isKanbanNotFoundErr(err) {
					_ = writeKanbanLifecycleJSON(cmd, kanbanLifecycleReportJSON{
						Build:  newBuildProvenance(),
						Action: "not_found",
						ID:     args[0],
						Error:  err.Error(),
					})
				}
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, kanbanClaimReportJSON{
					Build:   newBuildProvenance(),
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

// kanbanListReportJSON is the wire shape for `kanban list --json`.
// Fleet automation aggregating Kanban inventory across machines parses
// this to attribute each list snapshot to the binary version that
// emitted it. Build provenance leads — same convention as the rest of
// the `--json` arc. Existing list consumers parsed a bare `[]Task`;
// this slice wraps under `tasks` to make room for `build`.
type kanbanListReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	Tasks []kanban.Task       `json:"tasks"`
}

// kanbanClaimReportJSON is the wire shape for `kanban claim --json`.
// The pre-build-provenance shape was `{task, claimed}`; this slice
// adds `build` at the top so fleet automation orchestrating worker
// assignment across machines can attribute each claim outcome.
type kanbanClaimReportJSON struct {
	Build   buildProvenanceJSON `json:"build"`
	Task    kanban.Task         `json:"task"`
	Claimed bool                `json:"claimed"`
}

// kanbanTaskReportJSON wraps a single kanban.Task with build
// provenance for `kanban create --json` and `kanban show --json`.
// Fleet automation orchestrating Kanban state across machines parses
// this to attribute each task document to the binary version that
// emitted it. Existing kanban.Task fields stay top-level via struct
// embedding — callers parsing the old shape continue to work because
// Go's JSON decoder ignores the unknown `build` field by default.
type kanbanTaskReportJSON struct {
	Build buildProvenanceJSON `json:"build"`
	kanban.Task
}

type kanbanSpecifyReportJSON struct {
	Build   buildProvenanceJSON   `json:"build"`
	Action  string                `json:"action"`
	Outcome kanban.SpecifyOutcome `json:"outcome"`
	Task    kanban.Task           `json:"task,omitempty"`
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
	Error  string              `json:"error,omitempty"`
}

func writeKanbanLifecycleJSON(cmd *cobra.Command, report kanbanLifecycleReportJSON) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func openKanbanStore(ctx context.Context) (*kanban.Store, error) {
	path, err := currentKanbanDBPath()
	if err != nil {
		return nil, err
	}
	return kanban.Open(ctx, path)
}

func currentKanbanDBPath() (string, error) {
	if strings.TrimSpace(os.Getenv("GORMES_KANBAN_DB")) != "" {
		return config.KanbanDBPath(), nil
	}

	current, err := newBoardRegistry().Current()
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

var newKanbanTriageSpecifier = func(cfg config.Config) (kanban.TriageSpecifier, error) {
	providerName := strings.TrimSpace(cfg.Hermes.Provider)
	if providerName == "" && strings.TrimSpace(cfg.Hermes.Endpoint) == "" {
		return nil, nil
	}
	client, err := getOrCreateProviderClient(cfg, providerName)
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

func runTUIKanbanSlashCommand(ctx context.Context, input string) (string, error) {
	args, err := parseTUIKanbanSlashArgs(input)
	if err != nil {
		return "", err
	}

	cmd := newKanbanCommand()
	var stdout, stderr bytes.Buffer
	cmd.SetContext(ctx)
	cmd.SetArgs(args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	err = cmd.Execute()
	output := strings.TrimSpace(strings.Join(nonEmptyStrings(stdout.String(), stderr.String()), "\n"))
	return output, err
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

func newKanbanBoardsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boards",
		Short: "Manage named Kanban boards",
	}
	cmd.AddCommand(
		newKanbanBoardListCommand(),
		newKanbanBoardCreateCommand(),
		newKanbanBoardSwitchCommand(),
		newKanbanBoardRenameCommand(),
		newKanbanBoardRemoveCommand(),
	)
	return cmd
}

func newKanbanBoardListCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all Kanban boards",
		RunE: func(cmd *cobra.Command, _ []string) error {
			reg := newBoardRegistry()
			boards, err := reg.List()
			if err != nil {
				return err
			}
			cur, _ := reg.Current()
			if jsonOut {
				// Normalize a nil slice to an empty slice so the JSON
				// surface emits `"boards": []` rather than
				// `"boards": null`. Same convention as
				// emitSessionListJSON / collectSystemSnapshotForJSON:
				// consumers can iterate without nil-checks.
				if boards == nil {
					boards = []kanban.Board{}
				}
				return writeKanbanJSON(cmd, map[string]any{
					"build":   newBuildProvenance(),
					"current": cur.Name,
					"boards":  boards,
				})
			}
			for _, b := range boards {
				marker := "  "
				if b.Name == cur.Name {
					marker = "* "
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, b.Name)
			}
			if len(boards) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(no boards — using default)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON")
	return cmd
}

func newKanbanBoardCreateCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new Kanban board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Create(args[0]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": newBuildProvenance(),
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

func newKanbanBoardSwitchCommand() *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "switch <name>",
		Short: "Switch to a different Kanban board",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Switch(args[0]); err != nil {
				return err
			}
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": newBuildProvenance(),
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

func newKanbanBoardRenameCommand() *cobra.Command {
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
					"build":   newBuildProvenance(),
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

func newKanbanBoardRemoveCommand() *cobra.Command {
	var jsonOut bool
	var force bool
	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a Kanban board and its database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := newBoardRegistry()
			if err := reg.Remove(args[0]); err != nil {
				return err
			}
			_ = force
			if jsonOut {
				return writeKanbanJSON(cmd, map[string]any{
					"build": newBuildProvenance(),
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
