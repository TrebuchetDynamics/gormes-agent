package main

import (
	"database/sql"
	"log/slog"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/gonchotools"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

// newChannelGonchoService constructs a goncho.Service wired for channel runtime
// use. It attaches a Hermes-backed DialecticCaller when a provider client is
// available, enabling in-process honcho_reasoning without an external Honcho
// process.
func newChannelGonchoService(db *sql.DB, cfg goncho.Config, log *slog.Logger, client hermes.Client, model string) *goncho.Service {
	svc := goncho.NewService(db, cfg, log)
	if client != nil {
		svc.SetDialecticCaller(NewHermesDialecticCaller(client, model))
	}
	return svc
}

// registerChannelGonchoTools wires honcho_* memory tools onto the agent registry
// backed by the given goncho service. This is the shared entry point all
// channels (Telegram, WhatsApp, Slack, Discord) call to enable memory.
func registerChannelGonchoTools(reg *tools.Registry, svc *goncho.Service) {
	gonchotools.RegisterHonchoTools(reg, svc)
}
