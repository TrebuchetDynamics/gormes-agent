package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/gormes-agent/internal/config"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kanban"
)

func (m *Manager) handleKanbanCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	sub, args := parseKanbanSubcommand(ev.Text)

	store, err := kanban.Open(ctx, config.KanbanDBPath())
	if err != nil {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Kanban database not available: "+err.Error())
		return
	}
	defer store.Close()

	switch sub {
	case "create":
		if len(args) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban create <title>")
			return
		}
		task, err := store.CreateTask(ctx, kanban.CreateTaskInput{Title: strings.Join(args, " ")})
		if err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Create failed: "+err.Error())
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Created %s (%s)", task.ID, task.Status))
	case "list":
		tasks, err := store.ListTasks(ctx, kanban.ListFilter{})
		if err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "List failed: "+err.Error())
			return
		}
		if len(tasks) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "No tasks on the board.")
			return
		}
		var b strings.Builder
		for i, t := range tasks {
			if i > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(fmt.Sprintf("%s %s %s", kanbanStatusMark(t.Status), t.ID, t.Title))
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, b.String())
	case "show":
		if len(args) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban show <task-id>")
			return
		}
		task, err := store.GetTask(ctx, args[0])
		if err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Show failed: "+err.Error())
			return
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("%s: %s\n", task.ID, task.Title))
		b.WriteString(fmt.Sprintf("  Status: %s", task.Status))
		if task.Assignee != "" {
			b.WriteString(fmt.Sprintf("\n  Assignee: %s", displayKanbanAssignee(task.Assignee)))
		}
		if task.Priority > 0 {
			b.WriteString(fmt.Sprintf("\n  Priority: %d", task.Priority))
		}
		if task.Body != "" {
			body := task.Body
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			b.WriteString(fmt.Sprintf("\n  Body: %s", body))
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, b.String())
	case "complete":
		if len(args) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban complete <task-id>")
			return
		}
		if err := store.CompleteTask(ctx, args[0], kanban.CompleteTaskInput{}); err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Complete failed: "+err.Error())
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Completed %s", args[0]))
	case "claim":
		if len(args) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban claim <task-id>")
			return
		}
		task, claimed, err := store.ClaimTask(ctx, args[0], kanban.ClaimTaskInput{Worker: ev.UserID})
		if err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Claim failed: "+err.Error())
			return
		}
		if !claimed {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Already claimed: %s (%s)", task.ID, task.Status))
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Claimed %s (assignee=%s)", task.ID, displayKanbanAssignee(task.Assignee)))
	case "unblock":
		if len(args) == 0 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban unblock <task-id>")
			return
		}
		if err := store.UnblockTask(ctx, args[0]); err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Unblock failed: "+err.Error())
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Unblocked %s", args[0]))
	case "block":
		if len(args) < 2 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban block <task-id> <reason>")
			return
		}
		input := kanban.BlockTaskInput{Reason: strings.Join(args[1:], " ")}
		if err := store.BlockTask(ctx, args[0], input); err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Block failed: "+err.Error())
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Blocked %s", args[0]))
	case "link":
		if len(args) < 2 {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Usage: /kanban link <parent-id> <child-id>")
			return
		}
		if err := store.LinkTasks(ctx, args[0], args[1]); err != nil {
			_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "Link failed: "+err.Error())
			return
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Linked %s -> %s", args[0], args[1]))
	case "init":
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, fmt.Sprintf("Kanban initialized at %s", config.KanbanDBPath()))
	default:
		lines := []string{
			"/kanban create <title> — Create a task",
			"/kanban list — List tasks",
			"/kanban show <id> — Show task details",
			"/kanban complete <id> — Complete a task",
			"/kanban claim <id> — Claim a task",
			"/kanban block <id> <reason> — Block a task",
			"/kanban unblock <id> — Unblock a task",
			"/kanban link <parent> <child> — Link tasks",
			"/kanban init — Show kanban database path",
		}
		if sub != "" {
			lines = append([]string{fmt.Sprintf("Unknown subcommand: %s", sub)}, lines...)
		}
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, strings.Join(lines, "\n"))
	}
}

func parseKanbanSubcommand(text string) (string, []string) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || slashCommandName(fields[0]) != "kanban" {
		return "", nil
	}
	if len(fields) == 1 {
		return "", nil
	}
	return strings.ToLower(fields[1]), fields[2:]
}

func kanbanStatusMark(status kanban.Status) string {
	switch status {
	case kanban.StatusTriage:
		return "[~]"
	case kanban.StatusTodo:
		return "[ ]"
	case kanban.StatusReady:
		return "[>]"
	case kanban.StatusRunning:
		return "[*]"
	case kanban.StatusBlocked:
		return "[!]"
	case kanban.StatusDone:
		return "[ok]"
	case kanban.StatusArchived:
		return "[-]"
	default:
		return "[?]"
	}
}

func displayKanbanAssignee(assignee string) string {
	if assignee == "" {
		return "(unassigned)"
	}
	return assignee
}
