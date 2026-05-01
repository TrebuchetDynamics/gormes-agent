package main

import (
	"database/sql"
	"log/slog"

	"github.com/TrebuchetDynamics/gormes-agent/internal/goncho"
	"github.com/TrebuchetDynamics/gormes-agent/internal/hermes"
)

func newGonchoRuntimeService(db *sql.DB, cfg goncho.Config, log *slog.Logger, client hermes.Client, model string) *goncho.Service {
	svc := goncho.NewService(db, cfg, log)
	if client != nil {
		svc.SetDialecticCaller(goncho.NewHermesDialecticCaller(client, model))
	}
	return svc
}
