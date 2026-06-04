package gormescli

import (
	"database/sql"
	"log/slog"

	gormesgoncho "github.com/TrebuchetDynamics/goncho/integration/gormes"
	gonchoservice "github.com/TrebuchetDynamics/goncho/service"
	appchannelgoncho "github.com/TrebuchetDynamics/gormes-agent/internal/app/channelgoncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/llm"
	"github.com/TrebuchetDynamics/gormes-agent/internal/tools"
)

type HermesDialecticCaller = appchannelgoncho.HermesDialecticCaller

func NewChannelGonchoService(db *sql.DB, cfg gonchoservice.Config, log *slog.Logger, client llm.Client, model string) *gonchoservice.Service {
	return appchannelgoncho.NewService(db, cfg, log, client, model)
}

func RegisterChannelGonchoTools(reg *tools.Registry, svc *gonchoservice.Service) {
	appchannelgoncho.RegisterTools(reg, svc)
}

func RegisterGormesGonchoTools(reg *tools.Registry, mem *gormesgoncho.Runtime) {
	appchannelgoncho.RegisterGormesTools(reg, mem)
}

func FormatGormesGonchoStatus(status gormesgoncho.Status) string {
	return appchannelgoncho.FormatStatus(status)
}

func NewHermesDialecticCaller(client llm.Client, model string) *HermesDialecticCaller {
	return appchannelgoncho.NewHermesDialecticCaller(client, model)
}
