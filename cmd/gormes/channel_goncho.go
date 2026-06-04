package main

import (
	"database/sql"
	"log/slog"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/platform/cli/gormescli"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

func newChannelGonchoService(db *sql.DB, cfg gonchoservice.Config, log *slog.Logger, client llm.Client, model string) *gonchoservice.Service {
	return gormescli.NewChannelGonchoService(db, cfg, log, client, model)
}

func registerChannelGonchoTools(reg *tools.Registry, svc *gonchoservice.Service) {
	gormescli.RegisterChannelGonchoTools(reg, svc)
}

func registerGormesGonchoTools(reg *tools.Registry, mem *gormesgoncho.Runtime) {
	gormescli.RegisterGormesGonchoTools(reg, mem)
}

func formatGormesGonchoStatus(status gormesgoncho.Status) string {
	return gormescli.FormatGormesGonchoStatus(status)
}
