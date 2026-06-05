package gateway

import (
	"context"

	"github.com/TrebuchetDynamics/gormes-agent/internal/gateway/kanbancmd"
)

func (m *Manager) handleKanbanCommand(ctx context.Context, ch Channel, ev InboundEvent) {
	output, err := runGatewayKanbanSlash(ctx, kanbancmd.Runner(m.cfg.KanbanSlashRunner), ev.Text)
	if err != nil {
		_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, "kanban error: "+sanitizeConfigReloadError(err))
		return
	}
	_, _ = m.sendWithHooksReply(ctx, ch, ev.ChatID, ev.MsgID, boundGatewayKanbanOutput(output))
}

func runGatewayKanbanSlash(ctx context.Context, runner kanbancmd.Runner, input string) (string, error) {
	return kanbancmd.RunSlash(ctx, runner, input)
}

func boundGatewayKanbanOutput(output string) string {
	return kanbancmd.BoundOutput(output)
}
