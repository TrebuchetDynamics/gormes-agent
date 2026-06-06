package gateway

import (
	"context"

	runtimegateway "github.com/TrebuchetDynamics/gormes-agent/internal/gateway"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
)

// NewKanbanSlashRunner adapts the shared CLI/TUI Kanban command runner for
// gateway /kanban dispatch. Keeping this seam in the gateway CLI module lets
// cmd/gormes pass build/exit-code options without owning the slash-command
// wiring details.
func NewKanbanSlashRunner(opts gormescli.KanbanCommandOptions) runtimegateway.KanbanSlashRunner {
	return func(ctx context.Context, input string) (string, error) {
		return gormescli.RunTUIKanbanSlashCommand(ctx, input, opts)
	}
}
