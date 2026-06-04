package main

import (
	"database/sql"
	"log/slog"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	channelgoncho "github.com/TrebuchetDynamics/gormes-agent/cmd/gormes/channelgoncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newChannelGonchoService(db *sql.DB, cfg gonchoservice.Config, log *slog.Logger, client llm.Client, model string) *gonchoservice.Service {
	return channelgoncho.NewService(db, cfg, log, client, model)
}

func registerChannelGonchoTools(reg *tools.Registry, svc *gonchoservice.Service) {
	channelgoncho.RegisterTools(reg, svc)
}

func registerGormesGonchoTools(reg *tools.Registry, mem *gormesgoncho.Runtime) {
	channelgoncho.RegisterGormesTools(reg, mem)
}

func formatGormesGonchoStatus(status gormesgoncho.Status) string {
	return channelgoncho.FormatStatus(status)
}
