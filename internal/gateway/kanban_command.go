package gateway

import (
	"context"
	"errors"
	"strings"
)

const maxGatewayKanbanOutputBytes = 3800

func (m *Manager) handleKanbanCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	output, err := runGatewayKanbanSlash(ctx, m.cfg.KanbanSlashRunner, ev.Text)
	if err != nil {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "kanban error: "+sanitizeConfigReloadError(err))
		return
	}
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, boundGatewayKanbanOutput(output))
}

func runGatewayKanbanSlash(ctx context.Context, runner KanbanSlashRunner, input string) (string, error) {
	if runner == nil {
		return "", errors.New("kanban command runner unavailable")
	}
	input = strings.TrimSpace(input)
	if input == "" {
		input = "/kanban"
	}
	return runner(ctx, input)
}

func boundGatewayKanbanOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return "(no output)"
	}
	if len(output) <= maxGatewayKanbanOutputBytes {
		return output
	}
	return output[:maxGatewayKanbanOutputBytes] + "\n... (truncated; use `gormes kanban ...` in your terminal for full output)"
}
