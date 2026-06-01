package legacy

import (
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/adapters/channels/discord/legacy/runtime"
	"github.com/TrebuchetDynamics/gormes-agent/internal/kernel"
)

type Config = runtime.Config

type Bot = runtime.Bot

func New(cfg Config, client Client, k *kernel.Kernel, log *slog.Logger) *Bot {
	return runtime.New(cfg, client, k, log)
}
